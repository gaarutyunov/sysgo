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

// TestCanonicalGraphResolution exercises the real SysML JSON form: types are
// resolved via FeatureTyping references (by @id), multiplicity via a
// MultiplicityRange with a LiteralInfinity, and library elements are skipped.
func TestCanonicalGraphResolution(t *testing.T) {
	raw := []map[string]any{
		{"@id": "pkg", "@type": "Package", "declaredName": "Ctx",
			"ownedRelationship": []any{ref("m1"), ref("m2")}},
		{"@id": "m1", "@type": "OwningMembership", "ownedRelatedElement": []any{ref("order")}},
		{"@id": "m2", "@type": "LibraryPackage", "declaredName": "ScalarValues", "isLibraryElement": true,
			"ownedRelationship": []any{ref("ms")}},
		{"@id": "ms", "@type": "OwningMembership", "ownedRelatedElement": []any{ref("strType")}},
		{"@id": "strType", "@type": "DataType", "declaredName": "String", "isLibraryElement": true},

		{"@id": "order", "@type": "PartDefinition", "declaredName": "Order",
			"ownedRelationship": []any{ref("fm1"), ref("fm2")}},
		{"@id": "fm1", "@type": "FeatureMembership", "ownedRelatedElement": []any{ref("idf")}},
		{"@id": "idf", "@type": "AttributeUsage", "declaredName": "id",
			"ownedRelationship": []any{ref("ft1")}},
		{"@id": "ft1", "@type": "FeatureTyping", "type": ref("strType")},

		{"@id": "fm2", "@type": "FeatureMembership", "ownedRelatedElement": []any{ref("linesf")}},
		{"@id": "linesf", "@type": "PartUsage", "declaredName": "lines",
			"ownedRelationship": []any{ref("ft2"), ref("om2")}},
		{"@id": "ft2", "@type": "FeatureTyping", "type": ref("order")},
		{"@id": "om2", "@type": "OwningMembership", "ownedRelatedElement": []any{ref("mr")}},
		{"@id": "mr", "@type": "MultiplicityRange", "ownedRelationship": []any{ref("om3")}},
		{"@id": "om3", "@type": "OwningMembership", "ownedRelatedElement": []any{ref("inf")}},
		{"@id": "inf", "@type": "LiteralInfinity"},
	}
	g, _ := model.Build(raw)
	proj, _ := New(baseCfg()).Build(g)
	if len(proj.Contexts) != 1 {
		t.Fatalf("expected 1 context (library skipped), got %d", len(proj.Contexts))
	}
	ctx := proj.Contexts[0]
	if len(ctx.Entities) != 1 {
		t.Fatalf("expected Order entity, got %+v", ctx)
	}
	fields := map[string]string{}
	for _, f := range ctx.Entities[0].Fields {
		fields[f.Name] = f.GoType
	}
	if fields["ID"] != "string" {
		t.Fatalf("FeatureTyping scalar resolution failed: %q", fields["ID"])
	}
	if fields["Lines"] != "[]Order" {
		t.Fatalf("MultiplicityRange many resolution failed: %q", fields["Lines"])
	}
}

func ref(id string) map[string]any { return map[string]any{"@id": id} }

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
