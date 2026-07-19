package temporal

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const guardModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def Order { attribute id : String; }
	@Activity { taskQueue = "q"; }
	action def Charge { in order : Order; }
	@Activity { taskQueue = "q"; }
	action def Refund { in order : Order; }
	@Workflow { id = "W"; taskQueue = "q"; }
	action def W {
		in order : Order;
		in shouldRefund : Boolean;
		action charge : Charge;
		action refund : Refund;
		first charge if shouldRefund then refund;
	}
}`

func buildGuardModel(t *testing.T) *engine.Model {
	t.Helper()
	m := engine.New().AddFile("app.sysml", guardModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	return m
}

func TestGuardedSuccessionShape(t *testing.T) {
	src, err := GenerateWorkflows(buildGuardModel(t), "gen")
	if err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}
	n := norm(src)
	// Charge runs unconditionally; Refund is wrapped in the guard.
	if !strings.Contains(n, `if shouldRefund { if err := workflow.ExecuteActivity(ctx, "Refund", order).Get(ctx, nil); err != nil { return err } }`) {
		t.Errorf("guarded activity not wrapped in if:\n%s", src)
	}
	ci := strings.Index(n, `workflow.ExecuteActivity(ctx, "Charge", order)`)
	gi := strings.Index(n, "if shouldRefund {")
	if ci < 0 || gi < 0 || ci > gi {
		t.Errorf("charge should precede the guarded refund (charge=%d guard=%d):\n%s", ci, gi, src)
	}
}

// TestGuardedSuccessionCompiles builds the guarded workflow against the SDK —
// the guard expression (a bool param) must be in scope.
func TestGuardedSuccessionCompiles(t *testing.T) {
	m := buildGuardModel(t)
	acts, err := GenerateActivities(m, "gen")
	if err != nil {
		t.Fatalf("GenerateActivities: %v", err)
	}
	wf, err := GenerateWorkflows(m, "gen")
	if err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}
	compileFiles(t, map[string]string{"activities.go": acts, "workflows.go": wf})
}
