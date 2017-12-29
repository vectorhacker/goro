package goro_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vectorhacker/goro"
)

func TestClient(t *testing.T) {
	t.Run("it should return a valid client", func(t *testing.T) {
		c := goro.Connect("http://localhost:2113")

		assert.NotNil(t, c)
	})

	t.Run("it should return a valid sling.Sling object", func(t *testing.T) {
		c := goro.Connect("http://localhost:2113")

		assert.NotNil(t, c.Sling())
	})

	t.Run("it should return a reader", func(t *testing.T) {
		c := goro.Connect("http://localhost:2113")

		assert.NotNil(t, c.BackwardsReader("tests"))
	})

	t.Run("it should return a writer", func(t *testing.T) {
		c := goro.Connect("http://localhost:2113")
		var writer goro.Writer = c.Writer("tests")

		assert.NotNil(t, writer)
	})

	t.Run("it should return a CatchupSubscription", func(t *testing.T) {
		c := goro.Connect("http://localhost:2113")
		var subscription goro.Subscriber = c.CatchupSubscription("testing", 0)

		assert.NotNil(t, subscription)
	})

	t.Run("it should return a PersistantSubscription", func(t *testing.T) {
		mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})
		s := httptest.NewServer(mux)
		c := goro.Connect(s.URL, goro.WithHTTPClient(s.Client()))
		var subscription goro.Subscriber
		var err error
		subscription, err = c.PersistentSubscription("testing", "test", goro.PersistentSubscriptionSettings{})

		assert.Nil(t, err)
		assert.NotNil(t, subscription)
	})
}
