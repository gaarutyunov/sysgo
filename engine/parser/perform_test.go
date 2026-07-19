package parser

import "testing"

func TestParsePerformTyped(t *testing.T) {
	tree := roundTrip(t, "action def W {\n\tperform action charge : ChargeCard;\n}")
	k := nodeKinds(tree.Root())
	if countKind(k, "Perform") != 1 {
		t.Fatalf("Perform count = %d, want 1 (%v)", countKind(k, "Perform"), k)
	}
	if countKind(k, "ErrorNode") != 0 {
		t.Errorf("unexpected error node: %v", k)
	}
	// The perform carries the target as a relationship (: ChargeCard).
	if countKind(k, "Relationship") != 1 {
		t.Errorf("Relationship count = %d, want 1", countKind(k, "Relationship"))
	}
}

func TestParsePerformDirect(t *testing.T) {
	tree := roundTrip(t, "action def W {\n\tperform ChargeCard;\n}")
	k := nodeKinds(tree.Root())
	if countKind(k, "Perform") != 1 || countKind(k, "ErrorNode") != 0 {
		t.Errorf("direct perform parse wrong: %v", k)
	}
}

func TestParsePerformSequence(t *testing.T) {
	// Multiple performs in order, alongside a param.
	tree := roundTrip(t, "action def W {\n\tin x : T;\n\tperform charge : ChargeCard;\n\tperform notify : SendReceipt;\n}")
	if countKind(nodeKinds(tree.Root()), "Perform") != 2 {
		t.Errorf("want 2 performs: %v", nodeKinds(tree.Root()))
	}
}
