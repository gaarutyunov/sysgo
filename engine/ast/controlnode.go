package ast

import (
	"github.com/gaarutyunov/sysgo/engine/cst"
	"github.com/gaarutyunov/sysgo/engine/parser"
)

// ControlNode is a fork / join / merge / decide control node in an action body.
type ControlNode struct{ node cst.Node }

func (c ControlNode) Syntax() cst.Node { return c.node }

// Kind returns the control-node keyword: "fork", "join", "merge", or "decide".
func (c ControlNode) Kind() string {
	if t, ok := firstToken(c.node); ok {
		return t
	}
	return ""
}

// Name returns the control node's local name, or "" if anonymous.
func (c ControlNode) Name() string {
	if qn, ok := firstChildOfKind(c.node, parser.KindQualifiedName); ok {
		return QualifiedName{qn}.String()
	}
	return ""
}

// ControlNodes returns the declaration body's control nodes in order.
func (d Declaration) ControlNodes() []ControlNode {
	b, ok := d.Body()
	if !ok {
		return nil
	}
	var out []ControlNode
	for _, c := range childNodes(b.node) {
		if kindOf(c) == parser.KindControlNode {
			out = append(out, ControlNode{c})
		}
	}
	return out
}
