package goro

import (
	"encoding/json"

	"github.com/satori/go.uuid"
)

// Event represent an event to be stored.
type Event struct {
	ID        uuid.UUID       `json:"eventId"`
	JSON      bool            `json:"isJson"`
	Data      json.RawMessage `json:"data"`
	Metadata  json.RawMessage `json:"metaData"`
	Stream    string          `json:"streamId"`
	Type      string          `json:"eventType"`
	Version   int64           `json:"eventNumber"`
	Timestamp string          `json:"updated"`
}

// Stream represents a stream of events
type Stream <-chan StreamEvent

type StreamEvent struct {
	Event *Event
	Err   error
}
