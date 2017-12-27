package goro

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
		count := 20

		for {
			path := fmt.Sprintf("/streams/%s/%d/forwards/%d", s.stream, next, count)
			req, err := s.slinger.
				Sling().
				Get(path).
				Set("ES-LongPoll", "10").
				QueryStruct(&embedParams{
					Embed: "body",
				}).
				Request()
			if err != nil {
				stream <- StreamMessage{
					Error: err,
				}
				return
			}

			res, err := s.slinger.Sling().Do(req, &response, nil)
			if err != nil {
				stream <- StreamMessage{
					Error: err,
				}
				return
			}

			err = RelevantError(res.StatusCode)
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

			next += int64(len(response.Events))
			response.Events = nil
		}
	}()

	return stream
}

type persistentSubscription struct {
	stream  string
	group   string
	slinger Slinger
}

// PersistentSubscriptionSettings represents the settings for creating and updating a persistant subscripton.
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
func NewPersistentSubscription(slinger Slinger, stream, group string, settings *PersistentSubscriptionSettings) (Subscriber, error) {
	s := &persistentSubscription{
		slinger: slinger,
		group:   group,
		stream:  stream,
	}

	res, err := s.slinger.
		Sling().
		Post(fmt.Sprintf("/subscriptions/%s/%s", stream, group)).
		BodyJSON(settings).
		ReceiveSuccess(nil)
	if err != nil {
		return nil, err
	}

	return s, RelevantError(res.StatusCode)
}

// UpdatePersistentSubscription updates an existing subscription if it's persistant
func UpdatePersistentSubscription(subscription Subscriber, newSettings *PersistentSubscriptionSettings) (Subscriber, error) {
	s, ok := subscription.(*persistentSubscription)
	if !ok {
		return nil, errors.New("not a Persistant Subscription")
	}

	res, err := s.slinger.
		Sling().
		Post(fmt.Sprintf("/subscriptions/%s/%s", s.stream, s.group)).
		BodyJSON(newSettings).
		ReceiveSuccess(nil)
	if err != nil {
		return nil, err
	}

	return subscription, RelevantError(res.StatusCode)
}

// Subscribe implements the Subscriber interface
func (s *persistentSubscription) Subscribe(ctx context.Context) <-chan StreamMessage {
	stream := make(chan StreamMessage)
	return stream
}
