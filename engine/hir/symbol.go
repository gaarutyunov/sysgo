package hir

import (
	"strings"

	"github.com/gaarutyunov/sysgo/engine/text"
)

// SymbolKind classifies a named element.
type SymbolKind uint8

const (
	KindRoot       SymbolKind = iota // the implicit global namespace
	KindPackage                      // package
	KindDefinition                   // SysML definition (part def, …)
	KindUsage                        // SysML usage (part p, …)
	KindTypeDecl                     // KerML type/classifier/feature declaration
)

func (k SymbolKind) String() string {
	switch k {
	case KindRoot:
		return "Root"
	case KindPackage:
		return "Package"
	case KindDefinition:
		return "Definition"
	case KindUsage:
		return "Usage"
	case KindTypeDecl:
		return "TypeDecl"
	default:
		return "Symbol"
	}
}

// Symbol is a named element in the model tree.
type Symbol struct {
	Name   string
	Kind   SymbolKind
	Parent *Symbol
	Range  text.TextRange

	// Supertypes holds the resolved specialization targets — the types this
	// symbol inherits members from. Populated by relationship resolution.
	Supertypes []*Symbol

	// Relations holds this symbol's resolved relationship clauses (with typed
	// target symbols). Populated by relationship resolution.
	Relations []Relation

	// Annotations holds this symbol's metadata annotations (@Name { … }) with
	// their body assignment values.
	Annotations []Annotation

	// Keywords holds the declaration's leading keyword tokens (modifiers, kind
	// keyword, and optional "def"), used to expose direction and kind.
	Keywords []string

	// Performs holds the resolved `perform` action references in this action's
	// body, in declaration order.
	Performs []PerformStep

	// Successions holds the resolved `first A then B` control edges in this
	// action's body, in declaration order.
	Successions []SuccessionEdge

	members     map[string]*Symbol // local name → child (first wins on collision)
	order       []*Symbol          // children in declaration order
	imports     []importSpec       // imports declared directly in this scope
	rels        []relSpec          // relationship clauses declared on this symbol
	performs    []performSpec      // perform statements declared in this action body
	successions []successionSpec   // succession edges declared in this action body
	root        *Symbol
}

// RelKind classifies a KerML relationship clause.
type RelKind uint8

const (
	RelSpecializes RelKind = iota // :> / specializes
	RelSubsets                    // subsets
	RelRedefines                  // :>> / redefines
	RelTyped                      // : (feature typing)
	RelConjugates                 // ~ / conjugates
)

func (k RelKind) String() string {
	switch k {
	case RelSpecializes:
		return "specializes"
	case RelSubsets:
		return "subsets"
	case RelRedefines:
		return "redefines"
	case RelTyped:
		return "typed"
	case RelConjugates:
		return "conjugates"
	default:
		return "relationship"
	}
}

// Relation is a resolved relationship clause on a symbol, with its typed target.
type Relation struct {
	Kind   RelKind
	Name   string  // the referenced target name
	Target *Symbol // resolved target, or nil if unresolved
	Range  text.TextRange
}

// Annotation is a metadata annotation (@Name { key = value; … }) attached to a
// symbol, with its body assignments read as text.
type Annotation struct {
	Name   string            // e.g. "REST"
	Keys   []string          // assignment names, in source order
	Values map[string]string // assignment name → expression text
}

// Value returns the text of the assignment named key.
func (a Annotation) Value(key string) (string, bool) {
	v, ok := a.Values[key]
	return v, ok
}

// PerformStep is a resolved `perform` action reference: its step name (if any)
// and the target action it performs.
type PerformStep struct {
	Name   string  // local step name, or "" for a direct reference
	Target *Symbol // resolved target action, or nil if unresolved
	Name0  string  // the referenced target name as written
	Range  text.TextRange
}

// SuccessionEdge is a resolved `first A then B` control edge.
type SuccessionEdge struct {
	Source     *Symbol
	Target     *Symbol
	SourceName string
	TargetName string
	Range      text.TextRange
}

// successionSpec is a succession edge captured at build time, before resolution.
type successionSpec struct {
	source []string
	target []string
	rng    text.TextRange
}

// performSpec is a perform statement captured at build time, before resolution.
type performSpec struct {
	name   string
	target []string
	rng    text.TextRange
}

// relSpec is a relationship clause captured at build time, before resolution.
type relSpec struct {
	kind   RelKind
	target []string // qualified target name segments
	rng    text.TextRange
}

// relKindOf maps a relationship operator or keyword to its RelKind.
func relKindOf(op string) (RelKind, bool) {
	switch op {
	case ":>", "specializes":
		return RelSpecializes, true
	case "subsets":
		return RelSubsets, true
	case ":>>", "redefines":
		return RelRedefines, true
	case ":":
		return RelTyped, true
	case "~", "conjugates":
		return RelConjugates, true
	default:
		return 0, false
	}
}

func newSymbol(kind SymbolKind, name string, parent *Symbol, rng text.TextRange) *Symbol {
	s := &Symbol{Name: name, Kind: kind, Parent: parent, Range: rng, members: map[string]*Symbol{}}
	if parent != nil {
		s.root = parent.root
		parent.order = append(parent.order, s)
		if name != "" {
			if _, exists := parent.members[name]; !exists {
				parent.members[name] = s
			}
		}
	}
	return s
}

// Children returns the symbol's direct children in declaration order.
func (s *Symbol) Children() []*Symbol { return s.order }

// Member returns the direct child named name, if any.
func (s *Symbol) Member(name string) (*Symbol, bool) {
	m, ok := s.members[name]
	return m, ok
}

// QualifiedName returns the dotted-colon path from the root to this symbol
// (excluding the anonymous root), e.g. "A::B::C".
func (s *Symbol) QualifiedName() string {
	var parts []string
	for cur := s; cur != nil && cur.Kind != KindRoot; cur = cur.Parent {
		parts = append(parts, cur.Name)
	}
	// reverse
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, "::")
}

// importSpec is one import declared in a scope.
type importSpec struct {
	segments  []string
	wildcard  bool
	recursive bool
	rng       text.TextRange
}

func (imp importSpec) text() string {
	s := strings.Join(imp.segments, "::")
	switch {
	case imp.recursive:
		return s + "::**"
	case imp.wildcard:
		return s + "::*"
	default:
		return s
	}
}
