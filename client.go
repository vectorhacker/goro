package goro

import (
	"net/http"

	"github.com/dghubble/sling"
)

// Client is a connection to an event store
type Client struct {
	sling *sling.Sling
}

// ClientOption applies options to a client
type ClientOption func(*Client)

// WithBasicAuth adds basic authentication to the Event Store
func WithBasicAuth(username, password string) ClientOption {
	return func(c *Client) {
		c.sling.SetBasicAuth(username, password)
	}
}

// WithHTTPClient sets the http.Client for the Client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.sling.Client(httpClient)
	}
}

// Connect creates a new client
func Connect(host string, options ...ClientOption) *Client {
	c := &Client{
		sling: sling.New().Base(host),
	}
	for _, opt := range options {
		opt(c)
	}

	return c
}

// Sling creates a new Sling object
func (c Client) Sling() *sling.Sling {
	return c.sling.New()
}

// Writer creates a new Writer for a stream
func (c Client) Writer(stream string) Writer {
	return NewWriter(c, stream)
}

// BackwardsReader creates a new Reader that reads backwards on a stream
func (c Client) BackwardsReader(stream string) Reader {
	return NewBackwardsReader(c, stream)
}

// FowardsReader creates a new Reader that reads forwards on a stream
func (c Client) FowardsReader(stream string) Reader {
	return NewForwardsReader(c, stream)
}
