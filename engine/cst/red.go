package cst

import (
	"strings"

	"github.com/gaarutyunov/sysgo/engine/text"
)

// Element is a red cursor over a green child: either a [Node] or a [Token].
// Cursors are cheap, computed on demand, and never persisted (ENGINE §5, E11).
type Element interface {
	// Kind returns the syntax kind tag.
	Kind() RawKind
	// Range returns the absolute half-open byte range within the source.
	Range() text.TextRange
	// Text returns the source text this element covers, reconstructed losslessly.
	Text() string
	// IsToken reports whether the element is a leaf token.
	IsToken() bool

	isElement()
}

// Node is a red cursor over a green internal node: a green arena index plus the
// absolute offset it sits at plus a link to its parent cursor.
type Node struct {
	tree   *Tree
	green  uint32
	offset text.TextSize
	parent *Node
}

// Kind returns the node's syntax kind.
func (n Node) Kind() RawKind { return n.tree.nodes[n.green].kind }

// TextLen returns the node's total byte width, including descendants.
func (n Node) TextLen() text.TextSize { return n.tree.nodes[n.green].width }

// Range returns the node's absolute half-open byte range.
func (n Node) Range() text.TextRange { return text.RangeAt(n.offset, n.TextLen()) }

// ChildCount returns the number of direct children.
func (n Node) ChildCount() int { return len(n.tree.nodes[n.green].children) }

// Child returns the i-th direct child as a [Node] or [Token]. It panics if i is
// out of range.
func (n Node) Child(i int) Element {
	children := n.tree.nodes[n.green].children
	off := n.offset
	for j := 0; j < i; j++ {
		off = off.Add(n.tree.childWidth(children[j]))
	}
	self := n
	return n.tree.element(children[i], off, &self)
}

// Children returns all direct children in document order.
func (n Node) Children() []Element {
	children := n.tree.nodes[n.green].children
	out := make([]Element, len(children))
	self := n
	off := n.offset
	for i, c := range children {
		out[i] = n.tree.element(c, off, &self)
		off = off.Add(n.tree.childWidth(c))
	}
	return out
}

// Parent returns the parent cursor and whether one exists (false at the root).
func (n Node) Parent() (Node, bool) {
	if n.parent == nil {
		return Node{}, false
	}
	return *n.parent, true
}

// Text returns the source text under this node, reconstructed byte-for-byte by
// concatenating descendant token text in document order.
func (n Node) Text() string {
	var b strings.Builder
	b.Grow(int(n.TextLen()))
	n.tree.writeText(n.green, &b)
	return b.String()
}

// IsToken always returns false for a node.
func (n Node) IsToken() bool { return false }

func (n Node) isElement() {}

// Token is a red cursor over a green leaf token.
type Token struct {
	tree   *Tree
	green  uint32
	offset text.TextSize
	parent *Node
}

// Kind returns the token's syntax kind.
func (t Token) Kind() RawKind { return t.tree.tokens[t.green].kind }

// TextLen returns the token's byte width.
func (t Token) TextLen() text.TextSize { return t.tree.tokens[t.green].width }

// Range returns the token's absolute half-open byte range.
func (t Token) Range() text.TextRange { return text.RangeAt(t.offset, t.TextLen()) }

// Text returns the token's source text.
func (t Token) Text() string { return t.tree.interner.Lookup(t.tree.tokens[t.green].text) }

// Parent returns the token's parent node cursor and whether one exists.
func (t Token) Parent() (Node, bool) {
	if t.parent == nil {
		return Node{}, false
	}
	return *t.parent, true
}

// IsToken always returns true for a token.
func (t Token) IsToken() bool { return true }

func (t Token) isElement() {}

func (t *Tree) childWidth(c childRef) text.TextSize {
	if c.isToken() {
		return t.tokens[c.index()].width
	}
	return t.nodes[c.index()].width
}

func (t *Tree) element(c childRef, off text.TextSize, parent *Node) Element {
	if c.isToken() {
		return Token{tree: t, green: c.index(), offset: off, parent: parent}
	}
	return Node{tree: t, green: c.index(), offset: off, parent: parent}
}

func (t *Tree) writeText(nodeIdx uint32, b *strings.Builder) {
	for _, c := range t.nodes[nodeIdx].children {
		if c.isToken() {
			b.WriteString(t.interner.Lookup(t.tokens[c.index()].text))
		} else {
			t.writeText(c.index(), b)
		}
	}
}
