package goro

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client interface {
	Request(ctx context.Context, method, path string, body io.Reader) (*http.Request, error)
	HTTPClient() *http.Client
}

type client struct {
	host       string
	httpClient *http.Client
}

type ClientOption func(*client)

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

func (c client) Request(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	uri := fmt.Sprintf("%s/%s", c.host, url.PathEscape(path))
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c client) HTTPClient() *http.Client {
	return c.httpClient
}
