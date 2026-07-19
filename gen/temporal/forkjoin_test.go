package temporal

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const forkJoinModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def O { attribute id : String; }
	@Activity { taskQueue = "q"; } action def A { in o : O; }
	@Activity { taskQueue = "q"; } action def B { in o : O; }
	@Activity { taskQueue = "q"; } action def C { in o : O; }
	@Workflow { id = "W"; taskQueue = "q"; }
	action def W {
		in o : O;
		action pre : C;
		fork f;
		action a : A;
		action b : B;
		join j;
	}
}`

func buildForkJoinModel(t *testing.T) *engine.Model {
	t.Helper()
	m := engine.New().AddFile("app.sysml", forkJoinModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	return m
}

func TestForkJoinShape(t *testing.T) {
	src, err := GenerateWorkflows(buildForkJoinModel(t), "gen")
	if err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}
	n := norm(src)
	// Fork nodes are not workflow parameters.
	if strings.Contains(n, "f workflow") || strings.Contains(norm(src), "func WWorkflow(ctx workflow.Context, o O, f ") {
		t.Errorf("control node leaked into params:\n%s", src)
	}
	// pre runs sequentially before the fork.
	if !strings.Contains(n, `if err := workflow.ExecuteActivity(ctx, "C", o).Get(ctx, nil); err != nil { return err }`) {
		t.Errorf("pre step not sequential:\n%s", src)
	}
	// a and b start as futures, then join.
	for _, want := range []string{
		`f0 := workflow.ExecuteActivity(ctx, "A", o)`,
		`f1 := workflow.ExecuteActivity(ctx, "B", o)`,
		`if err := f0.Get(ctx, nil); err != nil { return err }`,
		`if err := f1.Get(ctx, nil); err != nil { return err }`,
	} {
		if !strings.Contains(n, want) {
			t.Errorf("parallel mapping missing %q\n%s", want, src)
		}
	}
	// The futures are started before either is Get (true parallelism).
	start1 := strings.Index(n, `f1 := workflow.ExecuteActivity(ctx, "B", o)`)
	get0 := strings.Index(n, "if err := f0.Get(ctx, nil)")
	if start1 < 0 || get0 < 0 || start1 > get0 {
		t.Errorf("both futures must start before any Get (start1=%d get0=%d):\n%s", start1, get0, src)
	}
}

func TestForkJoinCompiles(t *testing.T) {
	m := buildForkJoinModel(t)
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
