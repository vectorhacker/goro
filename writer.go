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
	stream string
	client Client
}

const (
	writePath = "/streams/%s"
)

// NewWriter creates a new Writer for a stream
func NewWriter(client Client, stream string) Writer {
	return &streamWriter{
		stream: stream,
		client: client,
	}
}

// Write implements the Writer interface. It writes events in a bulk after sorting them in version order
func (w streamWriter) Write(ctx context.Context, expectedVersion int64, events ...*Event) error {
	b := new(bytes.Buffer)

	path := fmt.Sprintf(writePath, w.stream)

	data := append(Events{}, events...)
	sort.Sort(data)

	json.NewEncoder(b).Encode(data)

	req, err := w.client.Request(ctx, http.MethodPost, path, b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/vnd.eventstore.events+json")
	req.Header.Set("ES-ExpectedVersion", fmt.Sprintf("%d", expectedVersion))

	resp, err := w.client.HTTPClient().Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return errors.New("not created with status" + resp.Status)
	}

	return nil
}
