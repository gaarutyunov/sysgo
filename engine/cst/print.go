package cst

import (
	"fmt"
	"strings"
)

// Print renders the tree rooted at n as an indented S-expression, one element
// per line, with each element's absolute range. Tokens also show their quoted
// text. The namer maps [RawKind] tags to names; if namer is nil, kinds are
// printed numerically. Print is intended for debugging and golden-file tests.
func Print(n Node, namer KindNamer) string {
	var b strings.Builder
	printNode(&b, n, namer, 0)
	return b.String()
}

func kindName(kind RawKind, namer KindNamer) string {
	if namer != nil {
		if name := namer(kind); name != "" {
			return name
		}
	}
	return fmt.Sprintf("%d", kind)
}

func printNode(b *strings.Builder, n Node, namer KindNamer, depth int) {
	fmt.Fprintf(b, "%s%s %s\n", strings.Repeat("  ", depth), kindName(n.Kind(), namer), n.Range())
	for _, child := range n.Children() {
		switch c := child.(type) {
		case Node:
			printNode(b, c, namer, depth+1)
		case Token:
			fmt.Fprintf(b, "%s%s %s %q\n", strings.Repeat("  ", depth+1), kindName(c.Kind(), namer), c.Range(), c.Text())
		}
	}
}
