package main

import (
	"testing"

	"go.temporal.io/sdk/testsuite"

	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/port/in"
	"github.com/gaarutyunov/sysgo/examples/combined/orders"
)

// spyUseCase records PlaceOrder invocations so the test can assert the Temporal
// activity drove the DDD use case.
type spyUseCase struct{ calls []in.PlaceOrderInput }

func (s *spyUseCase) PlaceOrder(input in.PlaceOrderInput) (in.PlaceOrderOutput, error) {
	s.calls = append(s.calls, input)
	return in.PlaceOrderOutput{}, nil
}

// TestProcessOrderWorkflow runs the generated workflow in the Temporal test
// environment (no server) and asserts ChargeCard drove the PlaceOrder use case.
func TestProcessOrderWorkflow(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	spy := &spyUseCase{}
	env.RegisterActivity(NewOrderActivities(spy))

	env.ExecuteWorkflow(orders.ProcessOrderWorkflow, orders.Order{Id: "o-1"})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	if len(spy.calls) != 1 {
		t.Fatalf("PlaceOrder called %d times, want 1", len(spy.calls))
	}
	if got := spy.calls[0].Order.ID; got != "o-1" {
		t.Errorf("order id = %q, want o-1", got)
	}
}
