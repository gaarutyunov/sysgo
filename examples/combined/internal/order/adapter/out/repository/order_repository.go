// Code scaffolded by sysgo; edit freely (not regenerated).

package repository

import (
	"sync"

	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/port/out"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/domain"
)

// OrderRepositoryImpl is an in-memory driven adapter implementing the
// OrderRepository port. A real service would talk to a database.
type OrderRepositoryImpl struct {
	mu     sync.Mutex
	orders map[string]domain.Order
}

var _ out.OrderRepository = (*OrderRepositoryImpl)(nil)

// NewOrderRepositoryImpl constructs an empty in-memory repository.
func NewOrderRepositoryImpl() *OrderRepositoryImpl {
	return &OrderRepositoryImpl{orders: map[string]domain.Order{}}
}

// OrderRepository persists the order.
func (o *OrderRepositoryImpl) OrderRepository(order domain.Order) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.orders[order.ID] = order
	return nil
}

// Get returns a stored order (helper for tests/consumers).
func (o *OrderRepositoryImpl) Get(id string) (domain.Order, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	order, ok := o.orders[id]
	return order, ok
}

// Len returns how many orders are stored.
func (o *OrderRepositoryImpl) Len() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.orders)
}
