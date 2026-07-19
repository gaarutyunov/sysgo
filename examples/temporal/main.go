package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"

	"github.com/gaarutyunov/sysgo/examples/temporal/orders"
)

// main dials a Temporal server and runs the generated worker with the
// hand-written activity implementations. Point it at a server with
// TEMPORAL_HOSTPORT (defaults to the local dev server on 127.0.0.1:7233).
func main() {
	hostPort := os.Getenv("TEMPORAL_HOSTPORT")
	if hostPort == "" {
		hostPort = client.DefaultHostPort
	}

	c, err := client.Dial(client.Options{HostPort: hostPort})
	if err != nil {
		log.Fatalf("dial temporal at %s: %v", hostPort, err)
	}
	defer c.Close()

	log.Printf("worker connected to %s, task queue \"orders\"", hostPort)
	if err := orders.RunWorker(c, OrderActivities{}); err != nil {
		log.Fatalf("run worker: %v", err)
	}
}
