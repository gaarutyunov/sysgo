package cst

import (
	"testing"

	"github.com/gaarutyunov/sysgo/engine/text"
)

// TestCheckpointWrap emits "1+2" flat, then retroactively wraps the three
// tokens in a BinExpr via a checkpoint — the pattern a Pratt parser uses once it
// discovers an operator after already emitting the left operand.
func TestCheckpointWrap(t *testing.T) {
	b := NewBuilder(nil)
	b.StartNode(kRoot)
	cp := b.Checkpoint()
	b.Token(kNum, "1")
	b.Token(kOp, "+")
	b.Token(kNum, "2")
	b.StartNodeAt(cp, kBinExpr)
	b.FinishNode() // BinExpr
	b.FinishNode() // Root
	tree := b.Finish()

	root := tree.Root()
	if root.ChildCount() != 1 {
		t.Fatalf("root ChildCount = %d, want 1 (the wrapped BinExpr)", root.ChildCount())
	}
	bin := root.Child(0).(Node)
	if bin.Kind() != kBinExpr {
		t.Fatalf("wrapped node kind = %d, want kBinExpr", bin.Kind())
	}
	if bin.ChildCount() != 3 {
		t.Errorf("BinExpr ChildCount = %d, want 3", bin.ChildCount())
	}
	if got := root.Text(); got != "1+2" {
		t.Errorf("root Text = %q, want %q", got, "1+2")
	}
	if bin.Range() != text.NewRange(0, 3) {
		t.Errorf("BinExpr range = %v, want [0,3)", bin.Range())
	}
}

// TestCheckpointPartialWrap wraps only the children emitted after the
// checkpoint, leaving earlier siblings alone.
func TestCheckpointPartialWrap(t *testing.T) {
	b := NewBuilder(nil)
	b.StartNode(kRoot)
	b.Token(kNum, "0")   // stays a direct child of Root
	cp := b.Checkpoint() // checkpoint after the first token
	b.Token(kNum, "1")
	b.Token(kOp, "+")
	b.Token(kNum, "2")
	b.StartNodeAt(cp, kBinExpr)
	b.FinishNode()
	b.FinishNode()
	tree := b.Finish()

	root := tree.Root()
	if root.ChildCount() != 2 {
		t.Fatalf("root ChildCount = %d, want 2 (Num + BinExpr)", root.ChildCount())
	}
	if !root.Child(0).IsToken() || root.Child(0).Text() != "0" {
		t.Errorf("root child 0 = %v, want token %q", root.Child(0), "0")
	}
	if root.Child(1).(Node).Kind() != kBinExpr {
		t.Errorf("root child 1 kind = %d, want kBinExpr", root.Child(1).Kind())
	}
}

func TestErrorNodeTolerance(t *testing.T) {
	// Any node may hold arbitrary children, including an error node with a
	// stray token — incomplete input still yields a lossless tree.
	b := NewBuilder(nil)
	b.StartNode(kRoot)
	b.Token(kNum, "1")
	b.StartNode(kError)
	b.Token(kOp, "@")
	b.FinishNode()
	b.FinishNode()
	tree := b.Finish()
	if got := tree.Root().Text(); got != "1@" {
		t.Errorf("root Text = %q, want %q", got, "1@")
	}
	if tree.Root().Child(1).(Node).Kind() != kError {
		t.Error("expected an Error node as the second child")
	}
}

func TestBuilderPanics(t *testing.T) {
	assertPanics(t, "FinishNode without StartNode", func() {
		b := NewBuilder(nil)
		b.FinishNode()
	})
	assertPanics(t, "Finish with open node", func() {
		b := NewBuilder(nil)
		b.StartNode(kRoot)
		b.Finish()
	})
	assertPanics(t, "Finish with no root", func() {
		b := NewBuilder(nil)
		b.Finish()
	})
	assertPanics(t, "Finish with a token root", func() {
		b := NewBuilder(nil)
		b.Token(kNum, "1")
		b.Finish()
	})
	assertPanics(t, "StartNodeAt at wrong level", func() {
		b := NewBuilder(nil)
		b.StartNode(kRoot)
		cp := b.Checkpoint()
		b.StartNode(kGroup) // descend a level
		b.StartNodeAt(cp, kBinExpr)
	})
}

// assertPanics fails the test if fn does not panic.
func assertPanics(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected panic, got none", what)
		}
	}()
	fn()
}
