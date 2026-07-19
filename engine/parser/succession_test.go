package parser

import "testing"

func TestParseSuccession(t *testing.T) {
	tree := roundTrip(t, "action def W {\n\taction a : A;\n\taction b : B;\n\tfirst a then b;\n}")
	k := nodeKinds(tree.Root())
	if countKind(k, "Succession") != 1 {
		t.Fatalf("Succession count = %d, want 1 (%v)", countKind(k, "Succession"), k)
	}
	if countKind(k, "ErrorNode") != 0 {
		t.Errorf("unexpected error node: %v", k)
	}
}

func TestParseThenOnly(t *testing.T) {
	tree := roundTrip(t, "action def W {\n\tthen b;\n}")
	if countKind(nodeKinds(tree.Root()), "Succession") != 1 {
		t.Errorf("then-only succession not parsed: %v", nodeKinds(tree.Root()))
	}
}

func TestParseSuccessionKeyword(t *testing.T) {
	tree := roundTrip(t, "action def W {\n\tsuccession first a then b;\n}")
	k := nodeKinds(tree.Root())
	if countKind(k, "Succession") != 1 || countKind(k, "ErrorNode") != 0 {
		t.Errorf("succession-keyword form wrong: %v", k)
	}
}
