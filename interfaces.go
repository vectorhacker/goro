package goro

import "context"

// Direction specifies the direction to point a stream at
type Direction string

// Direction enum
const (
	DirectionForward  Direction = "forward"
	DirectionBackward           = "backward"
)

// EventReader reads a single event from a stream
type EventReader interface {
	Read(ctx context.Context, stream string, version int64) (*Event, error)
}

// StreamingOptions sets the options for creating a stream of
// events from an Event Store stream
type StreamingOptions struct {
	Start int64
	Max   int
	Poll  bool // Specifies if long polling should be used
}

// EventStreamer reads many events into a stream
type EventStreamer interface {
	Stream(ctx context.Context, stream string, opts StreamingOptions) Stream
}

// EventWriter writes many events to a stream.
type EventWriter interface {
	Write(ctx context.Context, stream string, events ...*Event) error
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
