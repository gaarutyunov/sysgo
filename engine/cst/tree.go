package cst

import "github.com/gaarutyunov/sysgo/engine/text"

// Tree is an immutable green tree plus the arenas backing it. It is safe for
// concurrent readers. Navigate it through on-demand red cursors starting at
// [Tree.Root].
type Tree struct {
	interner  *text.Interner
	nodes     []greenNode
	tokens    []greenToken
	root      uint32
	rootWidth text.TextSize
}

// Root returns the red cursor for the tree's root node, positioned at offset 0.
func (t *Tree) Root() Node {
	return Node{tree: t, green: t.root, offset: 0, parent: nil}
}

// NodeCount reports the number of distinct green nodes stored. After
// deduplication this is the count of unique subtrees, not the number of
// StartNode calls.
func (t *Tree) NodeCount() int { return len(t.nodes) }

// TokenCount reports the number of distinct green tokens stored (deduplicated
// by kind and text).
func (t *Tree) TokenCount() int { return len(t.tokens) }
