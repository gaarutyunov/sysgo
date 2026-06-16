package model

import "testing"

func TestBuildOwnedElementFastPath(t *testing.T) {
	raw := []map[string]any{
		{"@id": "pkg", "@type": "Package", "declaredName": "P",
			"ownedElement": []any{map[string]any{"@id": "def"}}},
		{"@id": "def", "@type": "PartDefinition", "declaredName": "Order",
			"ownedElement": []any{map[string]any{"@id": "attr"}}},
		{"@id": "attr", "@type": "AttributeUsage", "declaredName": "id", "type": "String"},
	}
	g, err := Build(raw)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(g.Roots) != 1 || g.Roots[0].ID != "pkg" {
		t.Fatalf("expected single root pkg, got %+v", g.Roots)
	}
	def := g.Elements["def"]
	if len(def.Owned) != 1 || def.Owned[0].ID != "attr" {
		t.Fatalf("expected def to own attr, got %+v", def.Owned)
	}
	if def.Owned[0].Owner != def {
		t.Fatalf("owner backlink not set")
	}
}

func TestBuildMembershipHop(t *testing.T) {
	raw := []map[string]any{
		{"@id": "def", "@type": "PartDefinition", "declaredName": "Order",
			"ownedRelationship": []any{map[string]any{"@id": "mem"}}},
		{"@id": "mem", "@type": "OwningMembership",
			"ownedRelatedElement": []any{map[string]any{"@id": "child"}}},
		{"@id": "child", "@type": "AttributeUsage", "declaredName": "total"},
	}
	g, err := Build(raw)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	def := g.Elements["def"]
	if len(def.Owned) != 1 || def.Owned[0].ID != "child" {
		t.Fatalf("membership hop failed: %+v", def.Owned)
	}
	// Membership elements must not appear as roots.
	for _, r := range g.Roots {
		if r.ID == "mem" {
			t.Fatalf("membership leaked into roots")
		}
	}
}

func TestBuildDuplicateID(t *testing.T) {
	raw := []map[string]any{
		{"@id": "x", "@type": "Package"},
		{"@id": "x", "@type": "Package"},
	}
	if _, err := Build(raw); err == nil {
		t.Fatal("expected duplicate @id error")
	}
}

func TestQualifiedName(t *testing.T) {
	raw := []map[string]any{
		{"@id": "pkg", "@type": "Package", "declaredName": "Orders",
			"ownedElement": []any{map[string]any{"@id": "def"}}},
		{"@id": "def", "@type": "PartDefinition", "declaredName": "Order"},
	}
	g, _ := Build(raw)
	if got := g.Elements["def"].QualifiedName(); got != "Orders::Order" {
		t.Fatalf("QualifiedName = %q", got)
	}
}
