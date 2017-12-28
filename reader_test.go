package goro_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/satori/go.uuid"

	"github.com/dghubble/sling"

	"github.com/vectorhacker/goro"

	"github.com/gorilla/pat"
	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {

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

	t.Run("it should read forwards 2 pages", func(t *testing.T) {
		mux := pat.New()
		mux.Get("/streams/{stream}/{start}/{direction}/{pageSize}", func(w http.ResponseWriter, r *http.Request) {
			stream := r.URL.Query().Get(":stream")
			start, err := strconv.Atoi(r.URL.Query().Get(":start"))
			direction := r.URL.Query().Get(":direction")
			assert.Nil(t, err)

			assert.Equal(t, "forward", direction)
			assert.Equal(t, "test", stream)
			assert.True(t, start == 0 || start == 10)

			err = json.NewEncoder(w).Encode(map[string]interface{}{
				"entries": generateEvents(10),
			})
			assert.Nil(t, err)
		})
		s := httptest.NewServer(mux)

		r := goro.NewForwardsReader(goro.SlingerFunc(func() *sling.Sling {
			return sling.New().Base(s.URL).Client(s.Client()).New()
		}), "test")

		events, err := r.Read(context.Background(), 0, 20)
		assert.Nil(t, err)
		assert.Len(t, events, 20)
	})

	t.Run("it should read backwards 2 pages", func(t *testing.T) {
		mux := pat.New()
		mux.Get("/streams/{stream}/{start}/{direction}/{pageSize}", func(w http.ResponseWriter, r *http.Request) {
			stream := r.URL.Query().Get(":stream")
			start, err := strconv.Atoi(r.URL.Query().Get(":start"))
			direction := r.URL.Query().Get(":direction")
			assert.Nil(t, err)

			assert.Equal(t, "backward", direction)
			assert.Equal(t, "test", stream)
			assert.True(t, start == 21 || start == 11)

			err = json.NewEncoder(w).Encode(map[string]interface{}{
				"entries": generateEvents(10),
			})
			assert.Nil(t, err)
		})
		s := httptest.NewServer(mux)

		r := goro.NewBackwardsReader(goro.SlingerFunc(func() *sling.Sling {
			return sling.New().Base(s.URL).Client(s.Client()).New()
		}), "test")

		events, err := r.Read(context.Background(), 20, 20)
		assert.Nil(t, err)
		assert.Len(t, events, 20)
	})
}
