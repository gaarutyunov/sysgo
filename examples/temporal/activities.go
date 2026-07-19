package main

import (
	"context"
	"log"

	"github.com/gaarutyunov/sysgo/examples/temporal/orders"
)

// OrderActivities is the hand-written implementation of the generated
// orders.Activities port. The workflow calls these by name; the worker
// registers this value so Temporal can dispatch to them.
//
// Real activities would talk to a payment gateway, a mailer, a database, etc.
// Here they just log so the example stays dependency-free and deterministic.
type OrderActivities struct{}

// ChargeCard implements orders.Activities.
func (OrderActivities) ChargeCard(ctx context.Context, order orders.Order) error {
	log.Printf("ChargeCard: charging order %s", order.Id)
	return nil
}

// SendReceipt implements orders.Activities.
func (OrderActivities) SendReceipt(ctx context.Context, order orders.Order) error {
	log.Printf("SendReceipt: sending receipt for order %s", order.Id)
	return nil
}

// compile-time assertion that the hand-written type satisfies the generated port.
var _ orders.Activities = OrderActivities{}
