package goro

import "context"

type direction int

const (
	directionForwards direction = iota
	directionBackwards
)

type streamReader struct {
	stream    string
	direction direction
	client    Client
}

func NewBackwardsReader(client Client, stream string) Reader {
	return &streamReader{
		stream:    stream,
		direction: directionBackwards,
		client:    client,
	}
}

func NewForwardsReader(client Client, stream string) Reader {
	return &streamReader{
		stream:    stream,
		direction: directionForwards,
		client:    client,
	}
}

func (r streamReader) Read(ctx context.Context, start int64, count int) ([]*Event, error) {
	return nil, nil
}
