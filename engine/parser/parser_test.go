package parser

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine/cst"
)

// roundTrip asserts the parsed tree reproduces src byte-for-byte.
func roundTrip(t *testing.T, src string) *cst.Tree {
	t.Helper()
	tree := Parse(src)
	if got := tree.Root().Text(); got != src {
		t.Fatalf("round-trip mismatch:\n got %q\nwant %q", got, src)
	}
	if got := Namer(tree.Root().Kind()); got != "SourceFile" {
		t.Fatalf("root kind = %q, want SourceFile", got)
	}
	return tree
}

// nodeKinds returns the names of every node (not token) in a DFS pre-order.
func nodeKinds(n cst.Node) []string {
	out := []string{Namer(n.Kind())}
	for _, c := range n.Children() {
		if cn, ok := c.(cst.Node); ok {
			out = append(out, nodeKinds(cn)...)
		}
	}
	return out
}

func hasSeq(hay, needle []string) bool {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if eqStr(hay[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}

func eqStr(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParseEmpty(t *testing.T) {
	tree := roundTrip(t, "")
	if got := nodeKinds(tree.Root()); len(got) != 1 || got[0] != "SourceFile" {
		t.Errorf("empty file nodes = %v, want [SourceFile]", got)
	}
}

func TestParseOnlyTrivia(t *testing.T) {
	// A file of only comments/whitespace round-trips; trivia hangs off SourceFile.
	roundTrip(t, "// just a comment\n\n")
}

func TestParsePackageSemicolon(t *testing.T) {
	tree := roundTrip(t, "package Foo;")
	kinds := nodeKinds(tree.Root())
	if !hasSeq(kinds, []string{"Package", "QualifiedName", "Name"}) {
		t.Errorf("nodes = %v, want Package>QualifiedName>Name", kinds)
	}
}

func TestParseQualifiedName(t *testing.T) {
	tree := roundTrip(t, "package A::B::C;")
	// One QualifiedName with three Name segments.
	kinds := nodeKinds(tree.Root())
	names := 0
	for _, k := range kinds {
		if k == "Name" {
			names++
		}
	}
	if names != 3 {
		t.Errorf("Name count = %d, want 3 (A, B, C)", names)
	}
}

func TestParseNestedPackages(t *testing.T) {
	src := "package Outer {\n\tpackage Inner;\n}\n"
	tree := roundTrip(t, src)
	got := nodeKinds(tree.Root())
	// Pre-order: outer Package, its name, then its Body containing the inner
	// Package and its name.
	want := []string{
		"SourceFile",
		"Package", "QualifiedName", "Name",
		"Body",
		"Package", "QualifiedName", "Name",
	}
	if !eqStr(got, want) {
		t.Errorf("nodes = %v, want %v", got, want)
	}
}

func TestParseErrorRecovery(t *testing.T) {
	// Junk before a valid member: the junk is wrapped in an ErrorNode and the
	// package still parses. Whole thing stays lossless.
	src := "$$$ ; package Good;"
	tree := roundTrip(t, src)
	kinds := nodeKinds(tree.Root())
	if !hasSeq(kinds, []string{"ErrorNode"}) {
		t.Errorf("nodes = %v, want an ErrorNode", kinds)
	}
	if !hasSeq(kinds, []string{"Package", "QualifiedName", "Name"}) {
		t.Errorf("recovery failed, nodes = %v", kinds)
	}
}

func TestParseUnterminatedBody(t *testing.T) {
	// Missing closing brace — tolerant, still lossless, no panic.
	roundTrip(t, "package A {\n\tpackage B;")
}

func TestParseTriviaRichRoundTrip(t *testing.T) {
	src := `// top comment
package Vehicles {
	/* block */
	package Engines ;

	package Wheels {
	}
}
`
	roundTrip(t, src)
}

func TestParseGoldenStructure(t *testing.T) {
	tree := Parse("package A{}")
	want := strings.Join([]string{
		`SourceFile [0, 11)`,
		`  Package [0, 11)`,
		`    Ident [0, 7) "package"`,
		`    QualifiedName [7, 9)`,
		`      Name [7, 9)`,
		`        Whitespace [7, 8) " "`,
		`        Ident [8, 9) "A"`,
		`    Body [9, 11)`,
		`      LBrace [9, 10) "{"`,
		`      RBrace [10, 11) "}"`,
		``,
	}, "\n")
	if got := cst.Print(tree.Root(), Namer); got != want {
		t.Errorf("Print mismatch:\n got:\n%s\nwant:\n%s", got, want)
	}
}
