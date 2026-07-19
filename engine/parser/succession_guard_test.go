package parser

import "testing"

func TestParseGuardedSuccession(t *testing.T) {
	tree := roundTrip(t, "action def W {\n\taction a : A;\n\taction b : A;\n\tfirst a if ready then b;\n}")
	k := nodeKinds(tree.Root())
	if got := countKind(k, "Succession"); got != 1 {
		t.Fatalf("Succession count = %d, want 1 (%v)", got, k)
	}
	if got := countKind(k, "ErrorNode"); got != 0 {
		t.Errorf("unexpected error node: %v", k)
	}
	// The guard is captured as an Expr node.
	if got := countKind(k, "Expr"); got < 1 {
		t.Errorf("guard Expr not captured: %v", k)
	}
}
