// Code scaffolded by sysgo; edit freely (not regenerated).

package usecase

import (
	"errors"

	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/port/in"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/port/out"
)

// PlaceOrderInteractor implements the PlaceOrder use case — the single piece of
// business logic that both the REST and Temporal driving adapters invoke. It
// depends only on ports (here the OrderRepository driven port), never on any
// transport.
type PlaceOrderInteractor struct {
	orders out.OrderRepository
}

var _ in.PlaceOrderUseCase = (*PlaceOrderInteractor)(nil)

// NewPlaceOrderInteractor injects the driven OrderRepository port.
func NewPlaceOrderInteractor(orders out.OrderRepository) *PlaceOrderInteractor {
	return &PlaceOrderInteractor{orders: orders}
}

// PlaceOrder validates and persists the order. The same code runs whether the
// caller arrived over HTTP or Temporal.
func (p *PlaceOrderInteractor) PlaceOrder(input in.PlaceOrderInput) (in.PlaceOrderOutput, error) {
	if input.Order.ID == "" {
		return in.PlaceOrderOutput{}, errors.New("order id is required")
	}
	if err := p.orders.OrderRepository(input.Order); err != nil {
		return in.PlaceOrderOutput{}, err
	}
	return in.PlaceOrderOutput{}, nil
}
