package model

import (
	"fmt"
	"sort"
	"strings"
)

// membershipTypes are relationship element types that carry containment but are
// not themselves domain elements; they are excluded from Roots.
func isRelationshipType(t string) bool {
	return strings.Contains(t, "Membership") || strings.Contains(t, "Relationship") ||
		strings.HasSuffix(t, "Typing") || strings.HasSuffix(t, "Subsetting") ||
		strings.HasSuffix(t, "Redefinition") || strings.HasSuffix(t, "Specialization")
}

// Build resolves a flat list of decoded SysML element objects into a Graph.
// Each entry of raw is the JSON object for one element. Membership containment
// is resolved into Element.Owned/Owner, preferring the derived "ownedElement"
// fast path when present.
func Build(raw []map[string]any) (*Graph, error) {
	g := &Graph{Elements: make(map[string]*Element, len(raw))}

	// First pass: create elements keyed by id.
	for _, r := range raw {
		id := elementID(r)
		if id == "" {
			continue
		}
		e := &Element{
			ID:           id,
			Type:         stringField(r, "@type"),
			DeclaredName: declaredName(r),
			Raw:          r,
		}
		if _, dup := g.Elements[id]; dup {
			return nil, fmt.Errorf("duplicate element @id %q", id)
		}
		g.Elements[id] = e
	}

	contained := make(map[string]bool)

	// Second pass: resolve containment.
	for _, e := range g.Elements {
		children := g.resolveChildren(e)
		for _, c := range children {
			c.Owner = e
			contained[c.ID] = true
		}
		e.Owned = children
	}

	// Roots: non-contained, non-relationship elements.
	for _, e := range g.Elements {
		if contained[e.ID] || isRelationshipType(e.Type) {
			continue
		}
		g.Roots = append(g.Roots, e)
	}
	sort.Slice(g.Roots, func(i, j int) bool {
		if g.Roots[i].DeclaredName != g.Roots[j].DeclaredName {
			return g.Roots[i].DeclaredName < g.Roots[j].DeclaredName
		}
		return g.Roots[i].ID < g.Roots[j].ID
	})
	return g, nil
}

// resolveChildren returns the resolved child elements of e, deterministically
// ordered by their position in the source relationship arrays.
func (g *Graph) resolveChildren(e *Element) []*Element {
	var out []*Element
	seen := make(map[string]bool)

	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		if c, ok := g.Elements[id]; ok {
			seen[id] = true
			out = append(out, c)
		}
	}

	// Fast path: derived ownedElement array of references.
	if refs := refList(e.Raw["ownedElement"]); len(refs) > 0 {
		for _, id := range refs {
			add(id)
		}
		return out
	}

	// Membership hop: ownedRelationship -> membership -> ownedRelatedElement.
	for _, relID := range refList(e.Raw["ownedRelationship"]) {
		m, ok := g.Elements[relID]
		if !ok {
			continue
		}
		// A membership's contained element(s).
		for _, childID := range refList(m.Raw["ownedRelatedElement"]) {
			add(childID)
		}
		// Some serializations inline the related element id directly.
		if id := refID(m.Raw["memberElement"]); id != "" {
			add(id)
		}
	}
	return out
}

// ---- low-level field helpers ----

func elementID(r map[string]any) string {
	if id := stringField(r, "@id"); id != "" {
		return id
	}
	return stringField(r, "elementId")
}

func declaredName(r map[string]any) string {
	if n := stringField(r, "declaredName"); n != "" {
		return n
	}
	return stringField(r, "name")
}

func stringField(r map[string]any, key string) string {
	if v, ok := r[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// refID extracts the "@id" from a reference object {"@id": "..."} or a bare
// string id.
func refID(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		return stringField(t, "@id")
	}
	return ""
}

// refList normalizes a reference field that may be a single ref or an array of
// refs into a slice of ids.
func refList(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if id := refID(item); id != "" {
				out = append(out, id)
			}
		}
		return out
	default:
		if id := refID(v); id != "" {
			return []string{id}
		}
	}
	return nil
}
