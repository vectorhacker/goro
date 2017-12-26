package goro

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// requestApplier applies some option to an http.Request
type requestApplier func(*http.Request)

// Client is a connection to an event store
type Client interface {
	Request(ctx context.Context, method, path string, body io.Reader) (*http.Request, error)
	HTTPClient() *http.Client
}

type client struct {
	host       string
	httpClient *http.Client
	appliers   []requestApplier
}

// ClientOption applies options to a client
type ClientOption func(*client)

// WithBasicAuth adds basic authentication to the Event Store
func WithBasicAuth(username, password string) ClientOption {
	return func(c *client) {
		c.appliers = append(c.appliers, func(r *http.Request) {
			r.SetBasicAuth(username, password)
		})
	}
}

// WithHTTPClient sets the http.Client for the Client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *client) {
		c.httpClient = httpClient
	}
}

// Connect creates a new client
func Connect(host string, options ...ClientOption) Client {
	c := &client{
		host:       host,
		httpClient: http.DefaultClient,
	}
	for _, opt := range options {
		opt(c)
	}
	return c
}

// Request implements the Client interface
func (c client) Request(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	if path[0] == '/' {
		path = path[1:]
	}
	uri := fmt.Sprintf("%s/%s", c.host, path)
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	for _, a := range c.appliers {
		a(req)
	}

	return req, nil
}

// HTTPClient implements the Client interface
func (c client) HTTPClient() *http.Client {
	return c.httpClient
}
