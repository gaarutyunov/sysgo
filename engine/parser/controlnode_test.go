package parser

import "testing"

func TestParseControlNodes(t *testing.T) {
	tree := roundTrip(t, "action def W {\n\tfork f;\n\tjoin j;\n\tmerge m;\n\tdecide d;\n}")
	k := nodeKinds(tree.Root())
	if countKind(k, "ControlNode") != 4 {
		t.Fatalf("ControlNode count = %d, want 4 (%v)", countKind(k, "ControlNode"), k)
	}
	if countKind(k, "ErrorNode") != 0 {
		t.Errorf("unexpected error node: %v", k)
	}
}

func TestParseAnonymousControlNode(t *testing.T) {
	tree := roundTrip(t, "action def W {\n\tfork;\n}")
	if countKind(nodeKinds(tree.Root()), "ControlNode") != 1 {
		t.Errorf("anonymous fork not parsed: %v", nodeKinds(tree.Root()))
	}
}
