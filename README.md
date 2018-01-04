Goro
====

Goro is a [Go](http://golang.org) client library for [Event Store](http://eventstore.org).

[Godoc](https://godoc.org/github.com/vectorhacker/goro)

Example:
----

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/vectorhacker/goro"
)

func main() {
    // create a client to use the Event Store
    client := goro.Connect("http://localhost:2113", goro.WithBasicAuth("admin", "changeit"))


    writer := client.Writer("messages")
    reader := client.FowardsReader("messages")
    catchupSubscription := client.CatchupSubscription("messages", 0) // start from 0

    data := []byte("{\"message\": \"hello world\"}")

    // write the event
    ctx := context.Background()
    event := goro.CreateEvent(
        "message",
        data,
        nil, // nil metadata
        0,
    )
    err = writer.Write(ctx, goro.ExpectedVersionAny, event)
    if err != nil {
        panic(err)
    }

    // subscribe to a stream of events
    go func() {
        ctx := context.Background()
        events := catchupSubscription.Subscribe(ctx)

        for _, event := range events {
            fmt.Printf("%s\n", event.Data)
        }
    }()

    // read events
    events, err := reader.Read(ctx, 0, 1)
    if err != nil {
        panic(err)
    }

    for _, event := range event {
        fmt.Printf("%s\n", event.Data)
    }
}
```

TODO
---

- [x] Tests
- [x] Competing Consumers
- [ ] Projections
- [x] Read Events
- [x] Stream Events
- [x] Write Events
- [ ] User Management