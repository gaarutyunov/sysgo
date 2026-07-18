package ast

import (
	"github.com/gaarutyunov/sysgo/engine/cst"
	"github.com/gaarutyunov/sysgo/engine/parser"
)

// Node is implemented by every typed AST wrapper. Syntax returns the underlying
// CST node, the single source of truth the wrapper is a view over.
type Node interface {
	Syntax() cst.Node
}

// kindOf returns the parser syntax kind of a CST node.
func kindOf(n cst.Node) parser.SyntaxKind { return parser.SyntaxKind(n.Kind()) }

// childNodes returns the direct child nodes (tokens excluded).
func childNodes(n cst.Node) []cst.Node {
	var out []cst.Node
	for _, c := range n.Children() {
		if cn, ok := c.(cst.Node); ok {
			out = append(out, cn)
		}
	}
	return out
}

// firstChildOfKind returns the first direct child node of kind k.
func firstChildOfKind(n cst.Node, k parser.SyntaxKind) (cst.Node, bool) {
	for _, c := range childNodes(n) {
		if kindOf(c) == k {
			return c, true
		}
	}
	return cst.Node{}, false
}

// firstToken returns the text of the first direct significant (non-trivia)
// token child, and whether one was found.
func firstToken(n cst.Node) (string, bool) {
	for _, c := range n.Children() {
		if tok, ok := c.(cst.Token); ok && !kindIsTrivia(tok.Kind()) {
			return tok.Text(), true
		}
	}
	return "", false
}

func kindIsTrivia(k cst.RawKind) bool { return parser.SyntaxKind(k).IsTrivia() }

// Inspect traverses the CST rooted at n in depth-first pre-order, calling f for
// each node. If f returns false, the children of that node are not visited. It
// mirrors go/ast.Inspect.
func Inspect(n cst.Node, f func(cst.Node) bool) {
	if !f(n) {
		return
	}
	for _, c := range childNodes(n) {
		Inspect(c, f)
	}
}
