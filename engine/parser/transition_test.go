package parser

import "testing"

func TestParseTransition(t *testing.T) {
	src := "state def TrafficLight {\n" +
		"\tstate Red;\n" +
		"\tstate Green;\n" +
		"\ttransition RedToGreen first Red accept Go if ready do Open then Green;\n" +
		"\ttransition first Green then Red;\n" +
		"}"
	tree := roundTrip(t, src)
	k := nodeKinds(tree.Root())
	if got := countKind(k, "Transition"); got != 2 {
		t.Fatalf("Transition count = %d, want 2 (%v)", got, k)
	}
	if got := countKind(k, "ErrorNode"); got != 0 {
		t.Errorf("unexpected error node: %v", k)
	}
}

func TestParseTransitionMinimal(t *testing.T) {
	tree := roundTrip(t, "state def S {\n\ttransition first A then B;\n}")
	k := nodeKinds(tree.Root())
	if got := countKind(k, "Transition"); got != 1 {
		t.Fatalf("Transition count = %d, want 1 (%v)", got, k)
	}
	if got := countKind(k, "ErrorNode"); got != 0 {
		t.Errorf("unexpected error node: %v", k)
	}
}
