package goro

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/dghubble/sling"
	"github.com/satori/go.uuid"
)

const (
	readCount       = 10   // 10 events
	longPollTimeout = "10" // 10 seconds
)

type embedParams struct {
	Embed string `url:"embed,omitempty"`
}

type catchupSubscription struct {
	stream  string
	start   int64
	slinger Slinger
}

// NewCatchupSubscription creates a Subscriber that starts reading a stream from a specific event and then
// catches up to the head of the stream.
func NewCatchupSubscription(slinger Slinger, stream string, startFrom int64) Subscriber {
	return &catchupSubscription{
		stream:  stream,
		start:   startFrom,
		slinger: slinger,
	}
}

// Subscribe implements the Subscriber interface
func (s *catchupSubscription) Subscribe(ctx context.Context) <-chan StreamMessage {
	stream := make(chan StreamMessage)
	response := struct {
		Events Events `json:"entries"`
	}{}

	go func() {
		defer close(stream)
		next := s.start

		for {
			path := fmt.Sprintf("/streams/%s/%d/forward/%d", s.stream, next, readCount)
			res, err := s.slinger.
				Sling().
				Get(path).
				Add("Accept", "application/vnd.eventstore.atom+json").
				Set("ES-LongPoll", longPollTimeout).
				QueryStruct(&embedParams{
					Embed: "body",
				}).
				ReceiveSuccess(&response)
			if err != nil {
				stream <- StreamMessage{
					Error: err,
				}
				return
			}

			err = relevantError(res.StatusCode)
			if err != nil {
				stream <- StreamMessage{
					Error: err,
				}
				return
			}

			sort.Sort(response.Events)
			for _, event := range response.Events {
				select {
				case <-ctx.Done():
					return
				case stream <- StreamMessage{
					Event: event,
				}:
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
				next += int64(len(response.Events))
			}
		}
	}()

	return stream
}

type persistentSubscription struct {
	stream           string
	subscriptionName string
	slinger          Slinger
}

// PersistentSubscriptionSettings represents the settings for creating and updating a Persistent subscription.
// You can read more about those settings in the Event Store documentation [here](https://eventstore.org/docs/http-api/4.0.2/competing-consumers/#creating-a-persistent-subscription).
type PersistentSubscriptionSettings struct {
	ResolveLinkTos              bool   `json:"resolveLinktos,omitempty"`
	StartFrom                   int64  `json:"startFrom,omitempty"`
	ExtraStatistics             bool   `json:"extraStatistics,omitempty"`
	CheckPointAfterMilliseconds int64  `json:"checkPointAfterMilliseconds,omitempty"`
	LiveBufferSize              int    `json:"liveBufferSize,omitempty"`
	ReadBatchSize               int    `json:"readBatchSize,omitempty"`
	BufferSize                  int    `json:"bufferSize,omitempty"`
	MaxCheckPointCount          int    `json:"maxCheckPointCount,omitempty"`
	MaxRetryCount               int    `json:"maxRetryCount,omitempty"`
	MaxSubscriberCount          int    `json:"maxSubscriberCount,omitempty"`
	MessageTimeoutMilliseconds  int64  `json:"messageTimeoutMilliseconds,omitempty"`
	MinCheckPointCount          int    `json:"minCheckPointCount,omitempty"`
	NamedConsumerStrategy       string `json:"namedConsumerStrategy,omitempty"`
}

// NewPersistentSubscription creates a new subscription that implements the competing consumers pattern
func NewPersistentSubscription(slinger Slinger, stream, subscriptionName string, settings PersistentSubscriptionSettings) (Subscriber, error) {
	s := &persistentSubscription{
		slinger:          slinger,
		subscriptionName: subscriptionName,
		stream:           stream,
	}

	res, err := s.slinger.
		Sling().
		Put(fmt.Sprintf("/subscriptions/%s/%s", stream, subscriptionName)).
		BodyJSON(settings).
		ReceiveSuccess(nil)
	if err != nil {
		return nil, err
	}

	return s, relevantError(res.StatusCode)
}

// UpdatePersistentSubscription updates an existing subscription if it's Persistent
func UpdatePersistentSubscription(subscription Subscriber, newSettings PersistentSubscriptionSettings) (Subscriber, error) {
	s, ok := subscription.(*persistentSubscription)
	if !ok {
		return nil, errors.New("not a Persistent Subscription")
	}

	res, err := s.slinger.
		Sling().
		Post(fmt.Sprintf("/subscriptions/%s/%s", s.stream, s.subscriptionName)).
		BodyJSON(newSettings).
		ReceiveSuccess(nil)
	if err != nil {
		return nil, err
	}

	return subscription, relevantError(res.StatusCode)
}

// Subscribe implements the Subscriber interface
func (s *persistentSubscription) Subscribe(ctx context.Context) <-chan StreamMessage {
	stream := make(chan StreamMessage)

	response := struct {
		Events Events `json:"entries"`
	}{}

	go func() {
		defer close(stream)

		for {
			path := fmt.Sprintf("/subscriptions/%s/%s/%d", s.stream, s.subscriptionName, readCount)
			res, err := s.slinger.
				Sling().
				Get(path).
				Add("Accept", "application/vnd.eventstore.atom+json").
				QueryStruct(&embedParams{
					Embed: "body",
				}).
				ReceiveSuccess(&response)
			if err != nil {
				stream <- StreamMessage{
					Error: err,
				}
				return
			}

			err = relevantError(res.StatusCode)
			if err != nil {
				stream <- StreamMessage{
					Error: err,
				}
				return
			}

			sort.Sort(response.Events)
			for _, event := range response.Events {
				select {
				case <-ctx.Done():
					return
				case stream <- StreamMessage{
					Event: event,
					Acknowledger: persistentSubscriptionAcknowledger{
						eventID:          event.ID,
						stream:           s.stream,
						subscriptionName: s.subscriptionName,
						sling:            s.slinger.Sling(),
					},
				}:
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return stream
}

type persistentSubscriptionAcknowledger struct {
	eventID          uuid.UUID
	stream           string
	subscriptionName string
	sling            *sling.Sling
}

func (a persistentSubscriptionAcknowledger) path() string {
	path := "/subscriptions/{stream}/{subscription_name}"
	path = strings.Replace(path, "{stream}", a.stream, -1)
	path = strings.Replace(path, "{subscription_name}", a.subscriptionName, -1)
	return path
}

func (a persistentSubscriptionAcknowledger) Ack() error {
	path := a.path()
	res, err := a.sling.Post(path + "/ack/" + a.eventID.String()).ReceiveSuccess(nil)

	if err != nil {
		return err
	}

	return relevantError(res.StatusCode)
}

func (a persistentSubscriptionAcknowledger) Nack(action Action) error {
	path := a.path()
	res, err := a.sling.Post(path + "/nack/" + a.eventID.String()).QueryStruct(struct {
		Action Action `url:"action"`
	}{
		Action: action,
	}).ReceiveSuccess(nil)

	if err != nil {
		return err
	}

	return relevantError(res.StatusCode)
}
