package goro

import (
	"context"
	"errors"
)

type catchupSubscription struct {
	stream string
	start  int64
	client Client
}

// NewCatchupSubscription creates a Subscriber that starts reading a stream from a specific event and then
// catches up to the head of the stream.
func NewCatchupSubscription(client Client, stream string, startFrom int64) Subscriber {
	return &catchupSubscription{
		stream: stream,
		start:  startFrom,
		client: client,
	}
}

// Subscribe implements the Subscriber interface
func (s *catchupSubscription) Subscribe(ctx context.Context) <-chan StreamMessage {
	stream := make(chan StreamMessage)
	return stream
}

type persistentSubscription struct {
	stream string
	group  string
	client Client
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
func NewPersistentSubscription(client Client, stream, group string, settings *PersistentSubscriptionSettings) (Subscriber, error) {
	return &persistentSubscription{
		client: client,
		group:  group,
		stream: stream,
	}, nil
}

// UpdatePersistentSubscription updates an existing subscription if it's persistant
func UpdatePersistentSubscription(subscription Subscriber, newSettings *PersistentSubscriptionSettings) (Subscriber, error) {
	_, ok := subscription.(*persistentSubscription)
	if !ok {
		return nil, errors.New("not a Persistant Subscription")
	}

	return subscription, nil
}

// Subscribe implements the Subscriber interface
func (s *persistentSubscription) Subscribe(ctx context.Context) <-chan StreamMessage {
	stream := make(chan StreamMessage)
	return stream
}
