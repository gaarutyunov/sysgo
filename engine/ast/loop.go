package ast

import (
	"strings"

	"github.com/gaarutyunov/sysgo/engine/cst"
	"github.com/gaarutyunov/sysgo/engine/parser"
)

// Loop is a `loop <count> times <activity>` repetition inside an action body.
type Loop struct{ node cst.Node }

func (l Loop) Syntax() cst.Node { return l.node }

// Count returns the loop's repeat count as written — an integer literal or a
// parameter reference.
func (l Loop) Count() string {
	if e, ok := firstChildOfKind(l.node, parser.KindExpr); ok {
		return strings.TrimSpace(Expr{e}.Text())
	}
	// No literal: the first qualified name is the count reference.
	if qns := l.qualifiedNames(); len(qns) >= 2 {
		return qns[0].String()
	}
	return ""
}

// Target returns the repeated activity reference (the qualified name after
// `times`).
func (l Loop) Target() (QualifiedName, bool) {
	qns := l.qualifiedNames()
	if len(qns) == 0 {
		return QualifiedName{}, false
	}
	// With a literal count, the sole qualified name is the target; with a
	// reference count, the target is the last of the two.
	return qns[len(qns)-1], true
}

func (l Loop) qualifiedNames() []QualifiedName {
	var out []QualifiedName
	for _, c := range childNodes(l.node) {
		if kindOf(c) == parser.KindQualifiedName {
			out = append(out, QualifiedName{c})
		}
	}
	return out
}

// Loops returns the declaration body's loops in order.
func (d Declaration) Loops() []Loop {
	b, ok := d.Body()
	if !ok {
		return nil
	}
	var out []Loop
	for _, c := range childNodes(b.node) {
		if kindOf(c) == parser.KindLoop {
			out = append(out, Loop{c})
		}
	}
	return out
}
