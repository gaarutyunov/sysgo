package temporal

import (
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const appModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def Order {
		attribute id : String;
	}
	@Workflow { id = "ProcessOrder"; taskQueue = "orders"; }
	action def ProcessOrder {
		in order : Order;
	}
	@Activity { taskQueue = "orders"; }
	@RetryPolicy { maxAttempts = 3; }
	@Idempotent
	action def ChargeCard {
		in order : Order;
	}
	action def NotAnnotated;
}`

func build(t *testing.T) *engine.Model {
	t.Helper()
	m := engine.New().AddFile("app.sysml", appModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model has diagnostics (profile resolution): %v", d)
	}
	return m
}

func TestWorkflows(t *testing.T) {
	m := build(t)
	wfs := Workflows(m)
	if len(wfs) != 1 {
		t.Fatalf("workflows = %d, want 1", len(wfs))
	}
	w := wfs[0]
	if w.Name() != "ProcessOrder" {
		t.Errorf("workflow name = %q, want ProcessOrder", w.Name())
	}
	if w.ID != "ProcessOrder" || w.TaskQueue != "orders" {
		t.Errorf("workflow config = %+v, want id=ProcessOrder taskQueue=orders", w)
	}
}

func TestActivities(t *testing.T) {
	m := build(t)
	acts := Activities(m)
	if len(acts) != 1 {
		t.Fatalf("activities = %d, want 1", len(acts))
	}
	a := acts[0]
	if a.Name() != "ChargeCard" {
		t.Errorf("activity name = %q, want ChargeCard", a.Name())
	}
	if a.TaskQueue != "orders" {
		t.Errorf("taskQueue = %q, want orders", a.TaskQueue)
	}
	if !a.Idempotent {
		t.Error("ChargeCard should be idempotent")
	}
	if a.Retry == nil || a.Retry.MaxAttempts != "3" {
		t.Errorf("retry = %+v, want maxAttempts=3", a.Retry)
	}
	if a.Timeout != nil {
		t.Errorf("timeout = %+v, want nil (none declared)", a.Timeout)
	}
}

func TestUnannotatedActionsIgnored(t *testing.T) {
	m := build(t)
	// NotAnnotated is neither a workflow nor an activity.
	for _, w := range Workflows(m) {
		if w.Name() == "NotAnnotated" {
			t.Error("NotAnnotated wrongly classified as a workflow")
		}
	}
	for _, a := range Activities(m) {
		if a.Name() == "NotAnnotated" {
			t.Error("NotAnnotated wrongly classified as an activity")
		}
	}
}

func TestBundledProfilesExcluded(t *testing.T) {
	// The bundled profiles are never returned as user workflows/activities,
	// even though TemporalProfile is loaded into the workspace.
	m := build(t)
	if len(Workflows(m)) != 1 || len(Activities(m)) != 1 {
		t.Errorf("bundled profile leaked into classification: wf=%d act=%d",
			len(Workflows(m)), len(Activities(m)))
	}
}
