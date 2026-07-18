package engine

import (
	"os"
	"testing"
)

const sample = `package M {
	import ScalarValues::*;
	part def Vehicle {
		attribute mass : Real;
	}
	part def Car :> Vehicle;
}`

func build(t *testing.T, src string) *Model {
	t.Helper()
	m := New().AddFile("m.sysml", src).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("unexpected diagnostics: %v", d)
	}
	return m
}

func TestModelRootAndLookup(t *testing.T) {
	m := build(t, sample)

	pkg, ok := m.Lookup("M")
	if !ok || pkg.Kind() != ElementPackage || pkg.Name() != "M" {
		t.Fatalf("lookup M = %+v (%v)", pkg, ok)
	}
	car, ok := m.Lookup("M::Car")
	if !ok || car.Kind() != ElementDefinition || car.QualifiedName() != "M::Car" {
		t.Fatalf("lookup M::Car = %+v (%v)", car, ok)
	}
	// A library type resolves through the auto-loaded stdlib.
	if _, ok := m.Lookup("ScalarValues::Real"); !ok {
		t.Error("ScalarValues::Real not found")
	}
	if _, ok := m.Lookup("Nope::Missing"); ok {
		t.Error("unexpected lookup success for Nope::Missing")
	}
}

func TestChildrenAndMembers(t *testing.T) {
	m := build(t, sample)
	pkg, _ := m.Lookup("M")
	// M declares: Vehicle, Car (the import is not a named element).
	names := map[string]bool{}
	for _, c := range pkg.Children() {
		names[c.Name()] = true
	}
	if !names["Vehicle"] || !names["Car"] {
		t.Errorf("M children = %v, want Vehicle and Car", names)
	}
	vehicle, ok := pkg.Member("Vehicle")
	if !ok {
		t.Fatal("no Vehicle member")
	}
	if _, ok := vehicle.Member("mass"); !ok {
		t.Error("Vehicle has no mass member")
	}
}

func TestSupertypesAndRelationships(t *testing.T) {
	m := build(t, sample)
	car, _ := m.Lookup("M::Car")

	sts := car.Supertypes()
	if len(sts) != 1 || sts[0].QualifiedName() != "M::Vehicle" {
		t.Fatalf("Car supertypes = %v, want [M::Vehicle]", sts)
	}

	rels := car.Relationships()
	if len(rels) != 1 || rels[0].Kind() != Specializes {
		t.Fatalf("Car relationships = %v, want one Specializes", rels)
	}
	tgt, ok := rels[0].Target()
	if !ok || tgt.QualifiedName() != "M::Vehicle" {
		t.Errorf("relationship target = %+v (%v), want M::Vehicle", tgt, ok)
	}

	// mass : Real — feature typing to a library element.
	vehicle, _ := m.Lookup("M::Vehicle")
	mass, _ := vehicle.Member("mass")
	massRels := mass.Relationships()
	if len(massRels) != 1 || massRels[0].Kind() != FeatureTyping {
		t.Fatalf("mass relationships = %v, want one FeatureTyping", massRels)
	}
	if tn := massRels[0].TargetName(); tn != "Real" {
		t.Errorf("mass typing target name = %q, want Real", tn)
	}
	if tgt, ok := massRels[0].Target(); !ok || tgt.QualifiedName() != "ScalarValues::Real" {
		t.Errorf("mass typing target = %+v (%v), want ScalarValues::Real", tgt, ok)
	}
}

func TestInheritedMember(t *testing.T) {
	m := build(t, sample)
	car, _ := m.Lookup("M::Car")
	// mass is declared on Vehicle; Car inherits it via specialization.
	if _, ok := car.Member("mass"); ok {
		t.Error("mass should not be a direct member of Car")
	}
	got, ok := car.InheritedMember("mass")
	if !ok || got.QualifiedName() != "M::Vehicle::mass" {
		t.Errorf("Car.InheritedMember(mass) = %+v (%v), want M::Vehicle::mass", got, ok)
	}
	if lm, ok := car.LookupMember("mass"); !ok || lm.QualifiedName() != "M::Vehicle::mass" {
		t.Errorf("Car.LookupMember(mass) = %+v (%v)", lm, ok)
	}
}

func TestDiagnosticsSurfaceUnresolved(t *testing.T) {
	m := New().AddFile("m.sysml", "part def Car :> Missing;").Build()
	d := m.Diagnostics()
	if len(d) != 1 {
		t.Fatalf("diagnostics = %d, want 1 (%v)", len(d), d)
	}
	// The relationship is present but unresolved.
	car, _ := m.Lookup("Car")
	rels := car.Relationships()
	if len(rels) != 1 || rels[0].IsResolved() {
		t.Errorf("relationship should be unresolved: %+v", rels)
	}
	if _, ok := rels[0].Target(); ok {
		t.Error("unresolved relationship should have no target")
	}
}

func TestExampleModelViaPublicAPI(t *testing.T) {
	src, err := os.ReadFile("../examples/order/OrderContext.sysml")
	if err != nil {
		// The engine package sits at engine/, so the example is one level up.
		src, err = os.ReadFile("examples/order/OrderContext.sysml")
		if err != nil {
			t.Skipf("example model not found: %v", err)
		}
	}
	m := New().AddFile("OrderContext.sysml", string(src)).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("example resolved with diagnostics: %v", d)
	}
	oc, ok := m.Lookup("OrderContext")
	if !ok || oc.Kind() != ElementPackage {
		t.Fatalf("OrderContext = %+v (%v)", oc, ok)
	}
	if _, ok := oc.Member("Order"); !ok {
		t.Error("OrderContext has no Order definition")
	}
}
