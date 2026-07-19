package parser

import "testing"

func TestParseAccept(t *testing.T) {
	tree := roundTrip(t, "action def W {\n\taccept cancel;\n\taccept s : Sig;\n\taccept after 5;\n\taccept at now;\n}")
	k := nodeKinds(tree.Root())
	if countKind(k, "Accept") != 4 {
		t.Fatalf("Accept count = %d, want 4 (%v)", countKind(k, "Accept"), k)
	}
	if countKind(k, "ErrorNode") != 0 {
		t.Errorf("unexpected error node: %v", k)
	}
}
