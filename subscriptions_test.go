package goro_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/pat"
	uuid "github.com/satori/go.uuid"

	"github.com/dghubble/sling"
	"github.com/stretchr/testify/assert"
	"github.com/vectorhacker/goro"
)

func TestCatchupSubscription(t *testing.T) {
	generateEvents := func(count int) goro.Events {
		events := make(goro.Events, count)
		for i := range events {
			events[i] = goro.Event{
				ID:   uuid.NewV4(),
				Type: "deposit",
				Data: []byte("{\"double\":\"trouble\"}"),
			}
		}

		return events
	}

	t.Run("it should stream 3 events", func(t *testing.T) {
		called := false
		mux := pat.New()
		mux.Get("/streams/{stream}/{page}/{direction}/{count}", func(w http.ResponseWriter, r *http.Request) {
			if called {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"entries": goro.Events{},
				})
				w.WriteHeader(http.StatusOK)
				return
			}

			accept := r.Header.Get("Accept")
			stream := r.URL.Query().Get(":stream")

			direction := r.URL.Query().Get(":direction")
			count := r.URL.Query().Get(":count")
			embed := r.URL.Query().Get("embed")
			longPoll := r.Header.Get("ES-LongPoll")

			assert.Equal(t, "test", stream)

			assert.Equal(t, "forward", direction)
			assert.Equal(t, "10", count)
			assert.Equal(t, "application/vnd.eventstore.atom+json", accept)
			assert.Equal(t, "body", embed)
			assert.Equal(t, "10", longPoll)

			w.Header().Add("Content-Type", "application/vnd.eventstore.atom+json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"entries": generateEvents(3),
			})
			w.WriteHeader(http.StatusOK)
			called = true
		})
		s := httptest.NewServer(mux)

		subscription := goro.NewCatchupSubscription(goro.SlingerFunc(func() *sling.Sling {
			return sling.New().Base(s.URL).Client(s.Client()).New()
		}), "test", 0)

		events := goro.Events{}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		stream := subscription.Subscribe(ctx)
		defer cancel()
		for message := range stream {
			assert.Nil(t, message.Error)
			assert.NotNil(t, message.Event)
			events = append(events, message.Event)
		}

		t.Logf("%##v", events)
		assert.Len(t, events, 3)
	})
}
