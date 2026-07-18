package cst

import (
	"fmt"
	"sync"
	"testing"

	"github.com/gaarutyunov/sysgo/engine/text"
)

// Kinds for the benchmark/stress trees (distinct from the small test kinds).
const (
	kProg RawKind = 100
	kStmt RawKind = 101
	kTok  RawKind = 102
)

// buildWide builds a root with n statement nodes, each a unique token plus a
// shared (deduplicated) ";" token — exercising arena append and hash-consing.
func buildWide(in *text.Interner, n int) *Tree {
	b := NewBuilder(in)
	b.StartNode(kProg)
	for i := 0; i < n; i++ {
		b.StartNode(kStmt)
		b.Token(kTok, fmt.Sprintf("t%d", i))
		b.Token(kTok, ";")
		b.FinishNode()
	}
	b.FinishNode()
	return b.Finish()
}

func BenchmarkBuildTree(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = buildWide(nil, 1000)
	}
}

func BenchmarkTraverseText(b *testing.B) {
	t := buildWide(nil, 1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = t.Root().Text()
	}
}

// TestConcurrentIndependentBuilds exercises the chosen concurrency strategy:
// each goroutine builds its own tree with its own arena, sharing nothing. Must
// be race-clean and correct.
func TestConcurrentIndependentBuilds(t *testing.T) {
	const goroutines = 16
	var wg sync.WaitGroup
	for k := 0; k < goroutines; k++ {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				tr := buildWide(nil, 100)
				if tr.Root().ChildCount() != 100 {
					t.Errorf("goroutine %d: child count = %d, want 100", k, tr.Root().ChildCount())
					return
				}
			}
		}(k)
	}
	wg.Wait()
}

// TestConcurrentSharedInterner confirms builders may share one goroutine-safe
// interner while building separate trees concurrently (ENGINE §5a).
func TestConcurrentSharedInterner(t *testing.T) {
	in := text.NewInterner()
	const goroutines = 16
	var wg sync.WaitGroup
	for k := 0; k < goroutines; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				if tr := buildWide(in, 50); tr.Root().TextLen() == 0 {
					t.Error("empty tree")
					return
				}
			}
		}()
	}
	wg.Wait()
}
