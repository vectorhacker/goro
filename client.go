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
	ID     string              `json:"streamId"`
	Events []*Event            `json:"entries"`
	Links  []map[string]string `json:"links"`
}

type Client struct {
	host        string
	headers     map[string]string
	credentials *basicAuthCredentials
	http        *http.Client
}

type ConnectionOption func(client *Client)

// WithHostAndPort sets the host and port to use
func WithHost(host string) ConnectionOption {
	return func(c *Client) {
		c.host = host
	}
}

// WithBasicAuth sets the basic authentication for calling Event Store
func WithBasicAuth(username, password string) ConnectionOption {
	return func(c *Client) {
		c.credentials = &basicAuthCredentials{
			username,
			password,
		}
	}
}

// WithHTTPClient sets the http.Client to use in the Event Store client
func WithHTTPClient(client *http.Client) ConnectionOption {
	return func(c *Client) {
		c.http = client
	}
}

// Connect creates a new Client
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

// Read implements the EventReader interface
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

// Write implements the EventWriter interface
func (c *Client) Write(ctx context.Context, stream string, events ...Event) error {

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
