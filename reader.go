package goro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type direction string

const (
	directionForwards  direction = "foward"
	directionBackwards           = "bacward"
)

type streamReader struct {
	stream    string
	direction direction
	client    Client
}

// NewBackwardsReader creates a Reader that reads events backwards
func NewBackwardsReader(client Client, stream string) Reader {
	return &streamReader{
		stream:    stream,
		direction: directionBackwards,
		client:    client,
	}
}

// NewForwardsReader creates a Reader that reads events forwards
func NewForwardsReader(client Client, stream string) Reader {
	return &streamReader{
		stream:    stream,
		direction: directionForwards,
		client:    client,
	}
}

func (r streamReader) Read(ctx context.Context, start int64, count int) ([]*Event, error) {
	events := Events{}
	response := struct {
		Events Events `json:"entries"`
	}{}

	next := start
	if r.direction == directionBackwards {
		next++
	}

	for len(events) != count {
		path := fmt.Sprintf("/streams/%s/%d/%s/10", r.stream, next, r.direction)
		req, err := r.client.Request(ctx, http.MethodGet, path, nil)
		if err != nil {
			// TODO: enrich error
			return nil, err
		}
		req.Header.Add("Accept", "application/json")

		res, err := r.client.HTTPClient().Do(req)
		if err != nil {
			// TODO: enrich error
			return nil, err
		}

		err = json.NewDecoder(res.Body).Decode(&response)
		if err != nil {
			// TODO: enrich error
			return nil, err
		}

		events = append(events, response.Events...)

		switch r.direction {
		case directionBackwards:
			next -= int64(len(events))
		case directionForwards:
			next += int64(len(events))
		}
	}
	return nil, nil
}
