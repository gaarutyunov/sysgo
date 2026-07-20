// Command api is the REST entrypoint of the combined example. It is a thin
// composition root: it wires the in-memory repository into the PlaceOrder use
// case, hands the use case to the HTTP driving adapter, and hosts it on gin.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gaarutyunov/sysgo/examples/combined/api"
	adapterhttp "github.com/gaarutyunov/sysgo/examples/combined/internal/order/adapter/in/http"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/adapter/out/repository"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/usecase"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// Wire the hexagon: driven adapter -> use case -> driving adapter.
	repo := repository.NewOrderRepositoryImpl()
	placeOrder := usecase.NewPlaceOrderInteractor(repo)
	handler := adapterhttp.NewPlaceOrderUseCaseHandler(placeOrder)

	router := gin.Default()
	api.RegisterHandlers(router, handler)
	srv := &http.Server{Addr: addr, Handler: router}

	go func() {
		log.Printf("serving combined order API on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Printf("server stopped")
}
