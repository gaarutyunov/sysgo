package engine

import (
	"reflect"
	"testing"
)

const kwModel = `package M {
	import ScalarValues::*;
	part def Vehicle {
		attribute mass : Real;
	}
	action def PlaceOrder {
		in order : Vehicle;
		out receipt : Vehicle;
	}
	abstract classifier Base;
}`

func kwBuild(t *testing.T) *Model {
	t.Helper()
	m := New().AddFile("m.sysml", kwModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	return m
}

func TestDeclarationKeyword(t *testing.T) {
	m := kwBuild(t)
	cases := map[string]string{
		"M::Vehicle":       "part",
		"M::Vehicle::mass": "attribute",
		"M::PlaceOrder":    "action",
		"M::Base":          "classifier",
	}
	for path, wantKw := range cases {
		e, ok := m.Lookup(path)
		if !ok {
			t.Fatalf("lookup %s failed", path)
		}
		if got := e.DeclarationKeyword(); got != wantKw {
			t.Errorf("%s DeclarationKeyword = %q, want %q", path, got, wantKw)
		}
	}
}

func TestDirection(t *testing.T) {
	m := kwBuild(t)
	pl, _ := m.Lookup("M::PlaceOrder")

	order, ok := pl.Member("order")
	if !ok {
		t.Fatal("no order param")
	}
	if order.Direction() != "in" {
		t.Errorf("order direction = %q, want in", order.Direction())
	}
	// A directed usage has a direction but no kind noun.
	if order.DeclarationKeyword() != "" {
		t.Errorf("order DeclarationKeyword = %q, want empty", order.DeclarationKeyword())
	}

	receipt, _ := pl.Member("receipt")
	if receipt.Direction() != "out" {
		t.Errorf("receipt direction = %q, want out", receipt.Direction())
	}

	// A definition has no direction.
	if pl.Direction() != "" {
		t.Errorf("PlaceOrder direction = %q, want empty", pl.Direction())
	}
}

func TestKeywordsList(t *testing.T) {
	m := kwBuild(t)

	base, _ := m.Lookup("M::Base")
	if got := base.Keywords(); !reflect.DeepEqual(got, []string{"abstract", "classifier"}) {
		t.Errorf("Base keywords = %v, want [abstract classifier]", got)
	}
	pl, _ := m.Lookup("M::PlaceOrder")
	if got := pl.Keywords(); !reflect.DeepEqual(got, []string{"action", "def"}) {
		t.Errorf("PlaceOrder keywords = %v, want [action def]", got)
	}
	order, _ := pl.Member("order")
	if got := order.Keywords(); !reflect.DeepEqual(got, []string{"in"}) {
		t.Errorf("order keywords = %v, want [in]", got)
	}
}
