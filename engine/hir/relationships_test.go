package hir

import "testing"

func findRel(rels []RelRef, from string, kind RelKind) (RelRef, bool) {
	for _, r := range rels {
		if r.From == from && r.Kind == kind {
			return r, true
		}
	}
	return RelRef{}, false
}

func TestSpecializationResolves(t *testing.T) {
	r := Analyze("classifier Vehicle;\nclassifier Car :> Vehicle;")
	car := mustMember(t, r.Model.Root, "Car")
	if len(car.Supertypes) != 1 || car.Supertypes[0].Name != "Vehicle" {
		t.Fatalf("Car supertypes = %v, want [Vehicle]", car.Supertypes)
	}
	ref, ok := findRel(r.Relationships, "Car", RelSpecializes)
	if !ok || !ref.Resolved || ref.Target != "Vehicle" {
		t.Errorf("specialization ref = %+v, want resolved Vehicle", ref)
	}
	if len(r.Diagnostics) != 0 {
		t.Errorf("unexpected diagnostics: %v", r.Diagnostics)
	}
}

func TestInheritedMemberThroughChain(t *testing.T) {
	// A :> B :> C, and C declares `deep`; A inherits it transitively.
	r := Analyze("classifier C {\n\tfeature deep;\n}\nclassifier B :> C;\nclassifier A :> B;")
	m := r.Model
	a := mustMember(t, m.Root, "A")

	if got := m.InheritedMember(a, "deep"); got == nil || got.QualifiedName() != "C::deep" {
		t.Errorf("InheritedMember(A, deep) = %v, want C::deep", got)
	}
	// Not a direct member, but LookupMember finds it via inheritance.
	if _, ok := a.Member("deep"); ok {
		t.Error("deep should not be a direct member of A")
	}
	if got := m.LookupMember(a, "deep"); got == nil || got.QualifiedName() != "C::deep" {
		t.Errorf("LookupMember(A, deep) = %v, want C::deep", got)
	}
	// A name nobody declares stays unresolved.
	if got := m.InheritedMember(a, "nope"); got != nil {
		t.Errorf("InheritedMember(A, nope) = %v, want nil", got)
	}
}

func TestRedefinitionOfInheritedFeature(t *testing.T) {
	r := Analyze("classifier Base {\n\tfeature mass;\n}\nclassifier Car :> Base {\n\tfeature m :>> mass;\n}")
	ref, ok := findRel(r.Relationships, "Car::m", RelRedefines)
	if !ok {
		t.Fatalf("no redefines ref for Car::m: %+v", r.Relationships)
	}
	if !ref.Resolved || ref.Target != "Base::mass" {
		t.Errorf("redefines ref = %+v, want resolved Base::mass", ref)
	}
	if len(r.Diagnostics) != 0 {
		t.Errorf("unexpected diagnostics: %v", r.Diagnostics)
	}
}

func TestFeatureTypingResolves(t *testing.T) {
	r := Analyze("classifier Real;\nclassifier Car {\n\tattribute mass : Real;\n}")
	ref, ok := findRel(r.Relationships, "Car::mass", RelTyped)
	if !ok || !ref.Resolved || ref.Target != "Real" {
		t.Errorf("typing ref = %+v, want resolved Real", ref)
	}
}

func TestUnresolvedRelationshipDiagnostic(t *testing.T) {
	r := Analyze("classifier Car :> Missing;")
	if len(r.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %d, want 1 (%v)", len(r.Diagnostics), r.Diagnostics)
	}
	if r.Diagnostics[0].Message != "unresolved specializes target 'Missing'" {
		t.Errorf("message = %q", r.Diagnostics[0].Message)
	}
	ref, _ := findRel(r.Relationships, "Car", RelSpecializes)
	if ref.Resolved {
		t.Error("ref should be unresolved")
	}
}

func TestInheritanceCycleIsSafe(t *testing.T) {
	// A :> B and B :> A — resolution must terminate.
	r := Analyze("classifier A :> B;\nclassifier B :> A;")
	m := r.Model
	a := mustMember(t, m.Root, "A")
	if got := m.InheritedMember(a, "whatever"); got != nil {
		t.Errorf("cyclic InheritedMember = %v, want nil", got)
	}
	// Both specializations still resolve (to each other).
	if len(r.Relationships) != 2 {
		t.Errorf("relationships = %d, want 2", len(r.Relationships))
	}
	for _, ref := range r.Relationships {
		if !ref.Resolved {
			t.Errorf("ref %+v should be resolved", ref)
		}
	}
}

func TestRelationshipsIncrementalDb(t *testing.T) {
	db := NewDb()

	db.SetSource("f", "classifier Car :> Missing;")
	if rels := db.Relationships("f"); len(rels) != 1 || rels[0].Resolved {
		t.Fatalf("relationships = %+v, want one unresolved", rels)
	}
	if d := db.Diagnostics("f"); len(d) != 1 {
		t.Fatalf("diagnostics = %d, want 1", len(d))
	}

	db.SetSource("f", "classifier Vehicle;\nclassifier Car :> Vehicle;")
	rels := db.Relationships("f")
	if len(rels) != 1 || !rels[0].Resolved || rels[0].Target != "Vehicle" {
		t.Fatalf("after fix relationships = %+v, want resolved Vehicle", rels)
	}
	if d := db.Diagnostics("f"); len(d) != 0 {
		t.Errorf("after fix diagnostics = %v, want none", d)
	}
}
