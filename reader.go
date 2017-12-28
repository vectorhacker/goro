package goro

import (
	"context"
	"fmt"
)

type direction string

const (
	directionForwards  direction = "forward"
	directionBackwards           = "backward"
)

type streamReader struct {
	stream    string
	direction direction
	slinger   Slinger
}

// NewBackwardsReader creates a Reader that reads events backwards
func NewBackwardsReader(slinger Slinger, stream string) Reader {
	return &streamReader{
		stream:    stream,
		direction: directionBackwards,
		slinger:   slinger,
	}
}

// NewForwardsReader creates a Reader that reads events forwards
func NewForwardsReader(slinger Slinger, stream string) Reader {
	return &streamReader{
		stream:    stream,
		direction: directionForwards,
		slinger:   slinger,
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
		req, err := r.slinger.
			Sling().
			Get(path).
			Set("Accept", "application/vnd.eventstore.events+json").
			QueryStruct(embedParams{
				Embed: "body",
			}).
			Request()
		if err != nil {
			return nil, err
		}

		req = req.WithContext(ctx)

		res, err := r.slinger.Sling().Do(req, &response, nil)
		if err != nil {
			return nil, err
		}

		err = RelevantError(res.StatusCode)
		if err != nil {
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
	return events, nil
}
