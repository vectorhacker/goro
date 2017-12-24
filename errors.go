package goro

import "errors"

// errors
var (
	ErrStreamNeverCreated = errors.New("stream never created")
	ErrInvalidContentType = errors.New("invalid content type")
)
