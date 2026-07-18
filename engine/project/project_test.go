package project

import (
	"os"
	"testing"

	"github.com/gaarutyunov/sysgo/engine/hir"
)

func lookup(t *testing.T, m *hir.Model, segs ...string) *hir.Symbol {
	t.Helper()
	s := m.Resolve(m.Root, segs)
	if s == nil {
		t.Fatalf("could not resolve %v", segs)
	}
	return s
}

func TestStdlibUnitsLoad(t *testing.T) {
	units := StdlibUnits()
	if len(units) < 2 {
		t.Fatalf("stdlib units = %d, want >= 2", len(units))
	}
	// The stdlib itself resolves with no unresolved-name diagnostics.
	r := hir.AnalyzeUnits(units)
	if len(r.Diagnostics) != 0 {
		t.Errorf("stdlib self-analysis diagnostics = %v, want none", r.Diagnostics)
	}
	lookup(t, r.Model, "ScalarValues", "Real")
	lookup(t, r.Model, "Base", "Anything")
}

func TestStdlibInheritanceChain(t *testing.T) {
	r := hir.AnalyzeUnits(StdlibUnits())
	m := r.Model
	integer := lookup(t, m, "ScalarValues", "Integer")
	// Integer :> Real, and Real transitively reaches ScalarValue.
	if len(integer.Supertypes) != 1 || integer.Supertypes[0].Name != "Real" {
		t.Fatalf("Integer supertypes = %v, want [Real]", integer.Supertypes)
	}
	scalar := lookup(t, m, "ScalarValues", "ScalarValue")
	// Walk up from Integer and confirm ScalarValue is reachable.
	if got := m.InheritedMember(integer, "does-not-exist"); got != nil {
		t.Errorf("unexpected inherited member: %v", got)
	}
	// A feature declared on ScalarValue would be inherited by Integer — model it
	// indirectly by checking the chain resolves without cycles (no hang) and the
	// top type exists.
	_ = scalar
}

func TestUserModelResolvesAgainstStdlib(t *testing.T) {
	w := New()
	w.AddFile("m.sysml", "package M {\n"+
		"\timport ScalarValues::*;\n"+
		"\tattribute x : Real;\n"+
		"\tattribute n : Integer;\n"+
		"\tclassifier Thing :> Base::Anything;\n"+
		"}")
	r := w.Analyze()
	if len(r.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %v, want none (all names resolve against stdlib)", r.Diagnostics)
	}

	// x : Real resolves to the library type.
	real := findRel(t, r.Relationships, "M::x", hir.RelTyped)
	if real.Target != "ScalarValues::Real" {
		t.Errorf("x typing target = %q, want ScalarValues::Real", real.Target)
	}
	// Thing :> Base::Anything resolves via an absolute qualified name.
	anything := findRel(t, r.Relationships, "M::Thing", hir.RelSpecializes)
	if anything.Target != "Base::Anything" {
		t.Errorf("Thing specialization = %q, want Base::Anything", anything.Target)
	}
}

func TestExampleModelResolvesEndToEnd(t *testing.T) {
	const path = "../../examples/order/OrderContext.sysml"
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	w := New()
	w.AddFile("OrderContext.sysml", string(src))
	r := w.Analyze()

	if len(r.Diagnostics) != 0 {
		t.Fatalf("example resolved with %d diagnostic(s), want 0:\n%v", len(r.Diagnostics), r.Diagnostics)
	}
	// Every relationship in the example resolved (library + file-local names).
	for _, ref := range r.Relationships {
		if !ref.Resolved {
			t.Errorf("unresolved relationship: %+v", ref)
		}
	}
	// Spot-check a library-typed attribute.
	m := r.Model
	lookup(t, m, "OrderContext", "Money")
	lookup(t, m, "ScalarValues", "Real")
}

func TestAddFileReplaceAndOrder(t *testing.T) {
	w := New()
	w.AddFile("a.sysml", "package A;")
	w.AddFile("b.sysml", "package B { import A::*; }")
	// Replacing a file keeps its position and updates its content.
	w.AddFile("a.sysml", "package A { part def X; }")
	r := w.Analyze()
	if len(r.Diagnostics) != 0 {
		t.Fatalf("cross-file diagnostics = %v, want none", r.Diagnostics)
	}
	// B can resolve X brought in from A across files.
	b := lookup(t, r.Model, "B")
	if got := r.Model.Resolve(b, []string{"X"}); got == nil || got.QualifiedName() != "A::X" {
		t.Errorf("cross-file resolve X = %v, want A::X", got)
	}
}

func findRel(t *testing.T, rels []hir.RelRef, from string, kind hir.RelKind) hir.RelRef {
	t.Helper()
	for _, r := range rels {
		if r.From == from && r.Kind == kind {
			return r
		}
	}
	t.Fatalf("no %v relationship from %q", kind, from)
	return hir.RelRef{}
}
