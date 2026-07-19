package temporal

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const signalModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def O { attribute id : String; }
	item def Cancel { attribute r : String; }
	@Activity { taskQueue = "q"; } action def Charge { in o : O; }
	@Activity { taskQueue = "q"; } action def Finish { in o : O; }
	@Workflow { id = "W"; taskQueue = "q"; }
	action def W {
		in o : O;
		action charge : Charge;
		accept Cancel;
		action done : Finish;
	}
}`

func buildSignalModel(t *testing.T) *engine.Model {
	t.Helper()
	m := engine.New().AddFile("app.sysml", signalModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	return m
}

func TestSignalAcceptShape(t *testing.T) {
	src, err := GenerateWorkflows(buildSignalModel(t), "gen")
	if err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}
	n := norm(src)
	if !strings.Contains(n, `workflow.GetSignalChannel(ctx, "Cancel").Receive(ctx, nil)`) {
		t.Errorf("signal accept not mapped to a signal receive:\n%s", src)
	}
	// Interleaved in order: charge, signal, finish.
	ci := strings.Index(n, `workflow.ExecuteActivity(ctx, "Charge", o)`)
	si := strings.Index(n, "workflow.GetSignalChannel(ctx, \"Cancel\")")
	fi := strings.Index(n, `workflow.ExecuteActivity(ctx, "Finish", o)`)
	if ci < 0 || si < 0 || fi < 0 || ci > si || si > fi {
		t.Errorf("signal not interleaved in order (charge=%d signal=%d finish=%d):\n%s", ci, si, fi, src)
	}
}

func TestSignalAcceptCompiles(t *testing.T) {
	m := buildSignalModel(t)
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
