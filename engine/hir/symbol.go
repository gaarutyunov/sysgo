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

	members map[string]*Symbol // local name → child (first wins on collision)
	order   []*Symbol          // children in declaration order
	imports []importSpec       // imports declared directly in this scope
	root    *Symbol
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
