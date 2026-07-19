package ast

import (
	"github.com/gaarutyunov/sysgo/engine/cst"
	"github.com/gaarutyunov/sysgo/engine/parser"
)

// Perform is a `perform` action reference inside an action body.
type Perform struct{ node cst.Node }

func (p Perform) Syntax() cst.Node { return p.node }

// Target returns the qualified name of the performed action: the typing target
// when written as `perform x : Target`, otherwise the direct reference
// `perform Target`.
func (p Perform) Target() (QualifiedName, bool) {
	if rel, ok := firstChildOfKind(p.node, parser.KindRelationship); ok {
		if qn, ok := firstChildOfKind(rel, parser.KindQualifiedName); ok {
			return QualifiedName{qn}, true
		}
	}
	if qn, ok := firstChildOfKind(p.node, parser.KindQualifiedName); ok {
		return QualifiedName{qn}, true
	}
	return QualifiedName{}, false
}

// Name returns the local step name (`perform name : Target`), or "" for a direct
// `perform Target` reference.
func (p Perform) Name() string {
	if _, ok := firstChildOfKind(p.node, parser.KindRelationship); !ok {
		return "" // direct reference: the qualified name is the target, not a name
	}
	if qn, ok := firstChildOfKind(p.node, parser.KindQualifiedName); ok {
		return QualifiedName{qn}.String()
	}
	return ""
}

// Performs returns the declaration body's `perform` steps in declaration order.
func (d Declaration) Performs() []Perform {
	b, ok := d.Body()
	if !ok {
		return nil
	}
	var out []Perform
	for _, c := range childNodes(b.node) {
		if kindOf(c) == parser.KindPerform {
			out = append(out, Perform{c})
		}
	}
	return out
}
