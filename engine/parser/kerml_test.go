package parser

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine/cst"
)

func countKind(kinds []string, k string) int {
	n := 0
	for _, x := range kinds {
		if x == k {
			n++
		}
	}
	return n
}

func TestParseImport(t *testing.T) {
	for _, src := range []string{
		"import A;",
		"import A::B::C;",
		"import A::B::*;",
		"import A::**;",
		"public import Base::Thing;",
	} {
		tree := roundTrip(t, src)
		k := nodeKinds(tree.Root())
		if countKind(k, "Import") != 1 || countKind(k, "ImportName") != 1 {
			t.Errorf("Parse(%q): want one Import and one ImportName, got %v", src, k)
		}
	}
	// Visibility is captured on the import.
	if k := nodeKinds(Parse("public import X;").Root()); countKind(k, "Visibility") != 1 {
		t.Errorf("public import missing Visibility node: %v", k)
	}
}

func TestParseTypeDeclKinds(t *testing.T) {
	for _, src := range []string{
		"type T;",
		"classifier C;",
		"class K;",
		"struct S;",
		"datatype D;",
		"feature f;",
		"namespace N {}",
		"abstract classifier Base;",
	} {
		tree := roundTrip(t, src)
		if !hasSeq(nodeKinds(tree.Root()), []string{"TypeDecl"}) {
			t.Errorf("Parse(%q): missing TypeDecl, got %v", src, nodeKinds(tree.Root()))
		}
	}
}

func TestParseRelationships(t *testing.T) {
	cases := map[string]int{
		"classifier C :> Base;":       1, // specialization operator
		"classifier C specializes B;": 1, // specialization keyword
		"feature f :>> g;":            1, // redefinition operator
		"feature f redefines g;":      1, // redefinition keyword
		"feature f subsets g;":        1, // subsetting keyword
		"feature f : Real;":           1, // feature typing
		"feature f ~ Conj;":           1, // conjugation
		"classifier C :> A, B;":       1, // one clause, two targets
		"feature f : Real subsets g;": 2, // typing + subsetting
	}
	for src, wantRels := range cases {
		tree := roundTrip(t, src)
		if got := countKind(nodeKinds(tree.Root()), "Relationship"); got != wantRels {
			t.Errorf("Parse(%q): Relationship count = %d, want %d (%v)", src, got, wantRels, nodeKinds(tree.Root()))
		}
	}
	// The comma list yields two target QualifiedNames inside one Relationship.
	tree := Parse("classifier C :> A, B;")
	if got := countKind(nodeKinds(tree.Root()), "QualifiedName"); got != 3 { // C, A, B
		t.Errorf("QualifiedName count = %d, want 3", got)
	}
}

func TestParseFeatureValue(t *testing.T) {
	for _, src := range []string{
		"feature x : Real = 1.5;",
		"feature x := 42;",
		`feature s = "hi";`,
		"feature r = other::ref;",
	} {
		tree := roundTrip(t, src)
		k := nodeKinds(tree.Root())
		if !hasSeq(k, []string{"FeatureValue", "Expr"}) {
			t.Errorf("Parse(%q): missing FeatureValue>Expr, got %v", src, k)
		}
	}
}

func TestParseVisibilityOnDecl(t *testing.T) {
	tree := roundTrip(t, "private feature secret : Int;")
	if !hasSeq(nodeKinds(tree.Root()), []string{"TypeDecl", "Visibility"}) {
		t.Errorf("missing Visibility on decl: %v", nodeKinds(tree.Root()))
	}
}

func TestParseNestedKerMLBody(t *testing.T) {
	src := `package P {
	public import Base::*;
	import Other::**;
	abstract classifier Vehicle :> Base::Thing {
		feature mass : Real := 1000;
		feature 'peak power' : Real subsets mass;
	}
	feature v ~ Conj;
}
`
	tree := roundTrip(t, src)
	k := nodeKinds(tree.Root())
	if countKind(k, "Import") != 2 {
		t.Errorf("Import count = %d, want 2", countKind(k, "Import"))
	}
	if countKind(k, "TypeDecl") < 3 {
		t.Errorf("TypeDecl count = %d, want >= 3", countKind(k, "TypeDecl"))
	}
}

func TestParseKerMLErrorRecovery(t *testing.T) {
	// A junk member between two good ones: the junk is an ErrorNode and both
	// declarations still parse. Whole thing is lossless.
	src := "type A; %%% $ ; feature b : Int;"
	tree := roundTrip(t, src)
	k := nodeKinds(tree.Root())
	if countKind(k, "ErrorNode") == 0 {
		t.Errorf("expected an ErrorNode, got %v", k)
	}
	if countKind(k, "TypeDecl") != 2 {
		t.Errorf("TypeDecl count = %d, want 2 (recovery failed): %v", countKind(k, "TypeDecl"), k)
	}
}

func TestParseRelationshipGolden(t *testing.T) {
	tree := Parse("type T:>B;")
	want := strings.Join([]string{
		`SourceFile [0, 10)`,
		`  TypeDecl [0, 10)`,
		`    Ident [0, 4) "type"`,
		`    QualifiedName [4, 6)`,
		`      Name [4, 6)`,
		`        Whitespace [4, 5) " "`,
		`        Ident [5, 6) "T"`,
		`    Relationship [6, 9)`,
		`      Specializes [6, 8) ":>"`,
		`      QualifiedName [8, 9)`,
		`        Name [8, 9)`,
		`          Ident [8, 9) "B"`,
		`    Semicolon [9, 10) ";"`,
		``,
	}, "\n")
	if got := cst.Print(tree.Root(), Namer); got != want {
		t.Errorf("Print mismatch:\n got:\n%s\nwant:\n%s", got, want)
	}
}
