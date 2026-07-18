package cst

import "github.com/gaarutyunov/sysgo/engine/text"

// Builder incrementally constructs a green tree bottom-up and hands back an
// immutable [Tree]. It mirrors rowan's GreenNodeBuilder: the parser emits a
// stream of StartNode / Token / FinishNode events and the builder interns and
// deduplicates nodes and tokens as it goes.
//
// A Builder is not safe for concurrent use; build separate trees on separate
// goroutines. Several builders may share one goroutine-safe [text.Interner].
type Builder struct {
	interner   *text.Interner
	nodes      []greenNode
	tokens     []greenToken
	nodeDedup  map[string]uint32
	tokenDedup map[tokenKey]uint32
	stack      []frame
}

type frame struct {
	kind     RawKind
	children []childRef
}

// Checkpoint marks a position in the current node so it can be wrapped later
// with [Builder.StartNodeAt] — the mechanism a parser uses to retroactively
// group already-emitted children (e.g. on error recovery).
type Checkpoint struct {
	frame int
	n     int
}

// NewBuilder returns an empty Builder that interns token text in in. If in is
// nil a private interner is used.
func NewBuilder(in *text.Interner) *Builder {
	if in == nil {
		in = text.NewInterner()
	}
	return &Builder{
		interner:   in,
		nodeDedup:  make(map[string]uint32),
		tokenDedup: make(map[tokenKey]uint32),
		stack:      []frame{{}}, // base frame collects the root
	}
}

func (b *Builder) top() *frame { return &b.stack[len(b.stack)-1] }

// StartNode begins a new internal node of the given kind. Children emitted
// until the matching [Builder.FinishNode] belong to it.
func (b *Builder) StartNode(kind RawKind) {
	b.stack = append(b.stack, frame{kind: kind})
}

// Token appends a leaf token of the given kind and source text to the current
// node. Trivia (whitespace, comments) is emitted as ordinary tokens so the tree
// stays lossless.
func (b *Builder) Token(kind RawKind, s string) {
	idx := b.internToken(kind, s)
	f := b.top()
	f.children = append(f.children, tokenRef(idx))
}

// Checkpoint records the current position so a later [Builder.StartNodeAt] can
// wrap the children emitted after it. It must be consumed at the same nesting
// level at which it was taken.
func (b *Builder) Checkpoint() Checkpoint {
	return Checkpoint{frame: len(b.stack) - 1, n: len(b.top().children)}
}

// StartNodeAt begins a new node of the given kind that adopts every child
// emitted since cp was taken. It panics if cp was not taken at the current
// nesting level.
func (b *Builder) StartNodeAt(cp Checkpoint, kind RawKind) {
	if cp.frame != len(b.stack)-1 {
		panic("cst: StartNodeAt at a different nesting level than its Checkpoint")
	}
	top := b.top()
	if cp.n > len(top.children) {
		panic("cst: Checkpoint position is out of range")
	}
	stolen := append([]childRef(nil), top.children[cp.n:]...)
	top.children = top.children[:cp.n]
	b.stack = append(b.stack, frame{kind: kind, children: stolen})
}

// FinishNode completes the current node, interning it, and attaches it to its
// parent. It panics if there is no open node.
func (b *Builder) FinishNode() {
	if len(b.stack) <= 1 {
		panic("cst: FinishNode without a matching StartNode")
	}
	f := b.stack[len(b.stack)-1]
	b.stack = b.stack[:len(b.stack)-1]
	idx := b.internNode(f.kind, f.children)
	parent := b.top()
	parent.children = append(parent.children, nodeRef(idx))
}

// Finish returns the immutable tree. It panics unless exactly one root node was
// built and all nodes were finished.
func (b *Builder) Finish() *Tree {
	if len(b.stack) != 1 {
		panic("cst: Finish with an unfinished node open")
	}
	base := b.stack[0]
	if len(base.children) != 1 || base.children[0].isToken() {
		panic("cst: Finish expects exactly one root node")
	}
	root := base.children[0].index()
	return &Tree{
		interner:  b.interner,
		nodes:     b.nodes,
		tokens:    b.tokens,
		root:      root,
		rootWidth: b.nodes[root].width,
	}
}

func (b *Builder) internToken(kind RawKind, s string) uint32 {
	sym := b.interner.Intern(s)
	key := tokenKey{kind: kind, text: sym}
	if idx, ok := b.tokenDedup[key]; ok {
		return idx
	}
	idx := uint32(len(b.tokens))
	b.tokens = append(b.tokens, greenToken{kind: kind, text: sym, width: text.SizeOf(s)})
	b.tokenDedup[key] = idx
	return idx
}

func (b *Builder) internNode(kind RawKind, children []childRef) uint32 {
	key := nodeKey(kind, children)
	if idx, ok := b.nodeDedup[key]; ok {
		return idx
	}
	var w text.TextSize
	for _, c := range children {
		w += b.childWidth(c)
	}
	own := append([]childRef(nil), children...)
	idx := uint32(len(b.nodes))
	b.nodes = append(b.nodes, greenNode{kind: kind, width: w, children: own})
	b.nodeDedup[key] = idx
	return idx
}

func (b *Builder) childWidth(c childRef) text.TextSize {
	if c.isToken() {
		return b.tokens[c.index()].width
	}
	return b.nodes[c.index()].width
}

// nodeKey builds a comparable dedup key from a node's kind and children. Because
// construction is bottom-up and children are already deduplicated, identical
// subtrees produce identical childRefs and therefore identical keys.
func nodeKey(kind RawKind, children []childRef) string {
	buf := make([]byte, 0, 2+len(children)*4)
	buf = append(buf, byte(kind), byte(kind>>8))
	for _, c := range children {
		buf = append(buf, byte(c), byte(c>>8), byte(c>>16), byte(c>>24))
	}
	return string(buf)
}
