package temporal

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

func TestWorkerShape(t *testing.T) {
	m := buildWorkflowModel(t)
	src, err := GenerateWorker(m, "gen")
	if err != nil {
		t.Fatalf("GenerateWorker: %v", err)
	}
	n := norm(src)
	for _, want := range []string{
		"func RunWorker(c client.Client, acts Activities) error",
		`w0 := worker.New(c, "q", worker.Options{})`,
		"w0.RegisterWorkflow(ProcessOrderWorkflow)",
		"w0.RegisterActivity(acts)",
		"w.Start()",
		"<-worker.InterruptCh()",
		"w.Stop()",
	} {
		if !strings.Contains(n, want) {
			t.Errorf("generated worker missing %q\n---\n%s", want, src)
		}
	}
}

// TestWorkerCompiles builds the activities, workflows, and worker together
// against the Temporal SDK — proving RunWorker registers the reshaped workflows
// and the Activities port and that every SDK call type-checks.
func TestWorkerCompiles(t *testing.T) {
	m := buildWorkflowModel(t)
	acts, err := GenerateActivities(m, "gen")
	if err != nil {
		t.Fatalf("GenerateActivities: %v", err)
	}
	wf, err := GenerateWorkflows(m, "gen")
	if err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}
	worker, err := GenerateWorker(m, "gen")
	if err != nil {
		t.Fatalf("GenerateWorker: %v", err)
	}
	compileFiles(t, map[string]string{
		"activities.go": acts,
		"workflows.go":  wf,
		"worker.go":     worker,
	})
}

// TestWorkerMultipleTaskQueues checks one worker is emitted per distinct task
// queue.
func TestWorkerMultipleTaskQueues(t *testing.T) {
	src := `package App {
	import TemporalProfile::*;
	@Workflow { id = "A"; taskQueue = "qa"; }
	action def A;
	@Workflow { id = "B"; taskQueue = "qb"; }
	action def B;
}`
	m := engine.New().AddFile("app.sysml", src).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	out, err := GenerateWorker(m, "gen")
	if err != nil {
		t.Fatalf("GenerateWorker: %v", err)
	}
	n := norm(out)
	for _, want := range []string{
		`w0 := worker.New(c, "qa", worker.Options{})`,
		`w1 := worker.New(c, "qb", worker.Options{})`,
		"w0.RegisterWorkflow(AWorkflow)",
		"w1.RegisterWorkflow(BWorkflow)",
	} {
		if !strings.Contains(n, want) {
			t.Errorf("multi-queue worker missing %q\n---\n%s", want, out)
		}
	}
}
