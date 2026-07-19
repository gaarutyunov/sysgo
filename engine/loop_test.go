package engine

import "testing"

func TestLoopsResolved(t *testing.T) {
	m := New().AddFile("a.sysml", `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def O { attribute id : String; }
	@Activity { taskQueue = "q"; } action def Attempt { in o : O; }
	@Workflow { id = "W"; taskQueue = "q"; }
	action def W {
		in o : O;
		in retries : Integer;
		loop retries times Attempt;
		loop 3 times Attempt;
	}
}`).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("diagnostics: %v", d)
	}
	w, _ := m.Lookup("App::W")
	loops := w.Loops()
	if len(loops) != 2 {
		t.Fatalf("loops = %d, want 2", len(loops))
	}
	if loops[0].Count != "retries" || loops[1].Count != "3" {
		t.Errorf("counts = %q, %q, want retries, 3", loops[0].Count, loops[1].Count)
	}
	tgt, ok := loops[0].Target()
	if !ok || tgt.QualifiedName() != "App::Attempt" {
		t.Errorf("loop[0] target = %+v (%v), want App::Attempt", tgt, ok)
	}
}
