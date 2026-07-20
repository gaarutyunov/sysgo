// Code scaffolded by sysgo; edit freely (not regenerated).

package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gaarutyunov/sysgo/examples/combined/api"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/port/in"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/domain"
)

// PlaceOrderUseCaseHandler is the REST driving adapter. It implements the
// sysgo-generated api.ServerInterface and translates HTTP requests into calls on
// the PlaceOrder use case — REST is just an entrypoint over the business logic.
type PlaceOrderUseCaseHandler struct {
	uc in.PlaceOrderUseCase
}

var _ api.ServerInterface = (*PlaceOrderUseCaseHandler)(nil)

// NewPlaceOrderUseCaseHandler wires the driving adapter to a PlaceOrder use case.
func NewPlaceOrderUseCaseHandler(uc in.PlaceOrderUseCase) *PlaceOrderUseCaseHandler {
	return &PlaceOrderUseCaseHandler{uc: uc}
}

// PlaceOrderAPI implements POST /orders: decode the order and drive the use case.
func (h *PlaceOrderUseCaseHandler) PlaceOrderAPI(c *gin.Context) {
	var body api.OrderContextOrder
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, problem("invalid request body", err.Error()))
		return
	}
	if _, err := h.uc.PlaceOrder(in.PlaceOrderInput{Order: toDomain(body)}); err != nil {
		c.JSON(http.StatusBadRequest, problem("could not place order", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, body)
}

func toDomain(o api.OrderContextOrder) domain.Order {
	return domain.Order{
		ID:    o.Id,
		Total: domain.Money{Amount: float64(o.Total.Amount), Currency: o.Total.Currency},
		Lines: []domain.LineItem{{Sku: o.Lines.Sku, Qty: int64(o.Lines.Qty)}},
	}
}

func problem(title, detail string) api.ProblemDetails {
	return api.ProblemDetails{Title: &title, Detail: &detail}
}
