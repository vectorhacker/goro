// Package goro is a client for Event Store written in Go.
package goro

import "net/http"

// Credentials adds user credentials
type Credentials interface {
	Apply(req *http.Request)
}

// BasicCredentials represents basic auth credentials
type BasicCredentials struct {
	Username string
	Password string
}

// Apply implements the Credentials interfae
func (c BasicCredentials) Apply(req *http.Request) {
	req.SetBasicAuth(c.Username, c.Password)
}

var (
	DefaultCredentials = BasicCredentials{
		Username: "admin",
		Password: "changeit",
	}

	DefaultHost = "http://127.0.0.1:2113"
)
