package goro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"github.com/pkg/errors"
)

type basicAuthCredentials struct {
	Username string
	Password string
}

type streamResponse struct {
	ID     string `json:"streamId"`
	Events Events `json:"entries"`
}

// Client represents a client connection to an Event Store server.
// It implements the EventReadWriteStreamer interface. The Client uses
// http to connect to EventStore.
type Client struct {
	host        string
	headers     map[string]string
	credentials *basicAuthCredentials
	http        *http.Client
}

// ConnectionOption represents the different options to add to a client
type ConnectionOption func(client *Client)

// WithHost sets the host to use
func WithHost(host string) ConnectionOption {
	return func(c *Client) {
		c.host = host
	}
}

// WithBasicAuth sets the basic authentication for calling Event Store.
func WithBasicAuth(username, password string) ConnectionOption {
	return func(c *Client) {
		c.credentials = &basicAuthCredentials{
			username,
			password,
		}
	}
}

// WithHTTPClient sets a custom http.Client to use for calling the EventStore.
func WithHTTPClient(client *http.Client) ConnectionOption {
	return func(c *Client) {
		c.http = client
	}
}

// Connect creates a new Client with the options set. It does not start a persistent connection,
// since it uses the EventStore HTTP interface.
func Connect(opts ...ConnectionOption) *Client {

	client := &Client{
		http: http.DefaultClient,
		host: "127.0.0.1:2113",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

func (c *Client) hostPort() string {
	return fmt.Sprintf("%s", c.host)
}

// addRequestOptions adds additional request options
func (c *Client) addRequestOptions(req *http.Request) {
	if c.credentials != nil {
		req.SetBasicAuth(c.credentials.Username, c.credentials.Password)
	}

	for header, value := range c.headers {
		req.Header.Add(header, value)
	}
}

// Read implements the EventReader interface. It reads a single event from the store
func (c *Client) Read(ctx context.Context, stream string, version int64) (*Event, error) {
	eventc := make(chan *Event)
	errc := make(chan error, 1)

	go func() {
		var event *Event

		uri := fmt.Sprintf("%s/streams/%s/%d/forward/1", c.hostPort(), stream, version)
		req, err := http.NewRequest(http.MethodGet, uri, nil)
		if err != nil {
			errc <- errors.Wrap(err, "Unable to create request to database")
			return
		}

		// add context for cancelation of request
		req = req.WithContext(ctx)

		q := url.Values{}
		q.Add("embed", "body")
		req.URL.RawQuery = q.Encode()
		req.Header.Add("Accept", "application/json")

		c.addRequestOptions(req)

		resp, err := c.http.Do(req)
		if err != nil {
			errc <- errors.Wrap(err, "Calling datbase failed")
			return
		}

		s := &streamResponse{}
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(s)
		if err != nil {
			errc <- errors.WithStack(errors.Wrap(err, "Failed to decode stream"))
			return
		}

		if len(s.Events) > 0 {
			event = s.Events[0]
		}

		select {
		case eventc <- event:
		case <-ctx.Done():
			return
		}
	}()

	select {
	case err := <-errc:
		return nil, errors.Wrap(err, "Unexpected error when reading event")
	case event := <-eventc:
		return event, nil
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "Context closed.")
	}
}

// Write implements the EventWriter interface. It Write multiple events in batch. It will sort
// the events given to it using the Version Number and sets the `ES-ExpectedVersion` header equal
// to the Version of the first Event minus 1 i.e: `events[0].Version - 1`. It expects that the events
// have different versions.
func (c *Client) Write(ctx context.Context, stream string, events ...*Event) error {

	evs := Events{}
	if len(events) == 0 {
		return errors.New("can't save no events")
	}

	for _, ev := range events {
		evs = append(evs, ev)
	}

	sort.Sort(evs)

	errc := make(chan error)
	go func() {

		uri := fmt.Sprintf("%s/streams/%s", c.hostPort(), stream)
		b := new(bytes.Buffer)

		encoder := json.NewEncoder(b)
		err := encoder.Encode(evs)
		if err != nil {
			errc <- errors.Wrap(err, "unable to encode request")
		}

		req, err := http.NewRequest(http.MethodPost, uri, b)
		if err != nil {
			errc <- errors.Wrap(err, "unable to make request")
			return
		}
		req = req.WithContext(ctx)
		req.Header.Add("Content-Type", "application/vnd.eventstore.events+json")
		req.Header.Add("ES-ExpectedVersion", fmt.Sprintf("%d", evs[0].Version-1))

		c.addRequestOptions(req)

		resp, err := c.http.Do(req)
		if err != nil {
			errc <- errors.Wrap(err, "unable to make request")
			return
		}

		if resp.StatusCode != http.StatusCreated {
			errc <- errors.New("Unable to create")
		}

		select {
		case errc <- nil:
		case <-ctx.Done():
			return
		}
	}()

	select {
	case err := <-errc:
		if err != nil {
			return err
		}
	case <-ctx.Done():
	}

	return nil
}

// Stream implements the EventStreamer interfaace. It wills tream multiple events. To stream backwards from the head,
// use the start option of -1
func (c *Client) Stream(ctx context.Context, stream string, opts StreamingOptions) Stream {

	count := 20
	s := make(Stream)

	go func() {
		defer close(s)

		// start the next stream from the start
		next := opts.Start

		for {
			uri := fmt.Sprintf("%s/streams/%s/%d/%s/%d", c.hostPort(), stream, next, DirectionForward, count)

			req, err := http.NewRequest(http.MethodGet, uri, nil)
			if err != nil {
				s <- StreamEvent{
					Err: err,
				}
			}

			req = req.WithContext(ctx)

			c.addRequestOptions(req)

			if opts.Poll {
				// LongPoll for 10 seconds if no results yet.
				req.Header.Add("ES-LongPoll", "10")
			}

			q := req.URL.Query()
			q.Add("embed", "body")
			req.URL.RawQuery = q.Encode()

			res, err := c.http.Do(req)
			if err != nil {
				s <- StreamEvent{
					Err: err,
				}
			}

			_stream := streamResponse{}

			d := json.NewDecoder(res.Body)
			err = d.Decode(&_stream)
			if err != nil {
				s <- StreamEvent{
					Err: err,
				}
			}

			sort.Sort(_stream.Events)

			for i, event := range _stream.Events {
				if opts.Max > 0 {
					if i+int(next)+1 > opts.Max {
						return
					}
				}

				ev := StreamEvent{
					Event: event,
					Err:   nil,
				}

				select {
				case s <- ev:
				case <-ctx.Done():
					s <- StreamEvent{
						Err: errors.Wrap(ctx.Err(), "context stoped"),
					}
					return
				}
			}

			next += int64(len(_stream.Events))
		}
	}()

	return s
}
