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

	"github.com/gaarutyunov/sysgo/examples/openapi/api"
)

// main starts the generated gin server backed by the hand-written Catalog
// handlers. Set ADDR to override the listen address (default :8080).
//
// It shuts down gracefully on SIGINT/SIGTERM and returns from main normally, so
// a binary built with `go build -cover` flushes coverage to GOCOVERDIR when the
// integration test stops the container.
func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	router := gin.Default()
	api.RegisterHandlers(router, NewCatalog())
	srv := &http.Server{Addr: addr, Handler: router}

	go func() {
		log.Printf("serving catalog API on %s", addr)
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
