package parser

import "testing"

func TestParseLoop(t *testing.T) {
	tree := roundTrip(t, "action def W {\n\tloop retries times Attempt;\n\tloop 3 times Attempt;\n}")
	k := nodeKinds(tree.Root())
	if got := countKind(k, "Loop"); got != 2 {
		t.Fatalf("Loop count = %d, want 2 (%v)", got, k)
	}
	if got := countKind(k, "ErrorNode"); got != 0 {
		t.Errorf("unexpected error node: %v", k)
	}
}
