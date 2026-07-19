package engine

import "testing"

const controlModel = `package App {
	action def A;
	action def B;
	action def W {
		fork f;
		action a : A;
		action b : B;
		first f then a;
		first f then b;
		join j;
	}
}`

func TestControlNodesExposed(t *testing.T) {
	m := New().AddFile("app.sysml", controlModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	w, _ := m.Lookup("App::W")

	nodes := w.ControlNodes()
	kinds := map[string]string{}
	for _, n := range nodes {
		kinds[n.Name()] = n.ControlKind()
	}
	if kinds["f"] != "fork" {
		t.Errorf("f control kind = %q, want fork", kinds["f"])
	}
	if kinds["j"] != "join" {
		t.Errorf("j control kind = %q, want join", kinds["j"])
	}
	if len(nodes) != 2 {
		t.Errorf("control nodes = %d, want 2", len(nodes))
	}
}

func TestControlNodeResolvableBySuccession(t *testing.T) {
	m := New().AddFile("app.sysml", controlModel).Build()
	w, _ := m.Lookup("App::W")
	// Both successions fork f → a and fork f → b resolve the fork node.
	edges := w.Successions()
	if len(edges) != 2 {
		t.Fatalf("successions = %d, want 2", len(edges))
	}
	for _, e := range edges {
		src, ok := e.Source()
		if !ok || src.ControlKind() != "fork" {
			t.Errorf("succession source = %+v, want the fork node", src)
		}
	}
}

func TestNonControlNode(t *testing.T) {
	m := New().AddFile("app.sysml", controlModel).Build()
	a, _ := m.Lookup("App::A")
	if a.IsControlNode() || a.ControlKind() != "" {
		t.Error("action A should not be a control node")
	}
}
