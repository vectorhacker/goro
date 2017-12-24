// Package goro is pure Go client library for dealing with Event Store (versions 3.2.0 and later).
// It includes a high-level API for reading and writing events. Usage examples for the high-level
// APIs are provided inline with their full documentation.
package goro

import (
	"context"
	"time"

	"github.com/satori/go.uuid"
)

// RawBytes represents a byte array that can be turned into text and back again
// This is only to prevent the json decoder and econder from trying to parse them
// into base64 byte arrays
type RawBytes []byte

// MarshalText implements the TextMarshaler interface
func (r RawBytes) MarshalText() (text []byte, err error) {
	text = r[:]
	return
}

// UnmarshalText implements the TextUnamarshaler interface
func (r *RawBytes) UnmarshalText(text []byte) error {
	*r = text[:]
	return nil
}

// Author represents the Author of an Event
type Author struct {
	Name string `json:"name"`
}

// Event represents an Event in Event Store
type Event struct {
	ID             uuid.UUID `json:"eventID"`
	Type           string    `json:"eventType"`
	Version        int64     `json:"eventNumber"`
	Data           RawBytes  `json:"data,omitempty"`
	Stream         string    `json:"streamId"`
	IsJSON         bool      `json:"isJson"`
	Metadata       RawBytes  `json:"metadata,omitempty"`
	Position       int64     `json:"positionEventNumber,omitempty"`
	PositionStream string    `json:"positionStreamId,omitempty"`
	Title          string    `json:"title,omitempty"`
	At             time.Time `json:"updated,omitempty"`
	Author         Author    `json:"author,omitempty"`
	Summary        string    `json:"summary,omitempty"`
}

// StreamMessage contains an Event or an error
type StreamMessage struct {
	Event *Event
	Error error
}

// ExpectedVersions
const (
	ExpectedVersionAny   int64 = -2
	ExpectedVersionNone  int64 = -1
	ExpectedVersionEmpty int64 = 0
)

// Subscriber streams events
type Subscriber interface {
	Subscribe(ctx context.Context) <-chan StreamMessage
}

// Writer writes events to a stream
type Writer interface {
	Write(ctx context.Context, expectedVersion int64, events ...*Event) error
}

// Reader reads a couple of Events from a stream
type Reader interface {
	Read(ctx context.Context, start int64, count int) ([]*Event, error)
}
