// Package goro is pure Go client library for dealing with Event Store (versions 3.2.0 and later).
// It includes a high-level API for reading and writing events. Usage examples for the high-level
// APIs are provided inline with their full documentation.
package goro

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/dghubble/sling"
	"github.com/satori/go.uuid"
)

// Author represents the Author of an Event
type Author struct {
	Name string `json:"name"`
}

// Event represents an Event in Event Store
// the data and Metadata must be json encoded
type Event struct {
	At             time.Time       `json:"updated,omitempty"`
	Author         Author          `json:"author,omitempty"`
	Stream         string          `json:"streamId,omitempty"`
	Type           string          `json:"eventType"`
	PositionStream string          `json:"positionStreamId,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	ID             uuid.UUID       `json:"eventID"`
	Version        int64           `json:"eventNumber"`
	Position       int64           `json:"positionEventNumber,omitempty"`
}

func NewUUID() uuid.UUID {
	id, _ := uuid.NewV4()

	return id
}

// CreateEvent initializes a new Event with an event type, some data, metadata, and a version you
// specify. It then creates a random uuid and sets the time it was created at.
func CreateEvent(eventType string, data, metadata json.RawMessage, version int64) Event {
	return Event{
		ID:       NewUUID(),
		Type:     eventType,
		Data:     data,
		Metadata: metadata,
		Version:  version,
		At:       time.Now(),
	}
}

// Events is an array of events that implements the sort.Interface interface.
type Events []Event

// Len implements sort.Interface
func (e Events) Len() int {
	return len(e)
}

// Swap implements sort.Interface
func (e Events) Swap(a, b int) {
	e[b], e[a] = e[a], e[b]
}

// Less implements sort.Interface
func (e Events) Less(a, b int) bool {
	return e[a].Version < e[b].Version
}

// StreamMessage contains an Event or an error
type StreamMessage struct {
	Event        Event
	Acknowledger Acknowledger
	Error        error
}

// Ack acknowledges an Event or fails
func (m StreamMessage) Ack() error {
	if m.Acknowledger != nil {
		return m.Acknowledger.Ack()
	}

	return errors.New("no Acknowledger set")
}

// Nack rejects`` an Event or fails
func (m StreamMessage) Nack(action Action) error {
	if m.Acknowledger != nil {
		return m.Acknowledger.Nack(action)
	}

	return errors.New("no Acknowledger set")
}

// Action represents the action to take when Nacking an Event
type Action string

// Action enum
const (
	ActionPark  Action = "park"
	ActionRetry        = "retry"
	ActionSkip         = "skip"
	ActionStop         = "stop"
)

// Acknowledger can acknowledge or Not-Acknowledge an Event in a Persistent Subscription
type Acknowledger interface {
	Ack() error
	Nack(action Action) error
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
	Write(ctx context.Context, expectedVersion int64, events ...Event) error
}

// Reader reads a couple of Events from a stream
type Reader interface {
	Read(ctx context.Context, start int64, count int) (Events, error)
}

// Slinger is something that can return a sling object
type Slinger interface {
	Sling() *sling.Sling
}

// SlingerFunc is something that can return a sling object
type SlingerFunc func() *sling.Sling

// Sling implements the Slinger interface
func (f SlingerFunc) Sling() *sling.Sling {
	return f()
}

func relevantError(statusCode int) error {
	switch statusCode {
	case http.StatusNotFound:
		return ErrStreamNotFound
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusInternalServerError:
		return ErrInternalError
	case http.StatusBadRequest:
		return ErrInvalidContentType
	case http.StatusNotAcceptable:
		return ErrInvalidContentType
	default:
		return nil
	}
}
