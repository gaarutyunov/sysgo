package temporal

import (
	"strings"

	"github.com/gaarutyunov/sysgo/engine"
)

// goScalar maps a library scalar type to its Go type. Duration is carried as a
// string here; a later slice can map it to time.Duration.
func goScalar(qn string) (string, bool) {
	switch qn {
	case "ScalarValues::String", "TemporalProfile::Duration":
		return "string", true
	case "ScalarValues::Boolean":
		return "bool", true
	case "ScalarValues::Real":
		return "float64", true
	case "ScalarValues::Integer", "ScalarValues::Natural":
		return "int64", true
	default:
		return "", false
	}
}

// typedTarget returns an element's feature-typing target (the type of an
// attribute or parameter usage).
func typedTarget(e engine.Element) (engine.Element, bool) {
	for _, rel := range e.Relationships() {
		if rel.Kind() == engine.FeatureTyping {
			return rel.Target()
		}
	}
	return engine.Element{}, false
}

// goType returns the Go type name for a typed usage and, when the type is a
// generated struct (an item/attribute definition), that definition. Scalars map
// to Go primitives; an unknown/absent type falls back to string.
func goType(e engine.Element) (typeName string, structDef engine.Element, isStruct bool) {
	t, ok := typedTarget(e)
	if !ok {
		return "string", engine.Element{}, false
	}
	if s, ok := goScalar(t.QualifiedName()); ok {
		return s, engine.Element{}, false
	}
	if t.Kind() == engine.ElementDefinition {
		return exported(t.Name()), t, true
	}
	return "string", engine.Element{}, false
}

// attributes returns a definition's attribute usages (its typed usage children).
func attributes(def engine.Element) []engine.Element {
	var out []engine.Element
	for _, c := range def.Children() {
		if c.Kind() == engine.ElementUsage {
			out = append(out, c)
		}
	}
	return out
}

// exported upper-cases the first letter so a SysML name becomes an exported Go
// identifier.
func exported(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
