package goro

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
)

type streamWriter struct {
	stream  string
	slinger Slinger
}

const (
	writePath = "/streams/%s"
)

// NewWriter creates a new Writer for a stream
func NewWriter(slinger Slinger, stream string) Writer {
	return &streamWriter{
		stream:  stream,
		slinger: slinger,
	}
}

// Write implements the Writer interface. It writes events in a bulk after sorting them in version order
func (w streamWriter) Write(ctx context.Context, expectedVersion int64, events ...*Event) error {
	b := new(bytes.Buffer)

	path := fmt.Sprintf(writePath, w.stream)

	data := append(Events{}, events...)
	sort.Sort(data)

	json.NewEncoder(b).Encode(data)

	resp, err := w.slinger.
		Sling().
		Post(path).
		Body(b).
		Set("Content-Type", "application/vnd.eventstore.events+json").
		Set("ES-ExpectedVersion", fmt.Sprintf("%d", expectedVersion)).
		ReceiveSuccess(nil)

	if resp.StatusCode != http.StatusCreated {
		return errors.New("not created with status" + resp.Status)
	}

	return nil
}
