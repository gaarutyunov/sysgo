package hir

import "testing"

func mustMember(t *testing.T, s *Symbol, name string) *Symbol {
	t.Helper()
	m, ok := s.Member(name)
	if !ok {
		t.Fatalf("symbol %q has no member %q", s.QualifiedName(), name)
	}
	return m
}

func TestAnalyzeSymbolTree(t *testing.T) {
	r := Analyze("package A {\n\tpart def X;\n\tattribute y : Real;\n}")
	root := r.Model.Root
	a := mustMember(t, root, "A")
	if a.Kind != KindPackage {
		t.Errorf("A kind = %v, want Package", a.Kind)
	}
	x := mustMember(t, a, "X")
	if x.Kind != KindDefinition {
		t.Errorf("X kind = %v, want Definition", x.Kind)
	}
	y := mustMember(t, a, "y")
	if y.Kind != KindUsage {
		t.Errorf("y kind = %v, want Usage", y.Kind)
	}
	if got := x.QualifiedName(); got != "A::X" {
		t.Errorf("X qualified name = %q, want A::X", got)
	}
	if len(a.Children()) != 2 {
		t.Errorf("A children = %d, want 2", len(a.Children()))
	}
}

func TestImportResolution(t *testing.T) {
	r := Analyze("package A { part def X; }\npackage B { import A::X; }")
	if len(r.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %v, want none", r.Diagnostics)
	}
	if len(r.Names) != 1 {
		t.Fatalf("names = %d, want 1", len(r.Names))
	}
	ref := r.Names[0]
	if !ref.Resolved || ref.Target != "A::X" || ref.Path != "A::X" {
		t.Errorf("ref = %+v, want resolved A::X", ref)
	}
}

func TestUnresolvedImportDiagnostic(t *testing.T) {
	r := Analyze("package B { import A::Missing; }")
	if len(r.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %d, want 1 (%v)", len(r.Diagnostics), r.Diagnostics)
	}
	if r.Diagnostics[0].Message != "unresolved import 'A::Missing'" {
		t.Errorf("message = %q", r.Diagnostics[0].Message)
	}
	if r.Names[0].Resolved {
		t.Error("ref should be unresolved")
	}
}

func TestWildcardImportRef(t *testing.T) {
	r := Analyze("package A { part def X; }\npackage B { import A::*; }")
	if len(r.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", r.Diagnostics)
	}
	ref := r.Names[0]
	if ref.Path != "A::*" || !ref.Resolved || ref.Target != "A" {
		t.Errorf("wildcard ref = %+v, want A::* → A", ref)
	}
}

func TestResolveSimpleAndQualified(t *testing.T) {
	r := Analyze("package A {\n\tpart def X;\n}\npackage B {\n\timport A::X;\n\tfeature f;\n}")
	m := r.Model
	b := mustMember(t, m.Root, "B")

	// A local name resolves in its own scope.
	if got := m.Resolve(b, []string{"f"}); got == nil || got.Name != "f" {
		t.Errorf("resolve f = %v, want feature f", got)
	}
	// X resolves through B's `import A::X`.
	if got := m.Resolve(b, []string{"X"}); got == nil || got.QualifiedName() != "A::X" {
		t.Errorf("resolve X via import = %v, want A::X", got)
	}
	// Absolute qualified name A::X resolves from any scope.
	if got := m.Resolve(b, []string{"A", "X"}); got == nil || got.QualifiedName() != "A::X" {
		t.Errorf("resolve A::X = %v, want A::X", got)
	}
	// A name that isn't in scope stays unresolved.
	if got := m.Resolve(b, []string{"Nope"}); got != nil {
		t.Errorf("resolve Nope = %v, want nil", got)
	}
}

func TestResolveViaWildcardAndRecursiveImports(t *testing.T) {
	r := Analyze("package A {\n\tpart def X;\n\tpart def Y;\n}\npackage B {\n\timport A::*;\n}")
	m := r.Model
	b := mustMember(t, m.Root, "B")
	if got := m.Resolve(b, []string{"Y"}); got == nil || got.QualifiedName() != "A::Y" {
		t.Errorf("wildcard resolve Y = %v, want A::Y", got)
	}

	r2 := Analyze("package A {\n\tpackage Inner {\n\t\tpart def Deep;\n\t}\n}\npackage B {\n\timport A::**;\n}")
	m2 := r2.Model
	b2 := mustMember(t, m2.Root, "B")
	if got := m2.Resolve(b2, []string{"Deep"}); got == nil || got.QualifiedName() != "A::Inner::Deep" {
		t.Errorf("recursive resolve Deep = %v, want A::Inner::Deep", got)
	}
}

func TestIncrementalDb(t *testing.T) {
	db := NewDb()

	db.SetSource("f", "package B { import A::Missing; }")
	if d := db.Diagnostics("f"); len(d) != 1 {
		t.Fatalf("diagnostics = %d, want 1", len(d))
	}
	if n := db.Names("f"); len(n) != 1 || n[0].Resolved {
		t.Fatalf("names = %+v, want one unresolved", n)
	}

	// Fix the import: diagnostics clear.
	db.SetSource("f", "package A { part def Missing; }\npackage B { import A::Missing; }")
	if d := db.Diagnostics("f"); len(d) != 0 {
		t.Fatalf("after fix diagnostics = %d (%v), want 0", len(d), d)
	}
	if n := db.Names("f"); len(n) != 1 || !n[0].Resolved || n[0].Target != "A::Missing" {
		t.Fatalf("after fix names = %+v, want resolved A::Missing", n)
	}

	// A second file is analyzed independently.
	db.SetSource("g", "package Z { part def W; }")
	if d := db.Diagnostics("g"); len(d) != 0 {
		t.Errorf("file g diagnostics = %v, want none", d)
	}
}
