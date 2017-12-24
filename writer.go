package goro

import "context"

type streamWriter struct {
	stream string
	client Client
}

func NewWriter(client Client, stream string) Writer {
	return &streamWriter{
		stream: stream,
		client: client,
	}
}

func (w streamWriter) Write(ctx context.Context, expectedVersion int64, events ...*Event) error {
	return nil
}
