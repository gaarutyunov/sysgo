// Code scaffolded by sysgo; edit freely (not regenerated).

package usecase

import (
	"errors"

	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/port/in"
)

// PlaceOrderInteractor implements the PlaceOrder use case — the shared business
// logic that both the OpenAPI and Temporal entrypoints drive.
type PlaceOrderInteractor struct{}

var _ in.PlaceOrderUseCase = (*PlaceOrderInteractor)(nil)

// NewPlaceOrderInteractor constructs the interactor. Inject driven ports here.
func NewPlaceOrderInteractor() *PlaceOrderInteractor {
	return &PlaceOrderInteractor{}
}

// PlaceOrder accepts an order. This example keeps the logic trivial — it
// validates the order carries an id — to demonstrate the transport → use-case
// flow; a real interactor would orchestrate the domain and driven ports.
func (p *PlaceOrderInteractor) PlaceOrder(input in.PlaceOrderInput) (in.PlaceOrderOutput, error) {
	if input.Order.ID == "" {
		return in.PlaceOrderOutput{}, errors.New("order id is required")
	}
	return in.PlaceOrderOutput{}, nil
}
