package goro

import (
	"time"

	"github.com/satori/go.uuid"
)

// RawData implements the TextMarshaler and TextUnmarshaler interfaces
// in order to preserve the bytes given. For all intents an purposes you
// can treat this like []byte
type RawData []byte

// MarshalText implements the TextMashaler interface
func (r RawData) MarshalText() (text []byte, err error) {
	return r[:], nil
}

// UnmarshalText impelements the TextUnmarshaler interface
func (r *RawData) UnmarshalText(text []byte) error {
	*r = text[:]

	return nil
}

// Event represent an event to be stored.
type Event struct {
	ID        uuid.UUID `json:"eventId"`
	IsJSON    bool      `json:"isJson"`
	Data      RawData   `json:"data,omitempty"`
	Metadata  RawData   `json:"metaData,omitempty"`
	Stream    string    `json:"streamId"`
	Type      string    `json:"eventType"`
	Version   int64     `json:"eventNumber"`
	Timestamp time.Time `json:"updated"`
}

// Events is an array of events. It impelements the sort.Interface interface
type Events []Event

// Len implements the sort.Interface interface
func (e Events) Len() int {
	return len(e)
}

// Less implements the sort.Interface interface
func (e Events) Less(i, j int) bool {

	return e[i].Version < e[j].Version
}

// Swap implements the sort.Interface interface
func (e Events) Swap(i, j int) {
	temp := e[i]
	e[i] = e[j]
	e[j] = temp
}

// Stream represents a stream of events
type Stream <-chan StreamEvent

// StreamEvent represents an event as part of a stream. It contains the Event
// or an error if parsing an event failed
type StreamEvent struct {
	Event *Event
	Err   error
}
