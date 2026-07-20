package main

import (
	"context"

	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/port/in"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/domain"
	"github.com/gaarutyunov/sysgo/examples/combined/orders"
)

// OrderActivities implements the generated orders.Activities port by driving the
// DDD PlaceOrder use case. The Temporal worker is a thin transport over the same
// business logic the REST API drives.
type OrderActivities struct {
	placeOrder in.PlaceOrderUseCase
}

// NewOrderActivities wires the activities to a PlaceOrder use case.
func NewOrderActivities(uc in.PlaceOrderUseCase) *OrderActivities {
	return &OrderActivities{placeOrder: uc}
}

// ChargeCard drives the PlaceOrder use case for the order.
func (a *OrderActivities) ChargeCard(_ context.Context, order orders.Order) error {
	_, err := a.placeOrder.PlaceOrder(in.PlaceOrderInput{Order: domain.Order{ID: order.Id}})
	return err
}

// SendReceipt is a no-op side effect here, kept simple and deterministic.
func (a *OrderActivities) SendReceipt(_ context.Context, _ orders.Order) error {
	return nil
}

// A *pointer* is what gets registered with the worker, so assert on the pointer.
var _ orders.Activities = (*OrderActivities)(nil)
