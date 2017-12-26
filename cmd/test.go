package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/satori/go.uuid"

	"github.com/vectorhacker/goro"
)

func main() {
	c := goro.Connect("http://localhost:2113", goro.WithBasicAuth("admin", "changeit"))

	w := goro.NewWriter(c, "testing")

	d, _ := json.Marshal(map[string]string{
		"key": "value",
	})

	err := w.Write(context.Background(), -2, &goro.Event{
		ID:      uuid.NewV4(),
		Data:    d,
		Version: 0,
	})
	if err != nil {
		log.Fatal(err)
	}
}
