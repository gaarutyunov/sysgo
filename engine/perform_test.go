package engine

import "testing"

const performModel = `package App {
	import ScalarValues::*;
	item def Order { attribute id : String; }
	action def ChargeCard { in order : Order; }
	action def SendReceipt { in order : Order; }
	action def ProcessOrder {
		in order : Order;
		perform charge : ChargeCard;
		perform notify : SendReceipt;
	}
	action def Direct {
		perform ChargeCard;
	}
}`

func TestPerformStepsResolved(t *testing.T) {
	m := New().AddFile("app.sysml", performModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	proc, _ := m.Lookup("App::ProcessOrder")

	steps := proc.Performs()
	if len(steps) != 2 {
		t.Fatalf("perform steps = %d, want 2", len(steps))
	}
	// Declaration order is preserved.
	if steps[0].Name != "charge" || steps[1].Name != "notify" {
		t.Errorf("step names = %q/%q, want charge/notify", steps[0].Name, steps[1].Name)
	}
	// Targets resolve to the activity definitions.
	tgt0, ok := steps[0].Target()
	if !ok || tgt0.QualifiedName() != "App::ChargeCard" {
		t.Errorf("charge target = %+v (%v), want App::ChargeCard", tgt0, ok)
	}
	tgt1, ok := steps[1].Target()
	if !ok || tgt1.QualifiedName() != "App::SendReceipt" {
		t.Errorf("notify target = %+v (%v), want App::SendReceipt", tgt1, ok)
	}
	if steps[0].TargetName != "ChargeCard" {
		t.Errorf("TargetName = %q, want ChargeCard", steps[0].TargetName)
	}
}

func TestPerformDirectReference(t *testing.T) {
	m := New().AddFile("app.sysml", performModel).Build()
	direct, _ := m.Lookup("App::Direct")
	steps := direct.Performs()
	if len(steps) != 1 {
		t.Fatalf("perform steps = %d, want 1", len(steps))
	}
	// Direct `perform ChargeCard;` has no local name; target is the reference.
	if steps[0].Name != "" {
		t.Errorf("direct perform name = %q, want empty", steps[0].Name)
	}
	tgt, ok := steps[0].Target()
	if !ok || tgt.QualifiedName() != "App::ChargeCard" {
		t.Errorf("direct target = %+v (%v), want App::ChargeCard", tgt, ok)
	}
}

func TestNoPerforms(t *testing.T) {
	m := New().AddFile("app.sysml", performModel).Build()
	cc, _ := m.Lookup("App::ChargeCard")
	if len(cc.Performs()) != 0 {
		t.Errorf("ChargeCard should have no perform steps")
	}
}

func TestUnresolvedPerformDiagnostic(t *testing.T) {
	m := New().AddFile("app.sysml", "package App {\n\taction def W { perform Missing; }\n}").Build()
	w, _ := m.Lookup("App::W")
	steps := w.Performs()
	if len(steps) != 1 || steps[0].IsResolved() {
		t.Errorf("expected one unresolved perform, got %+v", steps)
	}
	if _, ok := steps[0].Target(); ok {
		t.Error("unresolved perform should have no target")
	}
}
