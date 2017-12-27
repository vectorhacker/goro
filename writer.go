package goro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	req, err := w.slinger.
		Sling().
		Post(path).
		Body(b).
		Set("Content-Type", "application/vnd.eventstore.events+json").
		Set("ES-ExpectedVersion", fmt.Sprintf("%d", expectedVersion)).
		Request()
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	resp, err := w.slinger.Sling().Do(req, nil, nil)
	if err != nil {
		return err
	}

	return RelevantError(resp.StatusCode)
}
