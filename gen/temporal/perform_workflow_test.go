package temporal

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const performWFModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def Order {
		attribute id : String;
	}
	@Activity { taskQueue = "q"; }
	action def ChargeCard {
		in order : Order;
	}
	@Activity { taskQueue = "q"; }
	action def SendReceipt {
		in order : Order;
	}
	@Workflow { id = "P"; taskQueue = "q"; }
	action def ProcessOrder {
		in order : Order;
		perform charge : ChargeCard;
		perform notify : SendReceipt;
	}
}`

func TestPerformBasedWorkflow(t *testing.T) {
	m := engine.New().AddFile("app.sysml", performWFModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	acts, err := GenerateActivities(m, "gen")
	if err != nil {
		t.Fatalf("GenerateActivities: %v", err)
	}
	wf, err := GenerateWorkflows(m, "gen")
	if err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}
	compileFiles(t, map[string]string{"activities.go": acts, "workflows.go": wf})

	n := norm(wf)
	if !strings.Contains(n, "func ProcessOrderWorkflow(ctx workflow.Context, order Order) error") {
		t.Errorf("workflow signature wrong:\n%s", wf)
	}
	ci := strings.Index(n, `workflow.ExecuteActivity(ctx, "ChargeCard", order)`)
	si := strings.Index(n, `workflow.ExecuteActivity(ctx, "SendReceipt", order)`)
	if ci < 0 || si < 0 {
		t.Fatalf("perform steps not emitted as activity calls:\n%s", wf)
	}
	if ci > si {
		t.Errorf("perform steps out of order:\n%s", wf)
	}
}
