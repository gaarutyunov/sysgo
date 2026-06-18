package overlay

import "testing"

func elements() []map[string]any {
	return []map[string]any{
		{"@id": "1", "@type": "PartDefinition", "declaredName": "Order"},
		{"@id": "2", "@type": "AttributeUsage", "declaredName": "price"},
		{"@id": "3", "@type": "PartDefinition", "declaredName": "Internal", "x-internal": true},
	}
}

func TestUpdateByFilter(t *testing.T) {
	eng, err := Parse([]byte(`
overlay: 1.0.0
info: { title: t, version: 1.0.0 }
actions:
  - target: $[?(@.declaredName=='price')]
    update: { x-go-type: "money.Money", x-go-type-import: "github.com/acme/money" }
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := eng.Apply(elements())
	if err != nil {
		t.Fatal(err)
	}
	if out[1]["x-go-type"] != "money.Money" {
		t.Fatalf("update not applied: %+v", out[1])
	}
	if out[0]["x-go-type"] != nil {
		t.Fatalf("update leaked to non-matching element")
	}
}

func TestUpdateCompoundFilter(t *testing.T) {
	eng, _ := Parse([]byte(`
actions:
  - target: $[?(@.declaredName=='Order' && @['@type']=='PartDefinition')]
    update: { x-ddd-stereotype: aggregate }
`))
	out, _ := eng.Apply(elements())
	if out[0]["x-ddd-stereotype"] != "aggregate" {
		t.Fatalf("compound filter failed: %+v", out[0])
	}
}

func TestRemove(t *testing.T) {
	eng, _ := Parse([]byte(`
actions:
  - target: $[?(@['@type']=='PartDefinition' && @.x-internal==true)]
    remove: true
`))
	out, _ := eng.Apply(elements())
	if len(out) != 2 {
		t.Fatalf("expected 2 remaining, got %d", len(out))
	}
	for _, e := range out {
		if e["declaredName"] == "Internal" {
			t.Fatal("Internal element not removed")
		}
	}
}

func TestSelectorSugar(t *testing.T) {
	eng, _ := Parse([]byte(`
actions:
  - target: byName:Order
    update: { x-ddd-stereotype: aggregate }
  - target: byType:AttributeUsage
    update: { x-go-name: Renamed }
`))
	out, _ := eng.Apply(elements())
	if out[0]["x-ddd-stereotype"] != "aggregate" {
		t.Fatalf("byName failed")
	}
	if out[1]["x-go-name"] != "Renamed" {
		t.Fatalf("byType failed")
	}
}

func TestCopy(t *testing.T) {
	eng, _ := Parse([]byte(`
actions:
  - target: byName:Order
    copy: dup
`))
	out, _ := eng.Apply(elements())
	if len(out) != 4 {
		t.Fatalf("expected 4 after copy, got %d", len(out))
	}
	if out[3]["@id"] != "1:dup" {
		t.Fatalf("copy id = %v", out[3]["@id"])
	}
}
