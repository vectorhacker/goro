package main

import (
	"context"
	"fmt"
	"log"

	"github.com/vectorhacker/goro"
)

func main() {
	// create a client to use the Event Store
	client := goro.Connect("http://localhost:2113", goro.WithBasicAuth("admin", "changeit"))

	ctx := context.Background()
	reader := client.FowardsReader("messages")
	catchupSubscription := client.CatchupSubscription("messages", 0) // start from 0

	// subscribe to a stream of events
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		messages := catchupSubscription.Subscribe(ctx)

		for message := range messages {
			if err := message.Error; err != nil {
				return
			}

			log.Printf("Event: %s", message.Event.Data)
		}
	}()

	// read events
	events, err := reader.Read(ctx, 0, 1)
	if err != nil {
		panic(err)
	}

	for _, event := range events {
		fmt.Printf("%s\n", event.Data)
	}
}
