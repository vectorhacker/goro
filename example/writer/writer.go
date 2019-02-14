package main

import (
	"context"

	"github.com/vectorhacker/goro"
)

func main() {
	// create a client to use the Event Store
	client := goro.Connect("http://localhost:2113", goro.WithBasicAuth("admin", "changeit"))

	writer := client.Writer("messages")
	data := []byte("{\"message\": \"hello world\"}")

	// write the event
	ctx := context.Background()
	event := goro.CreateEvent(
		"message",
		data,
		nil, // nil metadata
		0,
	)
	err := writer.Write(ctx, goro.ExpectedVersionAny, event)
	if err != nil {
		panic(err)
	}
}
