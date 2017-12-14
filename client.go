package goro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

		url := fmt.Sprintf("%s/streams/%s/%d/forward/1", c.hostPort(), stream, version)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			errc <- errors.Wrap(err, "Unable to create request to database")
			return
		}

		req.URL.Query().Add("embed", "body")
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
