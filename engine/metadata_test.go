package engine

import "testing"

const apiModel = `package API {
	import ScalarValues::*;
	import RESTProfile::*;
	item def Order {
		attribute id : String;
	}
	@REST { path = "/orders"; method = "POST"; successStatus = 201; }
	action placeOrder {
		in order : Order;
	}
}`

func TestElementMetadata(t *testing.T) {
	m := New().AddFile("api.sysml", apiModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model has diagnostics: %v", d)
	}
	op, ok := m.Lookup("API::placeOrder")
	if !ok {
		t.Fatal("placeOrder not found")
	}

	meta, ok := op.Metadata("REST")
	if !ok {
		t.Fatal("placeOrder has no REST metadata")
	}
	if v, _ := meta.Value("path"); v != `"/orders"` {
		t.Errorf("path = %q, want \"/orders\"", v)
	}
	if v, _ := meta.Value("method"); v != `"POST"` {
		t.Errorf("method = %q, want \"POST\"", v)
	}
	if v, _ := meta.Value("successStatus"); v != "201" {
		t.Errorf("successStatus = %q, want 201", v)
	}
	if _, ok := meta.Value("missing"); ok {
		t.Error("missing key should be absent")
	}
	// Keys preserve source order.
	if got := meta.Keys; len(got) != 3 || got[0] != "path" || got[2] != "successStatus" {
		t.Errorf("keys = %v, want [path method successStatus]", got)
	}

	if len(op.Annotations()) != 1 {
		t.Errorf("annotations = %d, want 1", len(op.Annotations()))
	}
	if _, ok := op.Metadata("Api"); ok {
		t.Error("placeOrder has no Api metadata")
	}

	// The operation's in-param is exposed as a child usage.
	order, ok := op.Member("order")
	if !ok || order.Kind() != ElementUsage {
		t.Errorf("order param = %+v (%v), want a usage", order, ok)
	}
}

func TestElementWithoutMetadata(t *testing.T) {
	m := New().AddFile("m.sysml", "package M { part def X; }").Build()
	x, _ := m.Lookup("M::X")
	if len(x.Annotations()) != 0 {
		t.Errorf("annotations = %d, want 0", len(x.Annotations()))
	}
	if _, ok := x.Metadata("REST"); ok {
		t.Error("X has no REST metadata")
	}
}
