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

// RelRef records the outcome of resolving one relationship clause. It is a
// value type (no *Symbol) so it can ride in a salsa query result.
type RelRef struct {
	Kind     RelKind
	From     string // qualified name of the declaring symbol
	Name     string // the referenced target name
	Target   string // qualified name of the resolved target, or ""
	Resolved bool
	Range    text.TextRange
}

// Result is the full output of [Analyze].
type Result struct {
	Model         *Model
	Diagnostics   []Diagnostic
	Names         []ResolvedRef
	Relationships []RelRef
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
	m := r.Model
	m.buildSupertypes(root)
	m.resolveRelationships(root, r)
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
		s.rels = relSpecsOf(x)
		for _, cm := range x.Members() {
			buildMember(s, cm)
		}
	}
}

// relSpecsOf captures a declaration's relationship clauses before resolution.
func relSpecsOf(d ast.Declaration) []relSpec {
	var out []relSpec
	for _, rel := range d.Relationships() {
		kind, ok := relKindOf(rel.Operator())
		if !ok {
			continue
		}
		for _, tgt := range rel.Targets() {
			var segs []string
			for _, seg := range tgt.Names() {
				segs = append(segs, seg.Text())
			}
			if len(segs) == 0 {
				continue
			}
			out = append(out, relSpec{kind: kind, target: segs, rng: rel.Syntax().Range()})
		}
	}
	return out
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

// scopeOf returns the scope a symbol's relationship targets resolve in: its
// enclosing scope.
func scopeOf(s *Symbol) *Symbol {
	if s.Parent != nil {
		return s.Parent
	}
	return s.root
}

// buildSupertypes resolves each symbol's specialization targets and records
// them as Supertypes, so inherited-member lookup can walk the chain.
func (m *Model) buildSupertypes(s *Symbol) {
	for _, rel := range s.rels {
		if rel.kind != RelSpecializes {
			continue
		}
		if target := m.Resolve(scopeOf(s), rel.target); target != nil {
			s.Supertypes = append(s.Supertypes, target)
		}
	}
	for _, child := range s.order {
		m.buildSupertypes(child)
	}
}

// resolveRelationships resolves every relationship clause to a target symbol,
// appending references and diagnostics to r. Redefinition and subsetting of a
// single-segment name fall back to inherited-member lookup on the enclosing
// type when the name is not lexically in scope.
func (m *Model) resolveRelationships(s *Symbol, r *Result) {
	for _, rel := range s.rels {
		target := m.Resolve(scopeOf(s), rel.target)
		if target == nil && len(rel.target) == 1 && (rel.kind == RelRedefines || rel.kind == RelSubsets) {
			target = m.InheritedMember(s.Parent, rel.target[0])
		}
		name := joinSegs(rel.target)
		ref := RelRef{Kind: rel.kind, From: s.QualifiedName(), Name: name, Range: rel.rng, Resolved: target != nil}
		if target != nil {
			ref.Target = target.QualifiedName()
		} else {
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Message: "unresolved " + rel.kind.String() + " target '" + name + "'",
				Range:   rel.rng,
			})
		}
		r.Relationships = append(r.Relationships, ref)
	}
	for _, child := range s.order {
		m.resolveRelationships(child, r)
	}
}

// InheritedMember looks up name among the members of owner's supertypes,
// walking the specialization chain transitively. It is cycle-safe. It returns
// nil if owner is nil or the name is not inherited.
func (m *Model) InheritedMember(owner *Symbol, name string) *Symbol {
	if owner == nil {
		return nil
	}
	seen := make(map[*Symbol]bool)
	var walk func(t *Symbol) *Symbol
	walk = func(t *Symbol) *Symbol {
		if t == nil || seen[t] {
			return nil
		}
		seen[t] = true
		for _, st := range t.Supertypes {
			if mem, ok := st.members[name]; ok {
				return mem
			}
			if mem := walk(st); mem != nil {
				return mem
			}
		}
		return nil
	}
	return walk(owner)
}

// LookupMember finds name as a direct member of owner or, failing that, as a
// member inherited through the specialization chain.
func (m *Model) LookupMember(owner *Symbol, name string) *Symbol {
	if owner == nil {
		return nil
	}
	if mem, ok := owner.members[name]; ok {
		return mem
	}
	return m.InheritedMember(owner, name)
}

func joinSegs(segs []string) string {
	out := ""
	for i, s := range segs {
		if i > 0 {
			out += "::"
		}
		out += s
	}
	return out
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
