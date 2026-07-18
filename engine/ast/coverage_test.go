package ast

import "testing"

func TestImportSegments(t *testing.T) {
	imp := parse(t, "import A::B::C;").Members()[0].(Import)
	in, ok := imp.ImportName()
	if !ok {
		t.Fatal("no import name")
	}
	segs := in.Segments()
	if len(segs) != 3 {
		t.Fatalf("segments = %d, want 3", len(segs))
	}
	if segs[0].Text() != "A" || segs[1].Text() != "B" || segs[2].Text() != "C" {
		t.Errorf("segments = %q/%q/%q", segs[0].Text(), segs[1].Text(), segs[2].Text())
	}
}

func TestDeclarationBodyMembers(t *testing.T) {
	d := parse(t, "part def Order {\n\tattribute id : String;\n\tpart line : L;\n}").Members()[0].(Declaration)
	if !d.IsDefinition() {
		t.Fatal("want definition")
	}
	if _, ok := d.Body(); !ok {
		t.Fatal("want a body")
	}
	ms := d.Members()
	if len(ms) != 2 {
		t.Fatalf("body members = %d, want 2", len(ms))
	}
	for i, m := range ms {
		if dd, ok := m.(Declaration); !ok || !dd.IsUsage() {
			t.Errorf("body member %d = %T, want usage Declaration", i, m)
		}
	}
}

func TestAbsentOptionalAccessors(t *testing.T) {
	d := parse(t, "feature f;").Members()[0].(Declaration)
	if _, ok := d.Multiplicity(); ok {
		t.Error("unexpected multiplicity")
	}
	if _, ok := d.FeatureValue(); ok {
		t.Error("unexpected feature value")
	}
	if _, ok := d.Visibility(); ok {
		t.Error("unexpected visibility")
	}
	if _, ok := d.Body(); ok {
		t.Error("unexpected body")
	}
	if d.Members() != nil {
		t.Error("expected no body members")
	}
	if len(d.Relationships()) != 0 || len(d.Annotations()) != 0 {
		t.Error("expected no relationships/annotations")
	}
	if n, ok := d.Name(); !ok || n.String() != "f" {
		t.Errorf("name = %q (%v), want f", n.String(), ok)
	}
}

func TestErrorMemberSkipped(t *testing.T) {
	// The junk becomes an ErrorNode in the CST but is not a typed Member.
	ms := parse(t, "$$$ ; package P;").Members()
	if len(ms) != 1 {
		t.Fatalf("members = %d, want 1 (ErrorNode skipped)", len(ms))
	}
	if _, ok := ms[0].(Package); !ok {
		t.Errorf("member 0 = %T, want Package", ms[0])
	}
}

func TestPackageNoBody(t *testing.T) {
	pkg := parse(t, "package P;").Members()[0].(Package)
	if _, ok := pkg.Body(); ok {
		t.Error("unexpected body")
	}
	if len(pkg.Members()) != 0 {
		t.Error("expected no members")
	}
	if _, ok := pkg.Visibility(); ok {
		t.Error("unexpected visibility")
	}
}

func TestRelationshipKeywordOperator(t *testing.T) {
	d := parse(t, "feature f subsets g;").Members()[0].(Declaration)
	rels := d.Relationships()
	if len(rels) != 1 || rels[0].Operator() != "subsets" {
		t.Errorf("operator = %q, want subsets", rels[0].Operator())
	}
	if ts := rels[0].Targets(); len(ts) != 1 || ts[0].String() != "g" {
		t.Errorf("targets = %v, want [g]", ts)
	}
}

func TestNonDefaultFeatureValue(t *testing.T) {
	d := parse(t, "feature f = 42;").Members()[0].(Declaration)
	fv, ok := d.FeatureValue()
	if !ok {
		t.Fatal("expected feature value")
	}
	if fv.IsDefault() {
		t.Error("= is not a default (:=)")
	}
	if e, ok := fv.Expr(); !ok || e.Text() != "42" {
		t.Errorf("expr = %q (%v), want 42", e.Text(), ok)
	}
}
