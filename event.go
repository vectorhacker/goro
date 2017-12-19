package goro

import (
	"time"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

var (
	errNoAcknowledger = errors.New("")
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

// Event represent an event to be stored or retrieved. It stores Data and Metadata as byte arrays.
// It is up to the client to unmarshal the Data ane Metadata of the the event.
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
type Events []*Event

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
	e[i], e[j] = e[j], e[i]
}

// Stream represents a stream of events. This is used when streaming events from the Event Store
type Stream chan StreamMessage

// StreamMessage represents an event as part of a stream. It contains the Event
// or an error if parsing an event failed
type StreamMessage struct {
	Acknowledger Acknowledger // the channel from which this delivery arrived
	Event        *Event
	Error        error
}

/*
Ack delegates an acknowledgement through the Acknowledger interface that the
client or server has finished work on a delivery.
An error will indicate that the acknowledge could not be delivered to the
channel it was sent from.
Either Delivery.Ack, Delivery.Reject or Delivery.Nack must be called for every
delivery that is not automatically acknowledged.
*/
func (s StreamMessage) Ack() error {
	if s.Acknowledger == nil {
		return errNoAcknowledger
	}
	return s.Acknowledger.Ack()
}

/*
Reject delegates a negatively acknowledgement through the Acknowledger interface.
When requeue is true, queue this message to be delivered to a consumer on a
different channel.  When requeue is false or the server is unable to queue this
message, it will be dropped.
If you are batch processing deliveries, and your server supports it, prefer
Delivery.Nack.
Either Delivery.Ack, Delivery.Reject or Delivery.Nack must be called for every
delivery that is not automatically acknowledged.
*/
func (s StreamMessage) Reject(requeue bool) error {
	if s.Acknowledger == nil {
		return errNoAcknowledger
	}
	return s.Acknowledger.Reject(requeue)
}

/*
Nack negatively acknowledge the delivery of message(s) identified by the
delivery tag from either the client or server.
When requeue is true, request the server to deliver this message to a different
consumer.  If it is not possible or requeue is false, the message will be
dropped or delivered to a server configured dead-letter queue.
This method must not be used to select or requeue messages the client wishes
not to handle, rather it is to inform the server that the client is incapable
of handling this message at this time.
Either Delivery.Ack, Delivery.Reject or Delivery.Nack must be called for every
delivery that is not automatically acknowledged.
*/
func (s StreamMessage) Nack(requeue bool) error {
	if s.Acknowledger == nil {
		return errNoAcknowledger
	}
	return s.Acknowledger.Nack(requeue)
}
