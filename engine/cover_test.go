package engine

import "testing"

func TestElementKindStrings(t *testing.T) {
	cases := map[ElementKind]string{
		ElementRoot:       "Root",
		ElementPackage:    "Package",
		ElementDefinition: "Definition",
		ElementUsage:      "Usage",
		ElementType:       "Type",
		ElementKind(99):   "Type",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("ElementKind(%d).String() = %q, want %q", k, got, want)
		}
	}
}

func TestRelationshipKindStrings(t *testing.T) {
	cases := map[RelationshipKind]string{
		Specializes:          "specializes",
		Subsets:              "subsets",
		Redefines:            "redefines",
		FeatureTyping:        "typed",
		Conjugates:           "conjugates",
		RelationshipKind(99): "relationship",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("RelationshipKind(%d).String() = %q, want %q", k, got, want)
		}
	}
}

func TestAllRelationshipKindsResolve(t *testing.T) {
	src := `package M {
	classifier Base;
	feature g;
	classifier C :> Base;
	feature a subsets g;
	feature b redefines g;
	feature c : Base;
	feature d ~ Base;
}`
	m := New().AddFile("m.sysml", src).Build()

	want := map[string]RelationshipKind{
		"M::C": Specializes,
		"M::a": Subsets,
		"M::b": Redefines,
		"M::c": FeatureTyping,
		"M::d": Conjugates,
	}
	for path, kind := range want {
		el, ok := m.Lookup(path)
		if !ok {
			t.Fatalf("lookup %s failed", path)
		}
		rels := el.Relationships()
		if len(rels) != 1 || rels[0].Kind() != kind {
			t.Errorf("%s relationships = %v, want one %v", path, rels, kind)
		}
	}
}

func TestElementKindsCovered(t *testing.T) {
	m := New().AddFile("m.sysml", "package P {\n\tpart def D;\n\tpart u : D;\n\tclassifier T;\n}").Build()
	p, _ := m.Lookup("P")
	if p.Kind() != ElementPackage {
		t.Errorf("P kind = %v", p.Kind())
	}
	if d, _ := m.Lookup("P::D"); d.Kind() != ElementDefinition {
		t.Errorf("D kind wrong")
	}
	if u, _ := m.Lookup("P::u"); u.Kind() != ElementUsage {
		t.Errorf("u kind wrong")
	}
	if tt, _ := m.Lookup("P::T"); tt.Kind() != ElementType {
		t.Errorf("T kind wrong")
	}
	// Root is the implicit global namespace.
	if m.Root().Kind() != ElementRoot {
		t.Errorf("root kind = %v, want Root", m.Root().Kind())
	}
}

func TestZeroElementInvalidAndRange(t *testing.T) {
	var e Element
	if e.IsValid() {
		t.Error("zero Element should be invalid")
	}
	m := New().AddFile("m.sysml", "package P;").Build()
	p, _ := m.Lookup("P")
	if !p.IsValid() {
		t.Error("resolved element should be valid")
	}
	if p.Range().End == 0 {
		t.Error("element range should be set")
	}
}

func TestLookupEmpty(t *testing.T) {
	m := New().AddFile("m.sysml", "package P;").Build()
	if _, ok := m.Lookup(""); ok {
		t.Error("empty lookup should fail")
	}
}
