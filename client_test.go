package goro_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/golang/protobuf/proto"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vectorhacker/goro"
)

func TestReadEvent(t *testing.T) {
	eventstore := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `
			{
				"title": "Event stream 'test'",
				"id": "http://localhost:2113/streams/test",
				"updated": "2017-12-14T05:09:58.816079Z",
				"streamId": "test",
				"author": {
					"name": "EventStore"
				},
				"headOfStream": true,
				"links": [
					{
						"uri": "http://localhost:2113/streams/test",
						"relation": "self"
					},
					{
						"uri": "http://localhost:2113/streams/test/head/backward/1",
						"relation": "first"
					},
					{
						"uri": "http://localhost:2113/streams/test/1/forward/1",
						"relation": "previous"
					},
					{
						"uri": "http://localhost:2113/streams/test/metadata",
						"relation": "metadata"
					}
				],
				"entries": [
					{
						"eventId": "fbf4a1a1-b4a3-4dfe-a01f-ec52c34e16e4",
						"eventType": "event-type",
						"eventNumber": 0,
						"data": "{\n  \"a\": \"1\"\n}",
						"metaData": "{\n  \"yes\": \"no\"\n}",
						"streamId": "test",
						"isJson": true,
						"isMetaData": true,
						"isLinkMetaData": false,
						"positionEventNumber": 0,
						"positionStreamId": "test",
						"title": "0@test",
						"id": "http://localhost:2113/streams/test/0",
						"updated": "2017-12-14T05:09:58.816079Z",
						"author": {
							"name": "EventStore"
						},
						"summary": "event-type",
						"links": [
							{
								"uri": "http://localhost:2113/streams/test/0",
								"relation": "edit"
							},
							{
								"uri": "http://localhost:2113/streams/test/0",
								"relation": "alternate"
							}
						]
					}
				]
			}
			`)
		w.WriteHeader(http.StatusOK)

		reg := regexp.MustCompile("(\\/streams\\/test\\/[0-9]+\\/forward\\/1)")
		if !reg.Match([]byte(r.URL.Path)) {
			t.Fatalf("Expected path %s to match regex %s", r.URL.Path, reg)
		}

		if embed := r.URL.Query().Get("embed"); embed != "body" {
			t.Fatalf("Expected embed to be \"body\" got %#v", embed)
		}

		if accept := r.Header.Get("Accept"); accept != "application/json" {
			t.Fatalf("Expected Accept header to be application/json got %#v", accept)
		}
	}))
	defer eventstore.Close()

	client := goro.Connect(goro.WithHost(eventstore.URL), goro.WithHTTPClient(eventstore.Client()))

	ctx := context.Background()
	event, err := client.Read(ctx, "test", 0)

	if err != nil {
		t.Fatal(err)
	}

	if event == nil {
		t.Fatalf("Expeted event not to be nil, got %#v", event)
	}

	ctx, cancel := context.WithCancel(ctx)
	cancel()
	event, err = client.Read(ctx, "test", 0)
	if err == nil {
		t.Fatalf("Expected error to not be nil")
	}

	if event != nil {
		t.Fatalf("Expected event to be nil, got %#v", event)
	}
}

func TestWriteEvent(t *testing.T) {
	assert := assert.New(t)

	d, err := proto.Marshal(&goro.Test{
		Test: "testing",
	})
	assert.Nil(err)

	events := goro.Events{
		{
			ID:       uuid.NewV4(),
			Version:  0,
			Data:     []byte("{\"a\":\"1\"}"),
			Metadata: []byte("{\"a\":\"1\"}"),
			Type:     "DataAdded",
		},
		{
			ID:      uuid.NewV4(),
			Version: 1,
			Data:    []byte("{\"a\":\"1\"}"),
		},
		{
			ID:      uuid.NewV4(),
			Version: 1,
			Data:    d,
			Type:    "DataAdded",
		},
	}

	eventstore := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Headers",
			"Content-Type, X-Requested-With, X-Forwarded-Host, X-Forwarded-Prefix, X-PINGOTHER, Authorization, ES-LongPoll, ES-ExpectedVersion, ES-EventId, ES-EventType, ES-RequiresMaster, ES-HardDelete, ES-ResolveLinkTos",
		)

		w.Header().Add("Access-Control-Allow-Methods", "POST, DELETE, GET, OPTIONS")
		w.Header().Add("Access-Control-Expose-Headers", "Location, ES-Position, ES-CurrentVersion")

		sentEvents := goro.Events{}

		d := json.NewDecoder(r.Body)

		err := d.Decode(&sentEvents)
		assert.Nil(err)

		assert.Equal(events, sentEvents)

		assert.Equal("-1", r.Header.Get("ES-ExpectedVersion"))

		w.WriteHeader(http.StatusCreated)

	}))
	defer eventstore.Close()

	client := goro.Connect(goro.WithHost(eventstore.URL), goro.WithHTTPClient(eventstore.Client()))
	ctx := context.Background()

	err = client.Write(ctx, "test", events...)
	assert.Nil(err)
}
