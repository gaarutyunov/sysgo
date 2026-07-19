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

// compileFiles writes several generated files into one throwaway module that
// requires the Temporal SDK and runs `go build`, verifying they compile
// together. The blank imports in sdkdeps_test.go keep the SDK in the module
// cache so this resolves without perturbing the network in the common case.
func compileFiles(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	mod := "module gen\n\ngo 1.26.5\n\nrequire go.temporal.io/sdk " +
		moduleVersion(t, "go.temporal.io/sdk") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	if sum, err := os.ReadFile(parentGoSum(t)); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644)
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated code did not compile: %v\n%s", err, out)
	}
}

// moduleVersion returns the selected version of a module in the parent module.
func moduleVersion(t *testing.T, path string) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Version}}", path).Output()
	if err != nil {
		t.Fatalf("go list -m %s: %v", path, err)
	}
	return strings.TrimSpace(string(out))
}

// parentGoSum returns the path to the parent module's go.sum.
func parentGoSum(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	return filepath.Join(filepath.Dir(strings.TrimSpace(string(out))), "go.sum")
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

	if !strings.Contains(n, "func ProcessOrderWorkflow(ctx workflow.Context, order Order) error") {
		t.Errorf("workflow signature wrong:\n%s", src)
	}
	if !strings.Contains(n, "workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})") {
		t.Errorf("activity options not applied:\n%s", src)
	}
	// The activities are executed by name, in declaration order.
	ci := strings.Index(n, `workflow.ExecuteActivity(ctx, "ChargeCard", order)`)
	si := strings.Index(n, `workflow.ExecuteActivity(ctx, "SendReceipt", order)`)
	if ci < 0 || si < 0 {
		t.Fatalf("missing activity calls; got:\n%s", src)
	}
	if ci > si {
		t.Errorf("activity calls out of order (charge should precede notify):\n%s", src)
	}
	if !strings.Contains(n, `if err := workflow.ExecuteActivity(ctx, "ChargeCard", order).Get(ctx, nil); err != nil { return err }`) {
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
