package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const genTemporalModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def Order { attribute id : String; }
	@Activity { taskQueue = "orders"; } action def Charge { in order : Order; }
	@Workflow { id = "ProcessOrder"; taskQueue = "orders"; }
	action def ProcessOrder {
		in order : Order;
		action c : Charge;
	}
}`

func runGen(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestGenTemporal(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "app.sysml")
	if err := os.WriteFile(model, []byte(genTemporalModel), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "gen")
	if _, err := runGen(t, "gen", "temporal", model, "--out", out, "--package", "orders"); err != nil {
		t.Fatalf("gen temporal: %v", err)
	}

	for _, f := range []string{"activities.go", "workflows.go", "worker.go", "workflowcheck.sh"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("expected %s to be generated: %v", f, err)
		}
	}
	wf, err := os.ReadFile(filepath.Join(out, "workflows.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"package orders", "func ProcessOrderWorkflow", "workflow.ExecuteActivity"} {
		if !strings.Contains(string(wf), want) {
			t.Errorf("workflows.go missing %q", want)
		}
	}
	worker, _ := os.ReadFile(filepath.Join(out, "worker.go"))
	if !strings.Contains(string(worker), "func RunWorker") {
		t.Errorf("worker.go missing RunWorker")
	}
	// The workflowcheck script is executable.
	info, err := os.Stat(filepath.Join(out, "workflowcheck.sh"))
	if err == nil && info.Mode().Perm()&0o100 == 0 {
		t.Errorf("workflowcheck.sh is not executable: %v", info.Mode())
	}
}

// TestGenTemporalDeterministic checks two runs of the command produce identical
// output — the property the drift check relies on.
func TestGenTemporalDeterministic(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "app.sysml")
	if err := os.WriteFile(model, []byte(genTemporalModel), 0o644); err != nil {
		t.Fatal(err)
	}
	read := func(out string) map[string]string {
		if _, err := runGen(t, "gen", "temporal", model, "--out", out, "--package", "orders"); err != nil {
			t.Fatalf("gen temporal: %v", err)
		}
		files := map[string]string{}
		entries, _ := os.ReadDir(out)
		for _, e := range entries {
			b, _ := os.ReadFile(filepath.Join(out, e.Name()))
			files[e.Name()] = string(b)
		}
		return files
	}
	a := read(filepath.Join(dir, "a"))
	b := read(filepath.Join(dir, "b"))
	if len(a) == 0 {
		t.Fatal("no files generated")
	}
	for name, ca := range a {
		if cb, ok := b[name]; !ok || cb != ca {
			t.Errorf("non-deterministic output for %s", name)
		}
	}
}

// TestGenTemporalDiagnostics fails (non-nil error) when the model has
// diagnostics, so CI generation surfaces model errors.
func TestGenTemporalDiagnostics(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "bad.sysml")
	// References an undefined type -> unresolved diagnostic.
	if err := os.WriteFile(model, []byte(`package App {
	@Workflow { id = "W"; taskQueue = "q"; }
	action def W { in x : Nonexistent; }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "gen")
	if _, err := runGen(t, "gen", "temporal", model, "--out", out); err == nil {
		t.Error("expected an error for a model with diagnostics, got nil")
	}
}
