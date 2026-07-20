// Command worker is the Temporal entrypoint of the combined example. It is a
// thin composition root: it wires the in-memory repository into the PlaceOrder
// use case, hands the use case to the Temporal driving adapter, and runs the
// generated worker.
package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"

	adaptertemporal "github.com/gaarutyunov/sysgo/examples/combined/internal/order/adapter/in/temporal"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/adapter/out/repository"
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

	// Wire the hexagon: driven adapter -> use case -> driving adapter. The SAME
	// PlaceOrder use case the REST server drives.
	repo := repository.NewOrderRepositoryImpl()
	placeOrder := usecase.NewPlaceOrderInteractor(repo)
	activities := adaptertemporal.NewActivities(placeOrder)

	log.Printf("worker connected to %s, task queue \"orders\"", hostPort)
	// Register a *pointer* so RegisterActivity picks up the methods as activities.
	if err := orders.RunWorker(c, activities); err != nil {
		log.Fatalf("run worker: %v", err)
	}
}
