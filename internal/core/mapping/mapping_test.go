package mapping

import (
	"testing"

	"github.com/gaarutyunov/sysgo/internal/config"
	"github.com/gaarutyunov/sysgo/internal/core/model"
)

func baseCfg() *config.Config {
	c := config.Default()
	c.Module = "example.com/m"
	c.Source.File = "x"
	return c
}

func TestAggregateHeuristic(t *testing.T) {
	raw := []map[string]any{
		{"@id": "p", "@type": "Package", "declaredName": "Ctx",
			"ownedElement": []any{map[string]any{"@id": "order"}}},
		{"@id": "order", "@type": "PartDefinition", "declaredName": "Order",
			"ownedElement": []any{map[string]any{"@id": "id"}}},
		{"@id": "id", "@type": "AttributeUsage", "declaredName": "id", "type": "String"},
	}
	m := New(baseCfg())
	g, _ := model.Build(raw)
	proj, _ := m.Build(g)
	ctx := proj.Contexts[0]
	if len(ctx.Entities) != 1 || !ctx.Entities[0].Aggregate {
		t.Fatalf("expected one aggregate, got %+v", ctx.Entities)
	}
	if ctx.Entities[0].Fields[0].GoType != "string" {
		t.Fatalf("scalar mapping failed: %+v", ctx.Entities[0].Fields[0])
	}
}

func TestValueObjectHeuristic(t *testing.T) {
	raw := []map[string]any{
		{"@id": "p", "@type": "Package", "declaredName": "Ctx",
			"ownedElement": []any{map[string]any{"@id": "money"}}},
		{"@id": "money", "@type": "PartDefinition", "declaredName": "Money"},
	}
	g, _ := model.Build(raw)
	proj, _ := New(baseCfg()).Build(g)
	if len(proj.Contexts[0].ValueObjects) != 1 {
		t.Fatalf("expected value object, got %+v", proj.Contexts[0])
	}
}

func TestStereotypeOverride(t *testing.T) {
	raw := []map[string]any{
		{"@id": "p", "@type": "Package", "declaredName": "Ctx",
			"ownedElement": []any{map[string]any{"@id": "x"}}},
		// Has an id attribute (would be aggregate) but overridden to value object.
		{"@id": "x", "@type": "PartDefinition", "declaredName": "Coord",
			"x-ddd-stereotype": "value-object",
			"ownedElement":     []any{map[string]any{"@id": "xid"}}},
		{"@id": "xid", "@type": "AttributeUsage", "declaredName": "id", "type": "String"},
	}
	g, _ := model.Build(raw)
	proj, _ := New(baseCfg()).Build(g)
	ctx := proj.Contexts[0]
	if len(ctx.Entities) != 0 || len(ctx.ValueObjects) != 1 {
		t.Fatalf("stereotype override ignored: %+v", ctx)
	}
}

func TestTypeMappingOverride(t *testing.T) {
	cfg := baseCfg()
	cfg.TypeMapping = map[string]config.TypeMap{
		"Money": {Type: "money.Money", Import: "github.com/acme/money"},
	}
	raw := []map[string]any{
		{"@id": "p", "@type": "Package", "declaredName": "Ctx",
			"ownedElement": []any{map[string]any{"@id": "order"}}},
		{"@id": "order", "@type": "PartDefinition", "declaredName": "Order",
			"ownedElement": []any{map[string]any{"@id": "id"}, map[string]any{"@id": "total"}}},
		{"@id": "id", "@type": "AttributeUsage", "declaredName": "id", "type": "String"},
		{"@id": "total", "@type": "AttributeUsage", "declaredName": "total", "type": "Money"},
	}
	g, _ := model.Build(raw)
	proj, _ := New(cfg).Build(g)
	var total *string
	for _, f := range proj.Contexts[0].Entities[0].Fields {
		if f.Name == "Total" {
			total = &f.GoType
		}
	}
	if total == nil || *total != "money.Money" {
		t.Fatalf("type mapping not applied: %v", total)
	}
}

func TestOptionalPointerAndMany(t *testing.T) {
	raw := []map[string]any{
		{"@id": "p", "@type": "Package", "declaredName": "Ctx",
			"ownedElement": []any{map[string]any{"@id": "o"}}},
		{"@id": "o", "@type": "PartDefinition", "declaredName": "Order",
			"ownedElement": []any{map[string]any{"@id": "id"}, map[string]any{"@id": "notes"}, map[string]any{"@id": "lines"}}},
		{"@id": "id", "@type": "AttributeUsage", "declaredName": "id", "type": "String"},
		{"@id": "notes", "@type": "AttributeUsage", "declaredName": "notes", "type": "String", "lowerBound": float64(0)},
		{"@id": "lines", "@type": "PartUsage", "declaredName": "lines", "type": "Line", "upperBound": "*"},
	}
	g, _ := model.Build(raw)
	proj, _ := New(baseCfg()).Build(g)
	fields := map[string]string{}
	for _, f := range proj.Contexts[0].Entities[0].Fields {
		fields[f.Name] = f.GoType
	}
	if fields["Notes"] != "*string" {
		t.Fatalf("optional pointer failed: %q", fields["Notes"])
	}
	if fields["Lines"] != "[]Line" {
		t.Fatalf("many failed: %q", fields["Lines"])
	}
}

func TestUseCaseAndDrivingPort(t *testing.T) {
	raw := []map[string]any{
		{"@id": "p", "@type": "Package", "declaredName": "Ctx",
			"ownedElement": []any{map[string]any{"@id": "act"}}},
		{"@id": "act", "@type": "ActionDefinition", "declaredName": "PlaceOrder",
			"ownedElement": []any{map[string]any{"@id": "in"}}},
		{"@id": "in", "@type": "ItemUsage", "declaredName": "order", "direction": "in", "type": "Order"},
	}
	g, _ := model.Build(raw)
	proj, _ := New(baseCfg()).Build(g)
	ctx := proj.Contexts[0]
	if len(ctx.UseCases) != 1 || len(ctx.DrivingPorts) != 1 {
		t.Fatalf("use case/port not produced: %+v", ctx)
	}
	if len(ctx.UseCases[0].Input.Fields) != 1 {
		t.Fatalf("input DTO field missing")
	}
}
