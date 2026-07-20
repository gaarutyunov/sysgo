package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gaarutyunov/sysgo/examples/combined/api"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/port/in"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/domain"
)

// Server implements the sysgo-generated api.ServerInterface by delegating to the
// DDD PlaceOrder use case. The REST layer is a thin transport: it decodes the
// request into the domain type and drives the business logic — the same use case
// the Temporal worker also drives.
type Server struct {
	placeOrder in.PlaceOrderUseCase
}

// NewServer wires the REST handlers to a PlaceOrder use case.
func NewServer(uc in.PlaceOrderUseCase) *Server { return &Server{placeOrder: uc} }

var _ api.ServerInterface = (*Server)(nil)

// PlaceOrderAPI implements POST /orders: decode the order, drive the PlaceOrder
// use case, and echo the accepted order.
func (s *Server) PlaceOrderAPI(c *gin.Context) {
	var body api.OrderContextOrder
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, problem("invalid request body", err.Error()))
		return
	}
	if _, err := s.placeOrder.PlaceOrder(in.PlaceOrderInput{Order: toDomainOrder(body)}); err != nil {
		c.JSON(http.StatusBadRequest, problem("could not place order", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, body)
}

// toDomainOrder maps the generated wire type to the domain aggregate.
func toDomainOrder(o api.OrderContextOrder) domain.Order {
	return domain.Order{
		ID:    o.Id,
		Total: domain.Money{Amount: float64(o.Total.Amount), Currency: o.Total.Currency},
		Lines: []domain.LineItem{{Sku: o.Lines.Sku, Qty: int64(o.Lines.Qty)}},
	}
}

func problem(title, detail string) api.ProblemDetails {
	return api.ProblemDetails{Title: &title, Detail: &detail}
}
