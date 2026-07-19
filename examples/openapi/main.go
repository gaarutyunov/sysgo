package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/gaarutyunov/sysgo/examples/openapi/api"
)

// main starts the generated gin server backed by the hand-written Catalog
// handlers. Set ADDR to override the listen address (default :8080).
func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	router := gin.Default()
	api.RegisterHandlers(router, NewCatalog())

	log.Printf("serving catalog API on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
