package cst

import (
	"sync"
	"testing"

	"github.com/gaarutyunov/sysgo/engine/text"
)

// buildTwoGroups builds "(1)(1)" as two identical Group subtrees under a Root:
//
//	Root
//	  Group( Paren "(" Num "1" Paren ")" )
//	  Group( Paren "(" Num "1" Paren ")" )
func buildTwoGroups(in *text.Interner) *Tree {
	b := NewBuilder(in)
	b.StartNode(kRoot)
	for i := 0; i < 2; i++ {
		b.StartNode(kGroup)
		b.Token(kParen, "(")
		b.Token(kNum, "1")
		b.Token(kParen, ")")
		b.FinishNode()
	}
	b.FinishNode()
	return b.Finish()
}

func TestGreenDedup(t *testing.T) {
	tree := buildTwoGroups(nil)

	// Identical Group subtrees are hash-consed to one green node; with Root that
	// is 2 distinct nodes, not 3.
	if got := tree.NodeCount(); got != 2 {
		t.Errorf("NodeCount = %d, want 2 (Group deduped + Root)", got)
	}
	// Tokens "(", "1", ")" — "(" and ")" differ, "1" once. 3 distinct.
	if got := tree.TokenCount(); got != 3 {
		t.Errorf("TokenCount = %d, want 3", got)
	}

	// Despite sharing one green node, the two cursors sit at different offsets.
	root := tree.Root()
	g0, g1 := root.Child(0).(Node), root.Child(1).(Node)
	if g0.green != g1.green {
		t.Errorf("expected the two groups to share a green node, got %d and %d", g0.green, g1.green)
	}
	if g0.Range() != text.NewRange(0, 3) {
		t.Errorf("group0 range = %v, want [0,3)", g0.Range())
	}
	if g1.Range() != text.NewRange(3, 6) {
		t.Errorf("group1 range = %v, want [3,6)", g1.Range())
	}
	if got := root.Text(); got != "(1)(1)" {
		t.Errorf("root Text = %q, want %q", got, "(1)(1)")
	}
}

func TestTokenDedupByKind(t *testing.T) {
	b := NewBuilder(nil)
	b.StartNode(kRoot)
	b.Token(kNum, "x")
	b.Token(kOp, "x")  // same text, different kind -> a distinct token
	b.Token(kNum, "x") // duplicate of the first -> deduped
	b.FinishNode()
	tree := b.Finish()
	if got := tree.TokenCount(); got != 2 {
		t.Errorf("TokenCount = %d, want 2 (kind participates in dedup)", got)
	}
}

func TestSharedInternerAcrossTrees(t *testing.T) {
	in := text.NewInterner()
	t1 := buildBinExpr(in)
	t2 := buildBinExpr(in)
	if t1.Root().Text() != t2.Root().Text() {
		t.Error("two trees from one interner produced different text")
	}
}

func TestConcurrentReaders(t *testing.T) {
	tree := buildTwoGroups(nil)
	var wg sync.WaitGroup
	for g := 0; g < 32; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if tree.Root().Text() != "(1)(1)" {
					t.Error("concurrent reader saw wrong text")
					return
				}
				_ = Print(tree.Root(), namer)
			}
		}()
	}
	wg.Wait()
}
