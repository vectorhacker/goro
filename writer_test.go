package goro_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dghubble/sling"
	"github.com/stretchr/testify/assert"
	"github.com/vectorhacker/goro"

	"github.com/gorilla/pat"
	"github.com/satori/go.uuid"
)

func TestWriter(t *testing.T) {
	t.Run("it should write succesfully", func(t *testing.T) {
		d1 := []byte("{\"key\":\"value\"}")

		evnt1 := &goro.Event{
			Data:    d1,
			Type:    "testevent",
			Version: 0,
			ID:      uuid.NewV4(),
		}
		evnt2 := &goro.Event{
			Data:    d1,
			Type:    "testevent",
			Version: 1,
			ID:      uuid.NewV4(),
		}

		mux := pat.New()
		mux.Post("/streams/{stream}", func(w http.ResponseWriter, r *http.Request) {
			stream := r.URL.Query().Get(":stream")
			assert.Equal(t, "test", stream)

			contentType := r.Header.Get("Content-Type")
			assert.Equal(t, "application/vnd.eventstore.events+json", contentType)
			events := goro.Events{}

			err := json.NewDecoder(r.Body).Decode(&events)
			assert.Nil(t, err)

			assert.Len(t, events, 2)

			w.WriteHeader(http.StatusCreated)
		})
		s := httptest.NewServer(mux)

		w := goro.NewWriter(goro.SlingerFunc(func() *sling.Sling {
			return sling.New().Base(s.URL).Client(s.Client())
		}), "test")

		ctx := context.Background()

		err := w.Write(ctx, goro.ExpectedVersionAny, evnt1, evnt2)
		assert.Nil(t, err)
	})
}
