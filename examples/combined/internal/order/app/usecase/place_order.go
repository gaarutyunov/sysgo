// Code scaffolded by sysgo; edit freely (not regenerated).

package usecase

import (
	"errors"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/port/in"
)

// PlaceOrderInteractor implements the PlaceOrder use case. This scaffold is
// written once; add orchestration logic here. sysgo will not overwrite it.
type PlaceOrderInteractor struct{}

var _ in.PlaceOrderUseCase = (*PlaceOrderInteractor)(nil)

// NewPlaceOrderInteractor constructs the interactor. Inject driven ports here.
func NewPlaceOrderInteractor() *PlaceOrderInteractor {
	return &PlaceOrderInteractor{}
}

func (p *PlaceOrderInteractor) PlaceOrder(input in.PlaceOrderInput) (in.PlaceOrderOutput, error) {
	return in.PlaceOrderOutput{}, errors.New("not implemented")
}
