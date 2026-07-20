// Command api is the OpenAPI (REST) entrypoint of the combined example. It runs
// the sysgo-generated gin server, backed by handlers that drive the DDD
// PlaceOrder use case. Set ADDR to override the listen address (default :8080).
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
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/usecase"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	router := gin.Default()
	api.RegisterHandlers(router, NewServer(usecase.NewPlaceOrderInteractor()))
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
