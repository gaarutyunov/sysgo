package temporal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const workflowModel = `package App {
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
	@Workflow { id = "ProcessOrder"; taskQueue = "q"; }
	action def ProcessOrder {
		in order : Order;
		action charge : ChargeCard;
		action notify : SendReceipt;
	}
}`

func buildWorkflowModel(t *testing.T) *engine.Model {
	t.Helper()
	m := engine.New().AddFile("app.sysml", workflowModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	return m
}

// compileFiles writes several generated files into one throwaway module and
// runs `go build`, verifying they compile together (stdlib-only, offline).
func compileFiles(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module gen\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated code did not compile: %v\n%s", err, out)
	}
}

func TestGeneratedWorkflowCompiles(t *testing.T) {
	m := buildWorkflowModel(t)
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

func TestWorkflowShapeAndOrder(t *testing.T) {
	m := buildWorkflowModel(t)
	src, err := GenerateWorkflows(m, "gen")
	if err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}
	n := norm(src)

	if !strings.Contains(n, "func ProcessOrderWorkflow(ctx context.Context, acts Activities, order Order) error") {
		t.Errorf("workflow signature wrong:\n%s", src)
	}
	// The activity steps are called in declaration order.
	ci := strings.Index(n, "acts.ChargeCard(ctx, order)")
	si := strings.Index(n, "acts.SendReceipt(ctx, order)")
	if ci < 0 || si < 0 {
		t.Fatalf("missing activity calls; got:\n%s", src)
	}
	if ci > si {
		t.Errorf("activity calls out of order (charge should precede notify):\n%s", src)
	}
	if !strings.Contains(n, "if err := acts.ChargeCard(ctx, order); err != nil { return err }") {
		t.Errorf("step error handling wrong:\n%s", src)
	}
}

func TestWorkflowDeterministic(t *testing.T) {
	m := buildWorkflowModel(t)
	a, _ := GenerateWorkflows(m, "gen")
	b, _ := GenerateWorkflows(m, "gen")
	if a != b {
		t.Error("GenerateWorkflows output is not deterministic")
	}
}

func TestNoWorkflows(t *testing.T) {
	m := engine.New().AddFile("m.sysml", "package M { part def X; }").Build()
	src, err := GenerateWorkflows(m, "gen")
	if err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}
	// Only the package clause + header; no workflow functions.
	if strings.Contains(src, "Workflow(ctx") {
		t.Errorf("unexpected workflow function:\n%s", src)
	}
}
