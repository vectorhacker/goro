package goro_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vectorhacker/goro"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Test struct {
	Test string `protobuf:"bytes,1,opt,name=Test" json:"Test,omitempty"`
}

func (m *Test) Reset()                    { *m = Test{} }
func (m *Test) String() string            { return proto.CompactTextString(m) }
func (*Test) ProtoMessage()               {}
func (*Test) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Test) GetTest() string {
	if m != nil {
		return m.Test
	}
	return ""
}

func init() {
	proto.RegisterType((*Test)(nil), "goro.Test")
}

func init() { proto.RegisterFile("buf_test.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 71 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x4b, 0x2a, 0x4d, 0x8b,
	0x2f, 0x49, 0x2d, 0x2e, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x49, 0xcf, 0x2f, 0xca,
	0x57, 0x92, 0xe2, 0x62, 0x09, 0x49, 0x2d, 0x2e, 0x11, 0x12, 0x82, 0xd0, 0x12, 0x8c, 0x0a, 0x8c,
	0x1a, 0x9c, 0x41, 0x60, 0x76, 0x12, 0x1b, 0x58, 0xa1, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff, 0x67,
	0xaa, 0x1c, 0x3d, 0x3a, 0x00, 0x00, 0x00,
}

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

	d, err := proto.Marshal(&Test{
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

func TestStream(t *testing.T) {
	assert := assert.New(t)

	client := goro.Connect(goro.WithHost("http://localhost:2113"), goro.WithBasicAuth("admin", "changeit"))
	ctx := context.Background()

	ctx, _ = context.WithTimeout(ctx, 5*time.Second)

	stream := client.Stream(ctx, "$projections-$master", goro.StreamingOptions{
		Max: 50,
	})

	expectedSum := 1225
	sum := 0

	for stremEvent := range stream {
		assert.Nil(stremEvent.Err)
		t.Log(stremEvent.Event.ID, stremEvent.Event.Version)
		sum += int(stremEvent.Event.Version)
	}

	assert.Equal(expectedSum, sum)
}
