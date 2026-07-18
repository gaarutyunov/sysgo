package contracts

import (
	"sort"

	"github.com/gaarutyunov/sysgo/engine"
)

// SchemaFor builds the JSON Schema for a resolved item/attribute definition. The
// schema is a self-contained object with a property per attribute, including
// attributes inherited through the specialization chain (flattened, no allOf).
func SchemaFor(def engine.Element) *Schema {
	return schemaForObject(def, map[string]bool{})
}

// schemaForObject builds an object schema, guarding against type cycles by the
// set of qualified names currently being expanded.
func schemaForObject(def engine.Element, active map[string]bool) *Schema {
	s := &Schema{Type: "object", Properties: map[string]*Schema{}}

	qn := def.QualifiedName()
	active[qn] = true
	defer delete(active, qn)

	// Collect attributes from the definition and its supertypes (flattened).
	// A subtype's own attribute wins over an inherited one of the same name.
	seen := map[string]bool{}
	for _, attr := range attributes(def) {
		addProperty(s, attr, seen, active)
	}
	for _, super := range allSupertypes(def) {
		for _, attr := range attributes(super) {
			addProperty(s, attr, seen, active)
		}
	}

	if len(s.Required) > 1 {
		sort.Strings(s.Required)
	}
	return s
}

func addProperty(s *Schema, attr engine.Element, seen map[string]bool, active map[string]bool) {
	name := attr.Name()
	if name == "" || seen[name] {
		return
	}
	seen[name] = true

	typ, ok := attributeType(attr)
	prop := &Schema{}
	switch {
	case !ok:
		// Untyped attribute → permissive empty schema.
	case isScalar(typ.QualifiedName()):
		prop = scalarSchema(typ.QualifiedName())
	case active[typ.QualifiedName()]:
		// A cycle back to a type already being expanded — stop with a bare
		// object rather than recursing forever.
		prop = &Schema{Type: "object"}
	default:
		prop = schemaForObject(typ, active)
	}

	s.Properties[name] = prop
	s.Required = append(s.Required, name) // all attributes required (this slice)
}

// attributes returns the direct attribute usages of a definition (its usage
// children).
func attributes(def engine.Element) []engine.Element {
	var out []engine.Element
	for _, c := range def.Children() {
		if c.Kind() == engine.ElementUsage {
			out = append(out, c)
		}
	}
	return out
}

// attributeType returns the resolved feature-typing target of an attribute.
func attributeType(attr engine.Element) (engine.Element, bool) {
	for _, rel := range attr.Relationships() {
		if rel.Kind() == engine.FeatureTyping {
			return rel.Target()
		}
	}
	return engine.Element{}, false
}

// allSupertypes returns every transitive supertype of def, nearest first,
// cycle-safe.
func allSupertypes(def engine.Element) []engine.Element {
	var out []engine.Element
	seen := map[string]bool{def.QualifiedName(): true}
	queue := def.Supertypes()
	for len(queue) > 0 {
		st := queue[0]
		queue = queue[1:]
		qn := st.QualifiedName()
		if seen[qn] {
			continue
		}
		seen[qn] = true
		out = append(out, st)
		queue = append(queue, st.Supertypes()...)
	}
	return out
}

func isScalar(qn string) bool {
	switch qn {
	case "ScalarValues::String", "ScalarValues::Boolean", "ScalarValues::Real",
		"ScalarValues::Integer", "ScalarValues::Natural":
		return true
	default:
		return false
	}
}

// scalarSchema maps a library scalar type to its JSON Schema 2020-12 form.
func scalarSchema(qn string) *Schema {
	switch qn {
	case "ScalarValues::String":
		return &Schema{Type: "string"}
	case "ScalarValues::Boolean":
		return &Schema{Type: "boolean"}
	case "ScalarValues::Real":
		return &Schema{Type: "number"}
	case "ScalarValues::Integer", "ScalarValues::Natural":
		return &Schema{Type: "integer"}
	default:
		return &Schema{}
	}
}
