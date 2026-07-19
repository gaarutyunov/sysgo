package engine

import "testing"

func TestSuccessionGuardResolved(t *testing.T) {
	m := New().AddFile("a.sysml", `package App {
	import ScalarValues::*;
	action def A;
	action def W {
		in ready : Boolean;
		action a : A;
		action b : A;
		first a if ready then b;
		first b then a;
	}
}`).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("diagnostics: %v", d)
	}
	w, _ := m.Lookup("App::W")
	edges := w.Successions()
	if len(edges) != 2 {
		t.Fatalf("successions = %d, want 2", len(edges))
	}
	if edges[0].Guard != "ready" {
		t.Errorf("edge[0].Guard = %q, want ready", edges[0].Guard)
	}
	if edges[1].Guard != "" {
		t.Errorf("edge[1] should be unguarded, got %q", edges[1].Guard)
	}
}
