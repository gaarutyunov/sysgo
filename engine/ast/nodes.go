package ast

import (
	"strings"

	"github.com/gaarutyunov/sysgo/engine/cst"
	"github.com/gaarutyunov/sysgo/engine/parser"
)

// New returns the typed [SourceFile] view of a parsed tree.
func New(tree *cst.Tree) SourceFile { return SourceFile{tree.Root()} }

// Member is a top-level or body member: a [Package], [Import], or [Declaration].
type Member interface {
	Node
	memberMarker()
}

func wrapMember(n cst.Node) (Member, bool) {
	switch kindOf(n) {
	case parser.KindPackage:
		return Package{n}, true
	case parser.KindImport:
		return Import{n}, true
	case parser.KindDef, parser.KindUsage, parser.KindTypeDecl:
		return Declaration{n}, true
	default:
		return nil, false
	}
}

func members(n cst.Node) []Member {
	var out []Member
	for _, c := range childNodes(n) {
		if m, ok := wrapMember(c); ok {
			out = append(out, m)
		}
	}
	return out
}

// SourceFile is a whole parsed file: a namespace of members.
type SourceFile struct{ node cst.Node }

func (n SourceFile) Syntax() cst.Node { return n.node }

// Members returns the file's top-level members.
func (n SourceFile) Members() []Member { return members(n.node) }

// Body is a { ... } member body.
type Body struct{ node cst.Node }

func (b Body) Syntax() cst.Node  { return b.node }
func (b Body) Members() []Member { return members(b.node) }

// Package is a package declaration.
type Package struct{ node cst.Node }

func (p Package) Syntax() cst.Node { return p.node }
func (p Package) memberMarker()    {}

// Name returns the package's qualified name, if present.
func (p Package) Name() (QualifiedName, bool) { return qualifiedNameChild(p.node) }

// Body returns the package body, if the package has one (vs. a `;`).
func (p Package) Body() (Body, bool) { return bodyChild(p.node) }

// Members returns the members declared in the package body.
func (p Package) Members() []Member {
	if b, ok := p.Body(); ok {
		return b.Members()
	}
	return nil
}

// Visibility returns the package's visibility prefix, if any.
func (p Package) Visibility() (Visibility, bool) { return visibilityChild(p.node) }

// Import is an import declaration.
type Import struct{ node cst.Node }

func (i Import) Syntax() cst.Node { return i.node }
func (i Import) memberMarker()    {}

// Visibility returns the import's visibility prefix, if any.
func (i Import) Visibility() (Visibility, bool) { return visibilityChild(i.node) }

// ImportName returns the imported name (possibly a wildcard), if present.
func (i Import) ImportName() (ImportName, bool) {
	if c, ok := firstChildOfKind(i.node, parser.KindImportName); ok {
		return ImportName{c}, true
	}
	return ImportName{}, false
}

// ImportName is a qualified name with an optional ::* / ::** wildcard tail.
type ImportName struct{ node cst.Node }

func (i ImportName) Syntax() cst.Node { return i.node }

// Segments returns the name segments of the import path.
func (i ImportName) Segments() []Name { return nameChildren(i.node) }

// IsWildcard reports whether the import ends in ::* or ::**.
func (i ImportName) IsWildcard() bool {
	for _, c := range i.node.Children() {
		if tok, ok := c.(cst.Token); ok && kindOf2(tok) == parser.KindStar {
			return true
		}
	}
	return false
}

// IsRecursive reports whether the import ends in the recursive ::** wildcard.
func (i ImportName) IsRecursive() bool {
	stars := 0
	for _, c := range i.node.Children() {
		if tok, ok := c.(cst.Token); ok && kindOf2(tok) == parser.KindStar {
			stars++
		}
	}
	return stars >= 2
}

// Declaration is a KerML or SysML declaration: a definition (`def`), a usage,
// or a bare KerML type declaration. The three share one structure and are
// distinguished by [Declaration.IsDefinition] / [Declaration.IsUsage] /
// [Declaration.IsTypeDecl].
type Declaration struct{ node cst.Node }

func (d Declaration) Syntax() cst.Node { return d.node }
func (d Declaration) memberMarker()    {}

// IsDefinition reports whether the declaration is a SysML definition (`def`).
func (d Declaration) IsDefinition() bool { return kindOf(d.node) == parser.KindDef }

// IsUsage reports whether the declaration is a SysML usage.
func (d Declaration) IsUsage() bool { return kindOf(d.node) == parser.KindUsage }

// IsTypeDecl reports whether the declaration is a bare KerML type declaration.
func (d Declaration) IsTypeDecl() bool { return kindOf(d.node) == parser.KindTypeDecl }

// Name returns the declared name (not relationship targets), if present.
func (d Declaration) Name() (QualifiedName, bool) { return qualifiedNameChild(d.node) }

// Visibility returns the declaration's visibility prefix, if any.
func (d Declaration) Visibility() (Visibility, bool) { return visibilityChild(d.node) }

// Annotations returns the declaration's @/# prefix annotations.
func (d Declaration) Annotations() []Annotation {
	var out []Annotation
	for _, c := range childNodes(d.node) {
		if kindOf(c) == parser.KindAnnotation {
			out = append(out, Annotation{c})
		}
	}
	return out
}

// Relationships returns the declaration's relationship clauses.
func (d Declaration) Relationships() []Relationship {
	var out []Relationship
	for _, c := range childNodes(d.node) {
		if kindOf(c) == parser.KindRelationship {
			out = append(out, Relationship{c})
		}
	}
	return out
}

