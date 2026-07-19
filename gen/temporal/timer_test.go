package temporal

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const timerModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def Order { attribute id : String; }
	@Activity { taskQueue = "q"; }
	action def Charge { in order : Order; }
	@Activity { taskQueue = "q"; }
	action def Notify { in order : Order; }
	@Workflow { id = "W"; taskQueue = "q"; }
	action def W {
		in order : Order;
		action charge : Charge;
		accept after 30;
		action notify : Notify;
	}
}`

func buildTimerModel(t *testing.T) *engine.Model {
	t.Helper()
	m := engine.New().AddFile("app.sysml", timerModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	return m
}

func TestTimerInterleaved(t *testing.T) {
	src, err := GenerateWorkflows(buildTimerModel(t), "gen")
	if err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}
	n := norm(src)
	if !strings.Contains(n, "workflow.Sleep(ctx, 30*time.Second)") {
		t.Errorf("timer not mapped to workflow.Sleep:\n%s", src)
	}
	// The timer sits between the two activities, in source order.
	ci := strings.Index(n, `workflow.ExecuteActivity(ctx, "Charge", order)`)
	ti := strings.Index(n, "workflow.Sleep(ctx, 30*time.Second)")
	ni := strings.Index(n, `workflow.ExecuteActivity(ctx, "Notify", order)`)
	if ci < 0 || ti < 0 || ni < 0 || ci > ti || ti > ni {
		t.Errorf("timer not interleaved in order (charge=%d timer=%d notify=%d):\n%s", ci, ti, ni, src)
	}
}

// TestTimerCompiles builds the workflow with a durable timer against the SDK.
func TestTimerCompiles(t *testing.T) {
	m := buildTimerModel(t)
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
