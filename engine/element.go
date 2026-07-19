package engine

import (
	"github.com/gaarutyunov/sysgo/engine/hir"
	"github.com/gaarutyunov/sysgo/engine/text"
)

// ElementKind classifies a model element.
type ElementKind uint8

const (
	ElementRoot ElementKind = iota
	ElementPackage
	ElementDefinition
	ElementUsage
	ElementType
)

func (k ElementKind) String() string {
	switch k {
	case ElementRoot:
		return "Root"
	case ElementPackage:
		return "Package"
	case ElementDefinition:
		return "Definition"
	case ElementUsage:
		return "Usage"
	default:
		return "Type"
	}
}

func fromSymbolKind(k hir.SymbolKind) ElementKind {
	switch k {
	case hir.KindRoot:
		return ElementRoot
	case hir.KindPackage:
		return ElementPackage
	case hir.KindDefinition:
		return ElementDefinition
	case hir.KindUsage:
		return ElementUsage
	default:
		return ElementType
	}
}

// Element is a typed, read-only view of one resolved model element. The zero
// value is invalid; check [Element.IsValid].
type Element struct {
	sym   *hir.Symbol
	model *Model
}

// IsValid reports whether the element refers to a real symbol.
func (e Element) IsValid() bool { return e.sym != nil }

// Name returns the element's local (unqualified) name.
func (e Element) Name() string { return e.sym.Name }

// QualifiedName returns the dotted-colon path from the root ("A::B::C").
func (e Element) QualifiedName() string { return e.sym.QualifiedName() }

// Kind returns the element's kind.
func (e Element) Kind() ElementKind { return fromSymbolKind(e.sym.Kind) }

// Range returns the element's source range.
func (e Element) Range() text.TextRange { return e.sym.Range }

// Children returns the element's directly declared members, in order.
func (e Element) Children() []Element {
	kids := e.sym.Children()
	out := make([]Element, len(kids))
	for i, k := range kids {
		out[i] = Element{sym: k, model: e.model}
	}
	return out
}

// Member returns the directly declared child named name.
func (e Element) Member(name string) (Element, bool) {
	if s, ok := e.sym.Member(name); ok {
		return Element{sym: s, model: e.model}, true
	}
	return Element{}, false
}

// Supertypes returns the resolved specialization targets this element inherits
// from.
func (e Element) Supertypes() []Element {
	out := make([]Element, len(e.sym.Supertypes))
	for i, st := range e.sym.Supertypes {
		out[i] = Element{sym: st, model: e.model}
	}
	return out
}

// InheritedMember looks up name among the members inherited through the
// specialization chain (cycle-safe).
func (e Element) InheritedMember(name string) (Element, bool) {
	if s := e.model.res.Model.InheritedMember(e.sym, name); s != nil {
		return Element{sym: s, model: e.model}, true
	}
	return Element{}, false
}

// LookupMember finds name as a direct member or, failing that, an inherited one.
func (e Element) LookupMember(name string) (Element, bool) {
	if s := e.model.res.Model.LookupMember(e.sym, name); s != nil {
		return Element{sym: s, model: e.model}, true
	}
	return Element{}, false
}

// Metadata is a metadata annotation attached to an element, with its assignment
// values read as text (e.g. an @REST annotation's path/method/status).
type Metadata struct {
	Name   string
	Keys   []string
	Values map[string]string
}

// Value returns the text of the assignment named key.
func (m Metadata) Value(key string) (string, bool) {
	v, ok := m.Values[key]
	return v, ok
}

// Annotations returns the element's metadata annotations.
func (e Element) Annotations() []Metadata {
	out := make([]Metadata, len(e.sym.Annotations))
	for i, a := range e.sym.Annotations {
		out[i] = Metadata{Name: a.Name, Keys: a.Keys, Values: a.Values}
	}
	return out
}

// Metadata returns the first annotation with the given name, if any.
func (e Element) Metadata(name string) (Metadata, bool) {
	for _, a := range e.sym.Annotations {
		if a.Name == name {
			return Metadata{Name: a.Name, Keys: a.Keys, Values: a.Values}, true
		}
	}
	return Metadata{}, false
}

// Relationships returns the element's resolved relationship clauses.
func (e Element) Relationships() []Relationship {
	out := make([]Relationship, len(e.sym.Relations))
	for i, rel := range e.sym.Relations {
		out[i] = Relationship{rel: rel, model: e.model}
	}
	return out
}

// RelationshipKind classifies a KerML relationship.
type RelationshipKind uint8

const (
	Specializes RelationshipKind = iota
	Subsets
	Redefines
	FeatureTyping
	Conjugates
)

func (k RelationshipKind) String() string {
	switch k {
	case Specializes:
		return "specializes"
	case Subsets:
		return "subsets"
	case Redefines:
		return "redefines"
	case FeatureTyping:
		return "typed"
	case Conjugates:
		return "conjugates"
	default:
		return "relationship"
	}
}

func fromRelKind(k hir.RelKind) RelationshipKind {
	switch k {
	case hir.RelSpecializes:
		return Specializes
	case hir.RelSubsets:
		return Subsets
	case hir.RelRedefines:
		return Redefines
	case hir.RelTyped:
		return FeatureTyping
	default:
		return Conjugates
	}
}

// Relationship is one resolved relationship clause on an element.
type Relationship struct {
	rel   hir.Relation
	model *Model
}

// Kind returns the relationship kind.
func (r Relationship) Kind() RelationshipKind { return fromRelKind(r.rel.Kind) }

// TargetName returns the referenced target name as written in source.
func (r Relationship) TargetName() string { return r.rel.Name }

// IsResolved reports whether the target name resolved to an element.
func (r Relationship) IsResolved() bool { return r.rel.Target != nil }

// Target returns the resolved target element, if the relationship resolved.
func (r Relationship) Target() (Element, bool) {
	if r.rel.Target == nil {
		return Element{}, false
	}
	return Element{sym: r.rel.Target, model: r.model}, true
}
