package temporal

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const loopModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def O { attribute id : String; }
	@Activity { taskQueue = "q"; } action def Attempt { in o : O; }
	@Activity { taskQueue = "q"; } action def Finish { in o : O; }
	@Workflow { id = "W"; taskQueue = "q"; }
	action def W {
		in o : O;
		in retries : Integer;
		loop retries times Attempt;
		loop 3 times Attempt;
		action done : Finish;
	}
}`

func buildLoopModel(t *testing.T) *engine.Model {
	t.Helper()
	m := engine.New().AddFile("app.sysml", loopModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	return m
}

func TestLoopShape(t *testing.T) {
	src, err := GenerateWorkflows(buildLoopModel(t), "gen")
	if err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}
	n := norm(src)
	for _, want := range []string{
		"for i0 := int64(0); i0 < retries; i0++ {",
		"for i1 := int64(0); i1 < 3; i1++ {",
		`workflow.ExecuteActivity(ctx, "Attempt", o)`,
	} {
		if !strings.Contains(n, want) {
			t.Errorf("loop mapping missing %q\n%s", want, src)
		}
	}
	// The trailing sequential activity is not inside a loop.
	if !strings.Contains(n, `if err := workflow.ExecuteActivity(ctx, "Finish", o).Get(ctx, nil); err != nil { return err }`) {
		t.Errorf("sequential Finish step missing:\n%s", src)
	}
}

func TestLoopCompiles(t *testing.T) {
	m := buildLoopModel(t)
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
