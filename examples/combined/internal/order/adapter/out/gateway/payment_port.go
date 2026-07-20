// Code scaffolded by sysgo; edit freely (not regenerated).

package gateway

import (
	"errors"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/port/out"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/domain"
)

// PaymentPortImpl is a driven adapter implementing the PaymentPort gateway port.
// This scaffold is written once; implement the external-system calls here.
type PaymentPortImpl struct{}

var _ out.PaymentPort = (*PaymentPortImpl)(nil)

// NewPaymentPortImpl constructs the gateway adapter. Inject your client here.
func NewPaymentPortImpl() *PaymentPortImpl {
	return &PaymentPortImpl{}
}

func (p *PaymentPortImpl) Payment(receipt domain.Receipt) (domain.ChargeRequest, error) {
	return domain.ChargeRequest{}, errors.New("not implemented")
}
