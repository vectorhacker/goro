package goro

import "errors"

// errors
var (
	ErrStreamNeverCreated = errors.New("stream never created")
	ErrInvalidContentType = errors.New("invalid content type")
	ErrStreamNotFound     = errors.New("the stream was not found")
	ErrUnauthorized       = errors.New("no access")
	ErrInternalError      = errors.New("internall error has occured")
)
