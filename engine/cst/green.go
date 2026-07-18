package cst

import "github.com/gaarutyunov/sysgo/engine/text"

// childRef packs a "token or node" discriminator bit with a 31-bit index into
// the corresponding arena. Storing children as a []childRef (a []uint32 under
// the hood) keeps green nodes pointer-free, which is the whole point of the
// arena design (ENGINE §5, E11).
type childRef uint32

const tokenBit childRef = 1 << 31

func nodeRef(index uint32) childRef  { return childRef(index) }
func tokenRef(index uint32) childRef { return childRef(index) | tokenBit }

func (c childRef) isToken() bool { return c&tokenBit != 0 }
func (c childRef) index() uint32 { return uint32(c &^ tokenBit) }

// greenNode is an interned, immutable internal node. It carries no absolute
// position — only its kind, its total byte width (for on-demand offset
// computation), and its children as arena indices.
type greenNode struct {
	kind     RawKind
	width    text.TextSize
	children []childRef
}

// greenToken is an interned, immutable leaf. Its text is interned in the
// tree's [text.Interner] so equal token texts are stored once; width is cached
// to avoid an interner lookup during traversal.
type greenToken struct {
	kind  RawKind
	text  text.Symbol
	width text.TextSize
}

// tokenKey deduplicates tokens by (kind, interned-text). It is comparable, so
// it can key a Go map directly.
type tokenKey struct {
	kind RawKind
	text text.Symbol
}
