// Package temporal is the Temporal driving adapter: it implements the generated
// orders.Activities port and drives the PlaceOrder use case — Temporal is just
// another entrypoint over the same business logic the REST adapter drives.
package temporal

import (
	"context"

	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/port/in"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/domain"
	"github.com/gaarutyunov/sysgo/examples/combined/orders"
)

// Activities implements the generated orders.Activities port.
type Activities struct {
	uc in.PlaceOrderUseCase
}

var _ orders.Activities = (*Activities)(nil)

// NewActivities wires the driving adapter to a PlaceOrder use case.
func NewActivities(uc in.PlaceOrderUseCase) *Activities {
	return &Activities{uc: uc}
}

// ChargeCard drives the PlaceOrder use case for the order — the same call the
// REST adapter makes.
func (a *Activities) ChargeCard(_ context.Context, order orders.Order) error {
	_, err := a.uc.PlaceOrder(in.PlaceOrderInput{Order: domain.Order{ID: order.Id}})
	return err
}

// SendReceipt is a no-op side effect here, kept simple and deterministic.
func (a *Activities) SendReceipt(_ context.Context, _ orders.Order) error {
	return nil
}
