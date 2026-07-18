package cst

import (
	"testing"

	"github.com/gaarutyunov/sysgo/engine/text"
)

// Test syntax kinds, shared across the package's tests.
const (
	kRoot RawKind = iota + 1
	kBinExpr
	kGroup
	kNum
	kOp
	kParen
	kWS
	kError
)

var kindNames = map[RawKind]string{
	kRoot: "Root", kBinExpr: "BinExpr", kGroup: "Group",
	kNum: "Num", kOp: "Op", kParen: "Paren", kWS: "WS", kError: "Error",
}

func namer(k RawKind) string { return kindNames[k] }

// buildBinExpr builds the tree for "1 + 2":
//
//	Root
//	  BinExpr
//	    Num "1"  WS " "  Op "+"  WS " "  Num "2"
func buildBinExpr(in *text.Interner) *Tree {
	b := NewBuilder(in)
	b.StartNode(kRoot)
	b.StartNode(kBinExpr)
	b.Token(kNum, "1")
	b.Token(kWS, " ")
	b.Token(kOp, "+")
	b.Token(kWS, " ")
	b.Token(kNum, "2")
	b.FinishNode() // BinExpr
	b.FinishNode() // Root
	return b.Finish()
}

func TestBuildAndRoundTrip(t *testing.T) {
	tree := buildBinExpr(nil)
	root := tree.Root()

	if got := root.Kind(); got != kRoot {
		t.Errorf("root kind = %d, want kRoot", got)
	}
	if got := root.Text(); got != "1 + 2" {
		t.Errorf("root Text() = %q, want %q", got, "1 + 2")
	}
	if got := root.Range(); got != text.NewRange(0, 5) {
		t.Errorf("root Range() = %v, want [0,5)", got)
	}
	if got := root.ChildCount(); got != 1 {
		t.Fatalf("root ChildCount = %d, want 1", got)
	}
}

func TestChildOffsetsAndRanges(t *testing.T) {
	tree := buildBinExpr(nil)
	bin := tree.Root().Child(0).(Node)
	if bin.Kind() != kBinExpr {
		t.Fatalf("child kind = %d, want kBinExpr", bin.Kind())
	}

	want := []struct {
		kind RawKind
		text string
		rng  text.TextRange
	}{
		{kNum, "1", text.NewRange(0, 1)},
		{kWS, " ", text.NewRange(1, 2)},
		{kOp, "+", text.NewRange(2, 3)},
		{kWS, " ", text.NewRange(3, 4)},
		{kNum, "2", text.NewRange(4, 5)},
	}
	if got := bin.ChildCount(); got != len(want) {
		t.Fatalf("BinExpr ChildCount = %d, want %d", got, len(want))
	}
	for i, w := range want {
		c := bin.Child(i)
		if !c.IsToken() {
			t.Errorf("child %d IsToken = false, want true", i)
		}
		if c.Kind() != w.kind {
			t.Errorf("child %d kind = %d, want %d", i, c.Kind(), w.kind)
		}
		if c.Text() != w.text {
			t.Errorf("child %d Text = %q, want %q", i, c.Text(), w.text)
		}
		if c.Range() != w.rng {
			t.Errorf("child %d Range = %v, want %v", i, c.Range(), w.rng)
		}
	}
}

func TestChildrenMatchesChild(t *testing.T) {
	tree := buildBinExpr(nil)
	bin := tree.Root().Child(0).(Node)
	all := bin.Children()
	if len(all) != bin.ChildCount() {
		t.Fatalf("Children len %d != ChildCount %d", len(all), bin.ChildCount())
	}
	for i := range all {
		if all[i].Range() != bin.Child(i).Range() || all[i].Kind() != bin.Child(i).Kind() {
			t.Errorf("Children[%d] disagrees with Child(%d)", i, i)
		}
	}
}

func TestParentNavigation(t *testing.T) {
	tree := buildBinExpr(nil)
	root := tree.Root()
	if _, ok := root.Parent(); ok {
		t.Error("root Parent() ok = true, want false")
	}
	bin := root.Child(0).(Node)
	p, ok := bin.Parent()
	if !ok {
		t.Fatal("BinExpr Parent() ok = false, want true")
	}
	if p.Kind() != kRoot {
		t.Errorf("BinExpr parent kind = %d, want kRoot", p.Kind())
	}
	tok := bin.Child(2).(Token) // "+"
	tp, ok := tok.Parent()
	if !ok || tp.Kind() != kBinExpr {
		t.Errorf("token parent = (%d, %v), want (kBinExpr, true)", tp.Kind(), ok)
	}
}

func TestPrint(t *testing.T) {
	tree := buildBinExpr(nil)
	want := "" +
		"Root [0, 5)\n" +
		"  BinExpr [0, 5)\n" +
		"    Num [0, 1) \"1\"\n" +
		"    WS [1, 2) \" \"\n" +
		"    Op [2, 3) \"+\"\n" +
		"    WS [3, 4) \" \"\n" +
		"    Num [4, 5) \"2\"\n"
	if got := Print(tree.Root(), namer); got != want {
		t.Errorf("Print() =\n%q\nwant\n%q", got, want)
	}
}
