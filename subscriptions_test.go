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

func TestPersistentSubscription(t *testing.T) {
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

	t.Run("it should read and acknowledge three events", func(t *testing.T) {

		var calledCreate, calledFetch, calledAck bool

		getParameters := func(r *http.Request) (stream string, subscriptionName string) {
			stream = r.URL.Query().Get(":stream")
			subscriptionName = r.URL.Query().Get(":subscription_name")
			return
		}

		mux := pat.New()
		mux.Put("/subscriptions/{stream}/{subscription_name}", func(w http.ResponseWriter, r *http.Request) {
			stream, subscriptionName := getParameters(r)
			assert.Equal(t, "test", stream)
			assert.Equal(t, "testing", subscriptionName)

			w.WriteHeader(http.StatusCreated)
			calledCreate = true
		})

		mux.Get("/subscriptions/{stream}/{subscription_name}/{count}", func(w http.ResponseWriter, r *http.Request) {
			stream, subscriptionName := getParameters(r)
			assert.Equal(t, "test", stream)
			assert.Equal(t, "testing", subscriptionName)

			json.NewEncoder(w).Encode(map[string]interface{}{
				"entries": generateEvents(3),
			})
			w.WriteHeader(http.StatusOK)
			calledFetch = true
		})

		mux.Post("/subscriptions/{stream}/{subscription_name}/ack/{messageid}", func(w http.ResponseWriter, r *http.Request) {
			stream, subscriptionName := getParameters(r)
			assert.Equal(t, "test", stream)
			assert.Equal(t, "testing", subscriptionName)
			w.WriteHeader(http.StatusOK)
			calledAck = true
		})
		s := httptest.NewServer(mux)

		slinger := goro.SlingerFunc(func() *sling.Sling {
			return sling.New().Base(s.URL).Client(s.Client()).New()
		})

		subscription, err := goro.NewPersistentSubscription(slinger, "test", "testing", goro.PersistentSubscriptionSettings{})
		assert.Nil(t, err)
		assert.NotNil(t, subscription)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stream := subscription.Subscribe(ctx)

		events := goro.Events{}
		for message := range stream {
			assert.Nil(t, message.Error)
			err := message.Ack()
			assert.Nil(t, err)
			events = append(events, message.Event)
		}

		assert.True(t, calledAck)
		assert.True(t, calledCreate)
		assert.True(t, calledFetch)
	})
}
