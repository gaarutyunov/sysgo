package ast

import (
	"strings"

	"github.com/gaarutyunov/sysgo/engine/cst"
	"github.com/gaarutyunov/sysgo/engine/parser"
)

// Succession is a `first A then B` control-flow edge inside an action body.
type Succession struct{ node cst.Node }

func (s Succession) Syntax() cst.Node { return s.node }

// qualifiedNames returns the succession's direct QualifiedName children in order.
func (s Succession) qualifiedNames() []QualifiedName {
	var out []QualifiedName
	for _, c := range childNodes(s.node) {
		if kindOf(c) == parser.KindQualifiedName {
			out = append(out, QualifiedName{c})
		}
	}
	return out
}

func (s Succession) hasFirst() bool {
	for _, c := range s.node.Children() {
		if tok, ok := c.(cst.Token); ok && !kindIsTrivia(tok.Kind()) &&
			parser.SyntaxKind(tok.Kind()) == parser.KindIdent && tok.Text() == "first" {
			return true
		}
	}
	return false
}

// Source returns the succession's source reference (`first A`), if present. A
// bare `then B` has no explicit source (the previous element is implied).
func (s Succession) Source() (QualifiedName, bool) {
	qns := s.qualifiedNames()
	if s.hasFirst() && len(qns) >= 2 {
		return qns[0], true
	}
	return QualifiedName{}, false
}

// Target returns the succession's target reference (`then B`).
func (s Succession) Target() (QualifiedName, bool) {
	qns := s.qualifiedNames()
	if len(qns) == 0 {
		return QualifiedName{}, false
	}
	return qns[len(qns)-1], true
}

// Guard returns the succession's `if` condition text, if present.
func (s Succession) Guard() (string, bool) {
	if e, ok := firstChildOfKind(s.node, parser.KindExpr); ok {
		return strings.TrimSpace(Expr{e}.Text()), true
	}
	return "", false
}

// Successions returns the declaration body's succession edges in order.
func (d Declaration) Successions() []Succession {
	b, ok := d.Body()
	if !ok {
		return nil
	}
	var out []Succession
	for _, c := range childNodes(b.node) {
		if kindOf(c) == parser.KindSuccession {
			out = append(out, Succession{c})
		}
	}
	return out
}
