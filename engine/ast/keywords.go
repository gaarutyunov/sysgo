package ast

import (
	"github.com/gaarutyunov/sysgo/engine/cst"
	"github.com/gaarutyunov/sysgo/engine/parser"
)

// Keywords returns the declaration's leading identifier tokens in source order —
// its modifiers (e.g. in / out / abstract), its kind keyword (e.g. part /
// action / item / classifier), and an optional "def" — everything before the
// declared name. Prefix visibility and annotations are skipped; the run ends at
// the name or the first non-identifier token.
func (d Declaration) Keywords() []string {
	var out []string
	for _, c := range d.node.Children() {
		switch e := c.(type) {
		case cst.Token:
			if kindIsTrivia(e.Kind()) {
				continue
			}
			if parser.SyntaxKind(e.Kind()) == parser.KindIdent {
				out = append(out, e.Text())
				continue
			}
			return out // a non-identifier significant token ends the keyword run
		case cst.Node:
			switch kindOf(e) {
			case parser.KindVisibility, parser.KindAnnotation:
				continue // prefix nodes precede the keywords
			default:
				return out // the name (QualifiedName) or later structure
			}
		}
	}
	return out
}
