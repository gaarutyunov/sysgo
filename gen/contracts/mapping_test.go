package contracts

import (
	"reflect"
	"sort"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const model = `package M {
	import ScalarValues::*;
	attribute def Money {
		attribute amount : Real;
		attribute currency : String;
	}
	item def Receipt :> Money {
		attribute id : String;
		attribute count : Integer;
	}
	item def Order {
		attribute total : Money;
		attribute paid : Boolean;
	}
	item def Node {
		attribute next : Node;
	}
}`

func build(t *testing.T) *engine.Model {
	t.Helper()
	m := engine.New().AddFile("m.sysml", model).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model has diagnostics: %v", d)
	}
	return m
}

func el(t *testing.T, m *engine.Model, qn string) engine.Element {
	t.Helper()
	e, ok := m.Lookup(qn)
	if !ok {
		t.Fatalf("lookup %q failed", qn)
	}
	return e
}

func propNames(s *Schema) []string {
	out := make([]string, 0, len(s.Properties))
	for k := range s.Properties {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestScalarProperties(t *testing.T) {
	m := build(t)
	s := SchemaFor(el(t, m, "M::Money"))

	if s.Type != "object" {
		t.Fatalf("type = %q, want object", s.Type)
	}
	if got := propNames(s); !reflect.DeepEqual(got, []string{"amount", "currency"}) {
		t.Fatalf("properties = %v, want [amount currency]", got)
	}
	if s.Properties["amount"].Type != "number" {
		t.Errorf("amount type = %q, want number", s.Properties["amount"].Type)
	}
	if s.Properties["currency"].Type != "string" {
		t.Errorf("currency type = %q, want string", s.Properties["currency"].Type)
	}
	if !reflect.DeepEqual(s.Required, []string{"amount", "currency"}) {
		t.Errorf("required = %v, want [amount currency]", s.Required)
	}
}

func TestScalarKinds(t *testing.T) {
	m := build(t)
	r := SchemaFor(el(t, m, "M::Receipt"))
	if r.Properties["count"].Type != "integer" {
		t.Errorf("count type = %q, want integer", r.Properties["count"].Type)
	}
	o := SchemaFor(el(t, m, "M::Order"))
	if o.Properties["paid"].Type != "boolean" {
		t.Errorf("paid type = %q, want boolean", o.Properties["paid"].Type)
	}
}

func TestSpecializationFlattened(t *testing.T) {
	m := build(t)
	s := SchemaFor(el(t, m, "M::Receipt"))

	// Own (id, count) + inherited from Money (amount, currency), all inlined.
	want := []string{"amount", "count", "currency", "id"}
	if got := propNames(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("flattened properties = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(s.Required, want) {
		t.Errorf("required = %v, want %v", s.Required, want)
	}
	// Inherited scalar types are correct.
	if s.Properties["amount"].Type != "number" || s.Properties["currency"].Type != "string" {
		t.Error("inherited attribute types wrong")
	}
}

func TestNestedObjectInlined(t *testing.T) {
	m := build(t)
	s := SchemaFor(el(t, m, "M::Order"))

	total := s.Properties["total"]
	if total == nil || total.Type != "object" {
		t.Fatalf("total = %+v, want an object schema", total)
	}
	if got := propNames(total); !reflect.DeepEqual(got, []string{"amount", "currency"}) {
		t.Errorf("nested Money props = %v, want [amount currency]", got)
	}
}

func TestCycleSafe(t *testing.T) {
	m := build(t)
	s := SchemaFor(el(t, m, "M::Node"))
	next := s.Properties["next"]
	if next == nil || next.Type != "object" {
		t.Fatalf("next = %+v, want a bare object (cycle stop)", next)
	}
	if len(next.Properties) != 0 {
		t.Errorf("cycle stop should not re-expand: %+v", next.Properties)
	}
}

func TestJSONDeterministicAndGolden(t *testing.T) {
	m := build(t)
	money := el(t, m, "M::Money")
	a := SchemaFor(money).JSON()
	b := SchemaFor(money).JSON()
	if a != b {
		t.Fatalf("JSON not deterministic:\n%s\nvs\n%s", a, b)
	}
	want := `{
  "type": "object",
  "properties": {
    "amount": {
      "type": "number"
    },
    "currency": {
      "type": "string"
    }
  },
  "required": [
    "amount",
    "currency"
  ]
}
`
	if a != want {
		t.Errorf("JSON mismatch:\n got:\n%s\nwant:\n%s", a, want)
	}
}
