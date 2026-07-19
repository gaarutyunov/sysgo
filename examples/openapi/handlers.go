package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/gaarutyunov/sysgo/examples/openapi/api"
)

// Catalog is the hand-written implementation of the generated
// api.ServerInterface. It keeps products in memory; a real service would use a
// database. The generated types (api.CatalogAPIProduct) are the request/response
// bodies, so the wire contract always matches the sysgo-generated schema.
type Catalog struct {
	mu       sync.Mutex
	products []api.CatalogAPIProduct
}

// NewCatalog returns a Catalog seeded with one product so GET /products returns
// data out of the box.
func NewCatalog() *Catalog {
	return &Catalog{
		products: []api.CatalogAPIProduct{
			{Id: "p-1", Name: "Widget", Price: 9.99},
		},
	}
}

// ListProducts implements GET /products.
func (c *Catalog) ListProducts(ctx *gin.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx.JSON(http.StatusOK, c.products)
}

// CreateProduct implements POST /products.
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
	c.products = append(c.products, product)
	c.mu.Unlock()
	ctx.JSON(http.StatusCreated, product)
}

// compile-time assertion that the hand-written type satisfies the generated port.
var _ api.ServerInterface = (*Catalog)(nil)

func strptr(s string) *string { return &s }
func intptr(i int) *int       { return &i }
