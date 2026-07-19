package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/gaarutyunov/sysgo/examples/openapi/api"
)

// Catalog is the hand-written implementation of the generated
// api.ServerInterface. It keeps the featured product in memory; a real service
// would use a database. The generated types (api.CatalogAPIProduct) are the
// request/response bodies, so the wire contract always matches the
// sysgo-generated schema.
type Catalog struct {
	mu       sync.Mutex
	featured api.CatalogAPIProduct
}

// NewCatalog returns a Catalog seeded with one product so GET
// /products/featured returns data out of the box.
func NewCatalog() *Catalog {
	return &Catalog{
		featured: api.CatalogAPIProduct{Id: "p-1", Name: "Widget", Price: 9.99},
	}
}

// CreateProduct implements POST /products: it stores the posted product as the
// featured one and echoes it back.
func (c *Catalog) CreateProduct(ctx *gin.Context) {
	var product api.CatalogAPIProduct
	if err := ctx.ShouldBindJSON(&product); err != nil {
		ctx.JSON(http.StatusBadRequest, api.ProblemDetails{
			Title:  strptr("invalid request body"),
			Status: intptr(http.StatusBadRequest),
			Detail: strptr(err.Error()),
		})
		return
	}
	c.mu.Lock()
	c.featured = product
	c.mu.Unlock()
	ctx.JSON(http.StatusCreated, product)
}

// GetFeaturedProduct implements GET /products/featured.
func (c *Catalog) GetFeaturedProduct(ctx *gin.Context) {
	c.mu.Lock()
	product := c.featured
	c.mu.Unlock()
	ctx.JSON(http.StatusOK, product)
}

// compile-time assertion that the hand-written type satisfies the generated port.
var _ api.ServerInterface = (*Catalog)(nil)

func strptr(s string) *string { return &s }
func intptr(i int) *int       { return &i }
