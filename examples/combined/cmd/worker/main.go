// Command worker is the Temporal entrypoint of the combined example. It runs the
// sysgo-generated worker, whose activities drive the DDD PlaceOrder use case.
// Point it at a server with TEMPORAL_HOSTPORT (defaults to 127.0.0.1:7233).
package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"

	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/usecase"
	"github.com/gaarutyunov/sysgo/examples/combined/orders"
)

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
	// Register a *pointer* so RegisterActivity picks up the methods as activities.
	if err := orders.RunWorker(c, NewOrderActivities(usecase.NewPlaceOrderInteractor())); err != nil {
		log.Fatalf("run worker: %v", err)
	}
}
