package goro_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/vectorhacker/goro"
)

func TestReadEvent(t *testing.T) {
	eventstore := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		f, _ := os.Open("./test.json")
		io.Copy(w, f)
		w.WriteHeader(http.StatusOK)
	}))
	defer eventstore.Close()

	client := goro.Connect(goro.WithHost(eventstore.URL), goro.WithHTTPClient(eventstore.Client()))

	ctx := context.Background()
	event, err := client.Read(ctx, "test", 10)

	if err != nil {
		t.Fatal(err)
	}

	if event == nil {
		t.Fatalf("Expeted event not to be nil, got %#v", event)
	}

}
