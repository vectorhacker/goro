package goro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/pkg/errors"
)

type Reader struct {
	Stream      string
	Credentials Credentials
	HTTPClient  *http.Client
	Host        string
}

func (r *Reader) ReadForwards(ctx context.Context, version int64, count int) (Events, error) {
	events := Events{}

	headOfStream := false
	next := version

	for len(events) != count || (count == -1 && !headOfStream) {
		uri := fmt.Sprintf("%s/streams/%d/forward/20", r.Host, r.Stream, next)
		result, err := retreiveEvents(ctx, uri, r.Credentials, r.HTTPClient)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		headOfStream = result.HeadOfStream
		events = append(events, result.Events...)

		next += int64(len(result.Events))
	}

	sort.Sort(events)

	return events, nil
}

func (r *Reader) ReadBackwards(ctx context.Context, version int64, count int) (Events, error) {
	events := Events{}

	next := version

	for len(events) != count || (count == -1 && next > 0) {
		uri := fmt.Sprintf("%s/streams/%d/forward/20", r.Host, r.Stream, next)
		result, err := retreiveEvents(ctx, uri, r.Credentials, r.HTTPClient)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}

		events = append(events, result.Events...)

		next -= int64(len(result.Events))
	}

	return events, nil
}

// Writer implements the EventWriter interface
type Writer struct {
	Stream      string
	Credentials Credentials
	HTTPClient  *http.Client
	Host        string
}

// Write implements the EventWriter interface
func (w *Writer) Write(ctx context.Context, events ...*Event) error {
	evs := append(Events{}, events...)

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(evs)
	uri := fmt.Sprintf("%s/streams/%s", w.Host, w.Stream)
	req, err := http.NewRequest(http.MethodPost, uri, b)
	if err != nil {
		return errors.Wrap(err, "unable to save events")
	}

	w.Credentials.Apply(req)
	req.Header.Add("Content-Type", "application/json")

	res, err := w.HTTPClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "unable to save events")
	}

	if res.StatusCode != http.StatusCreated {
		return errors.Errorf("unable to save events due to status %s", res.Status)
	}

	return nil
}
