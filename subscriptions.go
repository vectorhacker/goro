package goro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Author is the author of an event or stream
type Author struct {
	Name string `json:"name"`
}

// StreamResult represents a stream response
type StreamResult struct {
	Events       Events    `json:"entries"`
	HeadOfStream bool      `json:"headOfStream"`
	Title        string    `json:"title"`
	Timestamp    time.Time `json:"updated"`
	Author       Author    `json:"author"`
}

// PersistentSubscriptionParameters parameters for creating or updating a PersistentSubscription
type PersistentSubscriptionParameters struct {
	ResolveToLinks              bool  `json:"resolveLinktos,omitempty"`
	Start                       int64 `json:"startFrom,omitempty"`
	ExtraStatistics             bool  `json:"extraStatistics,omitempty"`
	CheckPointAfterMilliseconds int   `json:"checkPointAfterMilliseconds,omitempty"`
	LiveBufferSize              int   `json:"liveBufferSize,omitempty"`
	ReadBatchSize               int   `json:"readBatchSize,omitempty"`
	BufferSize                  int   `json:"bufferSize,omitempty"`
	MaxCheckPointCount          int   `json:"maxCheckPointCount,omitempty"`
	MaxRetryCount               int   `json:"maxRetryCount,omitempty"`
	MaxSubscriberCount          int   `json:"maxSubscriberCount,omitempty"`
	MessageTimeoutMilliseconds  int   `json:"messageTimeoutMilliseconds,omitempty"`
	MinCheckPointCount          int   `json:"minCheckPointCount,omitempty"`
	NamedConsumerStrategy       int   `json:"namedConsumerStrategy,omitempty"`
}

// PersistentSubscriptionOption changes the options in a PersistentSubscription
type PersistentSubscriptionOption func(*PersistentSubscription)

// PersistentSubscriptionWithHost sets the host for the PersistentSubscription
func PersistentSubscriptionWithHost(host string) PersistentSubscriptionOption {
	return func(c *PersistentSubscription) {
		c.Host = host
	}
}

// PersistentSubscription represents a consuming competing consumer subscription group.
type PersistentSubscription struct {
	HTTPClient       *http.Client
	Host             string
	Credentials      Credentials
	StreamName       string
	SubscriptionName string
}

// CreatePersistentSubscription creates a subscription group
func CreatePersistentSubscription(stream, name string, parameters PersistentSubscriptionParameters, options ...PersistentSubscriptionOption) (*PersistentSubscription, error) {
	s := &PersistentSubscription{
		HTTPClient:       http.DefaultClient,
		Host:             DefaultHost,
		Credentials:      DefaultCredentials,
		SubscriptionName: name,
		StreamName:       stream,
	}

	for _, option := range options {
		option(s)
	}

	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(parameters)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create subsciption due to bad parameters")
	}

	uri := fmt.Sprintf("%s/subscriptions/%s/%s", s.Host, stream, name)

	req, err := http.NewRequest(http.MethodPut, uri, b)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create subsciption")
	}
	req.Header.Add("Content-Type", "application/json")
	s.Credentials.Apply(req)

	res, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create subsciption")
	}

	if res.StatusCode != http.StatusCreated {
		return nil, errors.Errorf("unable to create subsciption due to status %d %s", res.StatusCode, res.Status)
	}

	return s, nil
}

// Update updates the subscription
func (s *PersistentSubscription) Update(parameter PersistentSubscriptionParameters) error {
	return nil
}

// Stream implemetns the EventStreamer interface
func (s *PersistentSubscription) Stream(ctx context.Context) Stream {
	stream := make(Stream)
	uri := fmt.Sprintf("%s/subscriptions/%s/%s/10", s.Host, s.StreamName, s.SubscriptionName)

	go func() {
		defer close(stream)

		for {
			result, err := retreiveEvents(ctx, uri, s.Credentials, s.HTTPClient)
			if err != nil {
				stream <- StreamMessage{
					Error: err,
				}
				return

			}

			for _, event := range result.Events {
				select {
				case stream <- StreamMessage{
					Event:        event,
					Acknowledger: nil, // TODO: implements Acknowledger
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return stream
}

// CatchupSubscription starts at a suggested point continues up to the head of the stream
type CatchupSubscription struct {
	StreamName  string
	HTTPClient  *http.Client
	Host        string
	Credentials Credentials
	Start       int64
}

// Stream implements the EventStreamer interface
func (s *CatchupSubscription) Stream(ctx context.Context) Stream {
	stream := make(Stream)
	next := s.Start

	go func() {
		for {
			uri := fmt.Sprintf("%s/streams/%s/%d/forward/10", s.Host, s.StreamName, next)

			result, err := retreiveEvents(ctx, uri, s.Credentials, s.HTTPClient)
			if err != nil {
				stream <- StreamMessage{
					Error: err,
				}
				return

			}

			for _, event := range result.Events {
				select {
				case stream <- StreamMessage{
					Event:        event,
					Acknowledger: nil, // TODO: implements Acknowledger
				}:

				case <-ctx.Done():
					return
				}
			}

			if len(result.Events) == 0 {
				<-time.After(10 * time.Second)
				continue
			}

			next += int64(len(result.Events))
		}
	}()

	return stream
}

func retreiveEvents(ctx context.Context, uri string, credentials Credentials, client *http.Client) (*StreamResult, error) {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read stream")
	}
	credentials.Apply(req)
	req.Header.Add("Accept", "application/vnd.eventstore.atom+json")

	q := req.URL.Query()
	q.Add("embed", "body")
	req.URL.RawQuery = q.Encode()

	req = req.WithContext(ctx)

	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read stream")
	}

	result := StreamResult{}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read stream")
	}

	return &result, nil
}
