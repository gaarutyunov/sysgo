package engine

import "testing"

const successionModel = `package App {
	import ScalarValues::*;
	item def Order { attribute id : String; }
	action def ChargeCard { in order : Order; }
	action def SendReceipt { in order : Order; }
	action def ProcessOrder {
		in order : Order;
		action charge : ChargeCard;
		action notify : SendReceipt;
		first charge then notify;
	}
}`

func TestSuccessionResolved(t *testing.T) {
	m := New().AddFile("app.sysml", successionModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	proc, _ := m.Lookup("App::ProcessOrder")

	edges := proc.Successions()
	if len(edges) != 1 {
		t.Fatalf("successions = %d, want 1", len(edges))
	}
	e := edges[0]
	if e.SourceName != "charge" || e.TargetName != "notify" {
		t.Errorf("edge names = %q → %q, want charge → notify", e.SourceName, e.TargetName)
	}
	src, ok := e.Source()
	if !ok || src.QualifiedName() != "App::ProcessOrder::charge" {
		t.Errorf("source = %+v (%v), want ProcessOrder::charge", src, ok)
	}
	tgt, ok := e.Target()
	if !ok || tgt.QualifiedName() != "App::ProcessOrder::notify" {
		t.Errorf("target = %+v (%v), want ProcessOrder::notify", tgt, ok)
	}
}

func TestThenOnlySuccession(t *testing.T) {
	src := `package App {
	action def A;
	action def W {
		action step : A;
		then step;
	}
}`
	m := New().AddFile("app.sysml", src).Build()
	w, _ := m.Lookup("App::W")
	edges := w.Successions()
	if len(edges) != 1 {
		t.Fatalf("successions = %d, want 1", len(edges))
	}
	if _, ok := edges[0].Source(); ok {
		t.Error("bare `then` should have no explicit source")
	}
	tgt, ok := edges[0].Target()
	if !ok || tgt.Name() != "step" {
		t.Errorf("target = %+v (%v), want step", tgt, ok)
	}
}

func TestNoSuccessions(t *testing.T) {
	m := New().AddFile("app.sysml", successionModel).Build()
	cc, _ := m.Lookup("App::ChargeCard")
	if len(cc.Successions()) != 0 {
		t.Error("ChargeCard should have no successions")
	}
}
