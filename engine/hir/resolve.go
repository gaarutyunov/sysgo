package hir

import (
	"github.com/gaarutyunov/sysgo/engine/ast"
	"github.com/gaarutyunov/sysgo/engine/parser"
	"github.com/gaarutyunov/sysgo/engine/text"
)

// Model is a resolved-symbol view of one source file.
type Model struct {
	Root *Symbol
}

// Diagnostic is a resolution problem tied to a source range.
type Diagnostic struct {
	Message string
	Range   text.TextRange
}

// ResolvedRef records the outcome of resolving one name reference (currently
// import targets).
type ResolvedRef struct {
	Path     string // the referenced name, e.g. "A::B" or "A::*"
	Resolved bool
	Target   string // qualified name of the resolved symbol, or ""
	Range    text.TextRange
}

// Result is the full output of [Analyze].
type Result struct {
	Model       *Model
	Diagnostics []Diagnostic
	Names       []ResolvedRef
}

// Analyze parses src, builds its symbol model, resolves imports, and returns the
// model together with the resolved references and diagnostics.
func Analyze(src string) *Result {
	sf := ast.New(parser.Parse(src))
	root := &Symbol{Kind: KindRoot, members: map[string]*Symbol{}}
	root.root = root
	for _, m := range sf.Members() {
		buildMember(root, m)
	}
	r := &Result{Model: &Model{Root: root}}
	resolveImports(root, root, r)
	return r
}

func buildMember(parent *Symbol, m ast.Member) {
	switch x := m.(type) {
	case ast.Package:
		s := newSymbol(KindPackage, declName(x.Name()), parent, rangeOf(x))
		for _, cm := range x.Members() {
			buildMember(s, cm)
		}
	case ast.Import:
		if spec, ok := importSpecOf(x); ok {
			parent.imports = append(parent.imports, spec)
		}
	case ast.Declaration:
		s := newSymbol(declKind(x), declName(x.Name()), parent, rangeOf(x))
		for _, cm := range x.Members() {
			buildMember(s, cm)
		}
	}
}

func declKind(d ast.Declaration) SymbolKind {
	switch {
	case d.IsDefinition():
		return KindDefinition
	case d.IsUsage():
		return KindUsage
	default:
		return KindTypeDecl
	}
}

// declName returns the local name a declaration introduces: the last segment of
// its (possibly qualified) name, or "" if anonymous.
func declName(qn ast.QualifiedName, ok bool) string {
	if !ok {
		return ""
	}
	names := qn.Names()
	if len(names) == 0 {
		return ""
	}
	return names[len(names)-1].Text()
}

func rangeOf(n ast.Node) text.TextRange { return n.Syntax().Range() }

func importSpecOf(imp ast.Import) (importSpec, bool) {
	in, ok := imp.ImportName()
	if !ok {
		return importSpec{}, false
	}
	var segs []string
	for _, seg := range in.Segments() {
		segs = append(segs, seg.Text())
	}
	if len(segs) == 0 {
		return importSpec{}, false
	}
	return importSpec{
		segments:  segs,
		wildcard:  in.IsWildcard(),
		recursive: in.IsRecursive(),
		rng:       imp.Syntax().Range(),
	}, true
}

// resolveImports walks the symbol tree and resolves each scope's imports,
// appending references and diagnostics to r.
func resolveImports(root, s *Symbol, r *Result) {
	for _, imp := range s.imports {
		base := resolveAbsolute(root, imp.segments)
		ref := ResolvedRef{Path: imp.text(), Range: imp.rng, Resolved: base != nil}
		if base != nil {
			ref.Target = base.QualifiedName()
		} else {
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Message: "unresolved import '" + imp.text() + "'",
				Range:   imp.rng,
			})
		}
		r.Names = append(r.Names, ref)
	}
	for _, child := range s.order {
		resolveImports(root, child, r)
	}
}

// resolveAbsolute resolves segs from the root namespace, segment by segment.
func resolveAbsolute(root *Symbol, segs []string) *Symbol {
	cur := root
	for _, seg := range segs {
		next, ok := cur.members[seg]
		if !ok {
			return nil
		}
		cur = next
	}
	return cur
}

// Resolve resolves segs starting from scope: the first segment by walking
// outward through enclosing scopes and their imports, then each remaining
// segment as a direct member. It returns nil if unresolved.
func (m *Model) Resolve(scope *Symbol, segs []string) *Symbol {
	if len(segs) == 0 {
		return nil
	}
	cur := m.resolveSimple(scope, segs[0])
	for _, seg := range segs[1:] {
		if cur == nil {
			return nil
		}
		next, ok := cur.members[seg]
		if !ok {
			return nil
		}
		cur = next
	}
	return cur
}

func (m *Model) resolveSimple(scope *Symbol, name string) *Symbol {
	for s := scope; s != nil; s = s.Parent {
		if sym, ok := s.members[name]; ok {
			return sym
		}
		if sym := m.resolveViaImports(s, name); sym != nil {
			return sym
		}
	}
	return nil
}

func (m *Model) resolveViaImports(s *Symbol, name string) *Symbol {
	for _, imp := range s.imports {
		base := resolveAbsolute(m.Root, imp.segments)
		if base == nil {
			continue
		}
		switch {
		case imp.recursive:
			if sym := findRecursive(base, name); sym != nil {
				return sym
			}
		case imp.wildcard:
			if sym, ok := base.members[name]; ok {
				return sym
			}
		default:
			// `import A::B` brings the single element B into scope.
			if base.Name == name {
				return base
			}
		}
	}
	return nil
}

// findRecursive searches base and all its descendants for a symbol named name,
// breadth-first from base's members.
func findRecursive(base *Symbol, name string) *Symbol {
	for _, child := range base.order {
		if child.Name == name {
			return child
		}
	}
	for _, child := range base.order {
		if sym := findRecursive(child, name); sym != nil {
			return sym
		}
	}
	return nil
}
