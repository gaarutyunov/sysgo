package ast

import (
	"os"
	"testing"

	"github.com/gaarutyunov/sysgo/engine/cst"
	"github.com/gaarutyunov/sysgo/engine/parser"
)

func parse(t *testing.T, src string) SourceFile {
	t.Helper()
	return New(parser.Parse(src))
}

func TestSourceFileMembers(t *testing.T) {
	sf := parse(t, "package P;\nimport A::B;\npart def Q;\nfeature f;")
	ms := sf.Members()
	if len(ms) != 4 {
		t.Fatalf("Members len = %d, want 4", len(ms))
	}
	if _, ok := ms[0].(Package); !ok {
		t.Errorf("member 0 = %T, want Package", ms[0])
	}
	if _, ok := ms[1].(Import); !ok {
		t.Errorf("member 1 = %T, want Import", ms[1])
	}
	d, ok := ms[2].(Declaration)
	if !ok || !d.IsDefinition() {
		t.Errorf("member 2 = %T (def=%v), want a definition Declaration", ms[2], ok && d.IsDefinition())
	}
	if d3, ok := ms[3].(Declaration); !ok || !d3.IsTypeDecl() {
		t.Errorf("member 3 wrong: %T", ms[3])
	}
}

func TestPackageAccessors(t *testing.T) {
	sf := parse(t, "public package Outer::Inner {\n\tpart def X;\n}")
	pkg := sf.Members()[0].(Package)

	vis, ok := pkg.Visibility()
	if !ok || vis.Text() != "public" {
		t.Errorf("visibility = %q (%v), want public", vis.Text(), ok)
	}
	name, ok := pkg.Name()
	if !ok || name.String() != "Outer::Inner" {
		t.Errorf("name = %q (%v), want Outer::Inner", name.String(), ok)
	}
	if _, ok := pkg.Body(); !ok {
		t.Error("expected a body")
	}
	inner := pkg.Members()
	if len(inner) != 1 {
		t.Fatalf("package members = %d, want 1", len(inner))
	}
	if d, ok := inner[0].(Declaration); !ok || !d.IsDefinition() {
		t.Errorf("inner member wrong: %T", inner[0])
	}
}

func TestImportAccessors(t *testing.T) {
	tests := []struct {
		src       string
		wildcard  bool
		recursive bool
	}{
		{"import A::B;", false, false},
		{"import A::B::*;", true, false},
		{"import A::**;", true, true},
	}
	for _, tt := range tests {
		imp := parse(t, tt.src).Members()[0].(Import)
		in, ok := imp.ImportName()
		if !ok {
			t.Fatalf("Parse(%q): no ImportName", tt.src)
		}
		if in.IsWildcard() != tt.wildcard {
			t.Errorf("Parse(%q): IsWildcard = %v, want %v", tt.src, in.IsWildcard(), tt.wildcard)
		}
		if in.IsRecursive() != tt.recursive {
			t.Errorf("Parse(%q): IsRecursive = %v, want %v", tt.src, in.IsRecursive(), tt.recursive)
		}
	}
	// Visibility on import.
	imp := parse(t, "private import X;").Members()[0].(Import)
	if v, ok := imp.Visibility(); !ok || v.Text() != "private" {
		t.Errorf("import visibility = %q (%v)", v.Text(), ok)
	}
}

func TestDeclarationAccessors(t *testing.T) {
	d := parse(t, "abstract classifier Car :> Vehicle, Machine;").Members()[0].(Declaration)
	if !d.IsTypeDecl() {
		t.Error("want IsTypeDecl")
	}
	name, ok := d.Name()
	if !ok || name.String() != "Car" {
		t.Errorf("name = %q (%v), want Car", name.String(), ok)
	}
	rels := d.Relationships()
	if len(rels) != 1 {
		t.Fatalf("relationships = %d, want 1", len(rels))
	}
	if rels[0].Operator() != ":>" {
		t.Errorf("operator = %q, want :>", rels[0].Operator())
	}
	targets := rels[0].Targets()
	if len(targets) != 2 || targets[0].String() != "Vehicle" || targets[1].String() != "Machine" {
		t.Errorf("targets = %v, want [Vehicle Machine]", targets)
	}
}

func TestDeclarationFeatureValueAndMultiplicity(t *testing.T) {
	d := parse(t, "part lines : LineItem[*] := seed;").Members()[0].(Declaration)
	if !d.IsUsage() {
		t.Error("want IsUsage")
	}
	if m, ok := d.Multiplicity(); !ok || m.Text() != "[*]" {
		t.Errorf("multiplicity = %q (%v), want [*]", m.Text(), ok)
	}
	fv, ok := d.FeatureValue()
	if !ok {
		t.Fatal("expected a feature value")
	}
	if !fv.IsDefault() {
		t.Error("expected := (default) feature value")
	}
	if e, ok := fv.Expr(); !ok || e.Text() != "seed" {
		t.Errorf("expr = %q (%v), want seed", e.Text(), ok)
	}
}

func TestAnnotations(t *testing.T) {
	d := parse(t, "@Approved part def X;").Members()[0].(Declaration)
	anns := d.Annotations()
	if len(anns) != 1 {
		t.Fatalf("annotations = %d, want 1", len(anns))
	}
	if n, ok := anns[0].Name(); !ok || n.String() != "Approved" {
		t.Errorf("annotation name = %q (%v), want Approved", n.String(), ok)
	}
}

func TestInspectVisitsAll(t *testing.T) {
	sf := parse(t, "package P { part def X; }")
	nodes := 0
	names := 0
	Inspect(sf.Syntax(), func(n cst.Node) bool {
		nodes++
		if kindOf(n) == parser.KindQualifiedName {
			names++
		}
		return true
	})
	if nodes < 5 {
		t.Errorf("visited %d nodes, want >= 5", nodes)
	}
	if names != 2 { // P and X
		t.Errorf("QualifiedName nodes = %d, want 2", names)
	}
}

func TestInspectStops(t *testing.T) {
	sf := parse(t, "package P { part def X; }")
	// Returning false at the package must skip its subtree.
	visited := 0
	Inspect(sf.Syntax(), func(n cst.Node) bool {
		visited++
		return kindOf(n) != parser.KindPackage
	})
	// SourceFile + Package only.
	if visited != 2 {
		t.Errorf("visited %d nodes, want 2 (stopped at Package)", visited)
	}
}

func TestFormatRoundTrip(t *testing.T) {
	srcs := []string{
		"package P;",
		"public package Outer::Inner {\n\tpart def X :> Base;\n}\n",
		"// c\nfeature f : Real := 1.5;\n",
	}
	for _, src := range srcs {
		sf := parse(t, src)
		if got := Format(sf); got != src {
			t.Errorf("Format round-trip:\n got %q\nwant %q", got, src)
		}
	}
}

func TestFormatExampleModel(t *testing.T) {
	const path = "../../examples/order/OrderContext.sysml"
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sf := parse(t, string(src))
	if got := Format(sf); got != string(src) {
		t.Fatal("example model did not round-trip through Format")
	}
	// The typed view sees the package and its members.
	ms := sf.Members()
	if len(ms) != 1 {
		t.Fatalf("top-level members = %d, want 1 (the package)", len(ms))
	}
	pkg, ok := ms[0].(Package)
	if !ok {
		t.Fatalf("top member = %T, want Package", ms[0])
	}
	defs := 0
	for _, m := range pkg.Members() {
		if d, ok := m.(Declaration); ok && d.IsDefinition() {
			defs++
		}
	}
	if defs < 5 {
		t.Errorf("definitions in package = %d, want >= 5", defs)
	}
}