// Multiplicity returns the declaration's multiplicity clause, if any.
func (d Declaration) Multiplicity() (Multiplicity, bool) {
	if c, ok := firstChildOfKind(d.node, parser.KindMultiplicity); ok {
		return Multiplicity{c}, true
	}
	return Multiplicity{}, false
}

// FeatureValue returns the declaration's `= expr` / `:= expr` value, if any.
func (d Declaration) FeatureValue() (FeatureValue, bool) {
	if c, ok := firstChildOfKind(d.node, parser.KindFeatureValue); ok {
		return FeatureValue{c}, true
	}
	return FeatureValue{}, false
}

// Body returns the declaration body, if it has one.
func (d Declaration) Body() (Body, bool) { return bodyChild(d.node) }

// Members returns the members declared in the declaration body.
func (d Declaration) Members() []Member {
	if b, ok := d.Body(); ok {
		return b.Members()
	}
	return nil
}

// QualifiedName is a Name (:: Name)* dotted-colon name.
type QualifiedName struct{ node cst.Node }

func (q QualifiedName) Syntax() cst.Node { return q.node }

// Names returns the individual name segments.
func (q QualifiedName) Names() []Name { return nameChildren(q.node) }

// String renders the qualified name as A::B::C.
func (q QualifiedName) String() string {
	parts := make([]string, 0)
	for _, n := range q.Names() {
		parts = append(parts, n.Text())
	}
	return strings.Join(parts, "::")
}

// Name is a single name segment (an identifier or restricted 'name').
type Name struct{ node cst.Node }

func (n Name) Syntax() cst.Node { return n.node }

// Text returns the segment's identifier text (quotes included for a restricted
// name), with surrounding trivia stripped.
func (n Name) Text() string {
	if t, ok := firstToken(n.node); ok {
		return t
	}
	return ""
}

// Relationship is one relationship clause (specialization, subsetting, etc.).
type Relationship struct{ node cst.Node }

func (r Relationship) Syntax() cst.Node { return r.node }

// Operator returns the relationship operator or keyword (":>", "subsets", …).
func (r Relationship) Operator() string {
	if t, ok := firstToken(r.node); ok {
		return t
	}
	return ""
}

// Targets returns the relationship's target names.
func (r Relationship) Targets() []QualifiedName {
	var out []QualifiedName
	for _, c := range childNodes(r.node) {
		if kindOf(c) == parser.KindQualifiedName {
			out = append(out, QualifiedName{c})
		}
	}
	return out
}

// Multiplicity is a [ ... ] multiplicity clause.
type Multiplicity struct{ node cst.Node }

func (m Multiplicity) Syntax() cst.Node { return m.node }

// Text returns the clause text, e.g. "[*]" or "[0..1]".
func (m Multiplicity) Text() string { return strings.TrimSpace(m.node.Text()) }

// FeatureValue is a `= expr` or `:= expr` feature value.
type FeatureValue struct{ node cst.Node }

func (f FeatureValue) Syntax() cst.Node { return f.node }

// IsDefault reports whether the value uses `:=` (a default) rather than `=`.
func (f FeatureValue) IsDefault() bool {
	for _, c := range f.node.Children() {
		if tok, ok := c.(cst.Token); ok && kindOf2(tok) == parser.KindColonEq {
			return true
		}
	}
	return false
}

// Expr returns the value expression.
func (f FeatureValue) Expr() (Expr, bool) {
	if c, ok := firstChildOfKind(f.node, parser.KindExpr); ok {
		return Expr{c}, true
	}
	return Expr{}, false
}

// Expr is a primary expression: a name reference or a literal.
type Expr struct{ node cst.Node }

func (e Expr) Syntax() cst.Node { return e.node }

// Text returns the expression text with surrounding trivia stripped.
func (e Expr) Text() string { return strings.TrimSpace(e.node.Text()) }

// Visibility is a public/private/protected prefix.
type Visibility struct{ node cst.Node }

func (v Visibility) Syntax() cst.Node { return v.node }

// Text returns the visibility keyword.
func (v Visibility) Text() string {
	if t, ok := firstToken(v.node); ok {
		return t
	}
	return ""
}

// Annotation is an @Name / #Name prefix annotation.
type Annotation struct{ node cst.Node }

func (a Annotation) Syntax() cst.Node { return a.node }

// Name returns the annotated name, if present.
func (a Annotation) Name() (QualifiedName, bool) { return qualifiedNameChild(a.node) }

// --- shared child helpers ---

func qualifiedNameChild(n cst.Node) (QualifiedName, bool) {
	if c, ok := firstChildOfKind(n, parser.KindQualifiedName); ok {
		return QualifiedName{c}, true
	}
	return QualifiedName{}, false
}

func bodyChild(n cst.Node) (Body, bool) {
	if c, ok := firstChildOfKind(n, parser.KindBody); ok {
		return Body{c}, true
	}
	return Body{}, false
}

func visibilityChild(n cst.Node) (Visibility, bool) {
	if c, ok := firstChildOfKind(n, parser.KindVisibility); ok {
		return Visibility{c}, true
	}
	return Visibility{}, false
}

func nameChildren(n cst.Node) []Name {
	var out []Name
	for _, c := range childNodes(n) {
		if kindOf(c) == parser.KindName {
			out = append(out, Name{c})
		}
	}
	return out
}

func kindOf2(t cst.Token) parser.SyntaxKind { return parser.SyntaxKind(t.Kind()) }
