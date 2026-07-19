package ast

import (
	"strings"

	"github.com/gaarutyunov/sysgo/engine/cst"
	"github.com/gaarutyunov/sysgo/engine/parser"
)

// Accept is an `accept` statement in an action body: a signal acceptance, or a
// durable timer (`accept after …` / `accept at …`).
type Accept struct{ node cst.Node }

func (a Accept) Syntax() cst.Node { return a.node }

// Mode returns "after", "at" (timers), or "signal" (the default).
func (a Accept) Mode() string {
	for _, c := range a.node.Children() {
		tok, ok := c.(cst.Token)
		if !ok || kindIsTrivia(tok.Kind()) {
			continue
		}
		if parser.SyntaxKind(tok.Kind()) == parser.KindIdent {
			switch tok.Text() {
			case "accept":
				continue // the leading keyword
			case "after", "at":
				return tok.Text()
			}
		}
		break
	}
	return "signal"
}

// Ref returns the accepted reference text: the signal/duration/time name, or a
// literal value for `accept after 5`.
func (a Accept) Ref() string {
	if qn, ok := firstChildOfKind(a.node, parser.KindQualifiedName); ok {
		return QualifiedName{qn}.String()
	}
	if e, ok := firstChildOfKind(a.node, parser.KindExpr); ok {
		return strings.TrimSpace(Expr{e}.Text())
	}
	return ""
}

// Target returns the accepted qualified name (a signal or duration reference),
// if one is present (absent for a literal timer value).
func (a Accept) Target() (QualifiedName, bool) {
	if qn, ok := firstChildOfKind(a.node, parser.KindQualifiedName); ok {
		return QualifiedName{qn}, true
	}
	return QualifiedName{}, false
}

// Accepts returns the declaration body's accept statements in order.
func (d Declaration) Accepts() []Accept {
	b, ok := d.Body()
	if !ok {
		return nil
	}
	var out []Accept
	for _, c := range childNodes(b.node) {
		if kindOf(c) == parser.KindAccept {
			out = append(out, Accept{c})
		}
	}
	return out
}
