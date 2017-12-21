package goro

import "context"

// EventReader reads a multiple event from a stream. If the count is -1, it will try to read all events starting
// from the version given. Practically, the count is limited to math.MaxInt32 or math.MaxInt64 depending on the
// system, if needed a long poll of 10 seconds will be used to read event up to the count, if using count -1 no long
// poll will be used
type EventReader interface {
	ReadForwards(ctx context.Context, version int64, count int) (Events, error)
	ReadBackwards(ctx context.Context, version int64, count int) (Events, error)
}

// EventStreamer reads many events into a stream.
type EventStreamer interface {
	Stream(ctx context.Context) Stream
}

// EventWriter writes many events to a stream.
type EventWriter interface {
	Write(ctx context.Context, events ...*Event) error
}

// EventReadWriter is a composite interface of EventReader
// and EventWriter
type EventReadWriter interface {
	EventReader
	EventWriter
}

// EventReadWriteStreamer is a composite interface of
// EventReadWriter and EventStreamer
type EventReadWriteStreamer interface {
	EventReadWriter
	EventStreamer
}

// Acknowledger notifies the server of successful or failed consumption of
// messages.
// Applications can provide mock implementations in tests of Delivery handlers.
// borrowed from: https://github.com/streadway/amqp/blob/master/delivery.go#L19
type Acknowledger interface {
	Ack() error
	Nack(requeue bool) error
}
