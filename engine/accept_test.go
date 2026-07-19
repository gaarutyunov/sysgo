package engine

import "testing"

const acceptModel = `package App {
	import ScalarValues::*;
	item def CancelReq { attribute id : String; }
	action def W {
		accept cancel : CancelReq;
		accept after 30;
		accept at deadline;
	}
	action def deadline;
}`

func TestAcceptsExposed(t *testing.T) {
	m := New().AddFile("app.sysml", acceptModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	w, _ := m.Lookup("App::W")
	accepts := w.Accepts()
	if len(accepts) != 3 {
		t.Fatalf("accepts = %d, want 3", len(accepts))
	}

	// signal accept
	if accepts[0].Mode != "signal" || accepts[0].Ref != "cancel" {
		t.Errorf("accept[0] = %+v, want signal cancel", accepts[0])
	}
	if accepts[0].IsTimer() {
		t.Error("signal accept should not be a timer")
	}
	// after timer with a literal value
	if accepts[1].Mode != "after" || !accepts[1].IsTimer() || accepts[1].Ref != "30" {
		t.Errorf("accept[1] = %+v, want after timer 30", accepts[1])
	}
	if _, ok := accepts[1].Target(); ok {
		t.Error("literal timer value should have no target")
	}
	// at timer referencing a symbol
	if accepts[2].Mode != "at" || !accepts[2].IsTimer() {
		t.Errorf("accept[2] = %+v, want at timer", accepts[2])
	}
	if tgt, ok := accepts[2].Target(); !ok || tgt.QualifiedName() != "App::deadline" {
		t.Errorf("at target = %+v (%v), want App::deadline", tgt, ok)
	}
}

func TestNoAccepts(t *testing.T) {
	m := New().AddFile("app.sysml", acceptModel).Build()
	d, _ := m.Lookup("App::deadline")
	if len(d.Accepts()) != 0 {
		t.Error("deadline should have no accepts")
	}
}
