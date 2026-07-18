package ast

import (
	"strings"

	"github.com/gaarutyunov/sysgo/engine/cst"
)

// Format renders an AST node back to source text by walking its tokens in
// document order. Because the CST preserves every token (including trivia),
// formatting a parsed tree is faithful: the output equals the original source
// bytes exactly.
//
// This is a lossless printer, not a canonical pretty-printer; re-indenting /
// normalizing formatting is a later addition on the same traversal.
func Format(n Node) string {
	var b strings.Builder
	writeTokens(n.Syntax(), &b)
	return b.String()
}

func writeTokens(n cst.Node, b *strings.Builder) {
	for _, c := range n.Children() {
		switch e := c.(type) {
		case cst.Token:
			b.WriteString(e.Text())
		case cst.Node:
			writeTokens(e, b)
		}
	}
}
