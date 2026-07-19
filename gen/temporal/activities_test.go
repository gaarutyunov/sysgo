package temporal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const activityModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def Order {
		attribute id : String;
		attribute total : Real;
	}
	@Activity { taskQueue = "orders"; }
	@RetryPolicy { maxAttempts = 3; }
	@Idempotent
	action def ChargeCard {
		in order : Order;
	}
	@Activity { taskQueue = "orders"; }
	action def SendReceipt {
		in order : Order;
	}
}`

// mustCompile writes src into a throwaway module and runs `go build`, so the
// test verifies the generated code actually compiles (stdlib-only, offline).
func mustCompile(t *testing.T, src string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gen.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module gen\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated code did not compile: %v\n%s\n--- source ---\n%s", err, out, src)
	}
}

func genActivities(t *testing.T) string {
	t.Helper()
	m := engine.New().AddFile("app.sysml", activityModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	src, err := GenerateActivities(m, "activities")
	if err != nil {
		t.Fatalf("GenerateActivities: %v", err)
	}
	return src
}

func TestGeneratedActivitiesCompile(t *testing.T) {
	mustCompile(t, genActivities(t))
}

// norm collapses runs of whitespace to single spaces so assertions ignore
// gofmt's field/value alignment.
func norm(s string) string { return strings.Join(strings.Fields(s), " ") }

func TestActivitiesInterfaceShape(t *testing.T) {
	src := norm(genActivities(t))
	for _, want := range []string{
		"package activities",
		"type Order struct",
		"Id string",
		"Total float64",
		"type Activities interface",
		"ChargeCard(ctx context.Context, order Order) error",
		"SendReceipt(ctx context.Context, order Order) error",
		"type ActivityOptions struct",
		"Options = map[string]ActivityOptions",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated source missing %q; got:\n%s", want, src)
		}
	}
}

func TestActivityOptionsValues(t *testing.T) {
	src := norm(genActivities(t))
	// ChargeCard's options: task queue, idempotent, and retry max attempts.
	for _, want := range []string{`TaskQueue: "orders"`, "Idempotent: true", `MaxAttempts: "3"`} {
		if !strings.Contains(src, want) {
			t.Errorf("options missing %q; got:\n%s", want, src)
		}
	}
}

func TestDeterministicOutput(t *testing.T) {
	a := genActivities(t)
	b := genActivities(t)
	if a != b {
		t.Error("GenerateActivities output is not deterministic")
	}
}
