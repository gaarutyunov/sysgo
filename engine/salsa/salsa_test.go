package salsa

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

// read is a small helper that runs a single query read and returns its value.
func readInt(t *testing.T, db *Db, q *Query[string, int], k string) int {
	t.Helper()
	var out int
	if err := db.Read(context.Background(), func(c *Ctx) { out = q.Get(c, k) }); err != nil {
		t.Fatalf("Read error: %v", err)
	}
	return out
}

func TestInputAndQueryBasic(t *testing.T) {
	db := New()
	a := NewInput[string, int]("a")
	a.Set(db, "x", 10)

	q := NewQuery("plus1", func(c *Ctx, k string) int { return a.Get(c, k) + 1 })
	if got := readInt(t, db, q, "x"); got != 11 {
		t.Errorf("got %d, want 11", got)
	}
}

func TestUnsetInputIsZeroThenInvalidates(t *testing.T) {
	db := New()
	a := NewInput[string, int]("a")
	q := NewQuery("id", func(c *Ctx, k string) int { return a.Get(c, k) })

	if got := readInt(t, db, q, "x"); got != 0 {
		t.Errorf("unset input query = %d, want 0", got)
	}
	a.Set(db, "x", 7)
	if got := readInt(t, db, q, "x"); got != 7 {
		t.Errorf("after Set = %d, want 7", got)
	}
}

func TestMemoizationAndInvalidation(t *testing.T) {
	db := New()
	a := NewInput[string, int]("a")
	b := NewInput[string, int]("b")
	a.Set(db, "a", 1)
	b.Set(db, "b", 2)

	var sumCalls, doubleCalls int
	sum := NewQuery("sum", func(c *Ctx, _ string) int {
		sumCalls++
		return a.Get(c, "a") + b.Get(c, "b")
	})
	double := NewQuery("double", func(c *Ctx, k string) int {
		doubleCalls++
		return sum.Get(c, k) * 2
	})

	if got := readInt(t, db, double, "x"); got != 6 {
		t.Fatalf("double = %d, want 6", got)
	}
	if sumCalls != 1 || doubleCalls != 1 {
		t.Fatalf("first read calls = (%d,%d), want (1,1)", sumCalls, doubleCalls)
	}

	// Same revision: pure cache hit, no recompute.
	readInt(t, db, double, "x")
	if sumCalls != 1 || doubleCalls != 1 {
		t.Errorf("cached read recomputed: (%d,%d)", sumCalls, doubleCalls)
	}

	// No-op set (equal value) does not bump the revision or recompute.
	a.Set(db, "a", 1)
	if rev := db.Revision(); rev != 2 { // two initial Sets only
		t.Errorf("revision after no-op set = %d, want 2", rev)
	}
	readInt(t, db, double, "x")
	if sumCalls != 1 || doubleCalls != 1 {
		t.Errorf("no-op set recomputed: (%d,%d)", sumCalls, doubleCalls)
	}

	// Real change: both recompute.
	a.Set(db, "a", 5)
	if got := readInt(t, db, double, "x"); got != 14 {
		t.Fatalf("after change double = %d, want 14", got)
	}
	if sumCalls != 2 || doubleCalls != 2 {
		t.Errorf("after change calls = (%d,%d), want (2,2)", sumCalls, doubleCalls)
	}

	// Change b to an equal value: no bump, no recompute.
	b.Set(db, "b", 2)
	readInt(t, db, double, "x")
	if sumCalls != 2 || doubleCalls != 2 {
		t.Errorf("equal-b set recomputed: (%d,%d)", sumCalls, doubleCalls)
	}
}

func TestBackdatingMinimizesRecompute(t *testing.T) {
	db := New()
	x := NewInput[string, int]("x")
	x.Set(db, "x", 1)

	var posCalls, depCalls int
	isPos := NewQuery("isPos", func(c *Ctx, _ string) bool {
		posCalls++
		return x.Get(c, "x") > 0
	})
	dep := NewQuery("dep", func(c *Ctx, k string) string {
		depCalls++
		if isPos.Get(c, k) {
			return "yes"
		}
		return "no"
	})

	readDep := func() string {
		var out string
		if err := db.Read(context.Background(), func(c *Ctx) { out = dep.Get(c, "k") }); err != nil {
			t.Fatalf("Read: %v", err)
		}
		return out
	}

	if got := readDep(); got != "yes" {
		t.Fatalf("dep = %q, want yes", got)
	}
	if posCalls != 1 || depCalls != 1 {
		t.Fatalf("initial (%d,%d), want (1,1)", posCalls, depCalls)
	}

	// x changes 1 -> 2: isPos recomputes but stays true, so dep is NOT
	// recomputed (its input's value did not change — the essence of backdating).
	x.Set(db, "x", 2)
	if got := readDep(); got != "yes" {
		t.Fatalf("dep = %q, want yes", got)
	}
	if posCalls != 2 {
		t.Errorf("isPos calls = %d, want 2 (recomputed)", posCalls)
	}
	if depCalls != 1 {
		t.Errorf("dep calls = %d, want 1 (backdated, not recomputed)", depCalls)
	}

	// x changes 2 -> -1: isPos flips to false, so dep must recompute.
	x.Set(db, "x", -1)
	if got := readDep(); got != "no" {
		t.Fatalf("dep = %q, want no", got)
	}
	if posCalls != 3 || depCalls != 2 {
		t.Errorf("after flip (%d,%d), want (3,2)", posCalls, depCalls)
	}
}

func TestCycleDetection(t *testing.T) {
	db := New()
	var qa, qb *Query[string, int]
	qa = NewQuery("qa", func(c *Ctx, k string) int { return qb.Get(c, k) + 1 })
	qb = NewQuery("qb", func(c *Ctx, k string) int { return qa.Get(c, k) + 1 })

	err := db.Read(context.Background(), func(c *Ctx) { _ = qa.Get(c, "x") })
	var ce CycleError
	if !errors.As(err, &ce) {
		t.Fatalf("err = %v, want CycleError", err)
	}
}

func TestCancellation(t *testing.T) {
	db := New()
	a := NewInput[string, int]("a")
	a.Set(db, "x", 1)
	q := NewQuery("id", func(c *Ctx, k string) int { return a.Get(c, k) })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := db.Read(ctx, func(c *Ctx) { _ = q.Get(c, "x") })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	db := New()
	a := NewInput[string, int]("a")
	a.Set(db, "a", 0)
	var computes int64
	q := NewQuery("sq", func(c *Ctx, _ string) int {
		atomic.AddInt64(&computes, 1)
		v := a.Get(c, "a")
		return v * v
	})

	var wg sync.WaitGroup
	// Writers.
	for w := 1; w <= 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				a.Set(db, "a", w*i)
			}
		}(w)
	}
	// Readers.
	for r := 0; r < 8; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				var v, in int
				err := db.Read(context.Background(), func(c *Ctx) {
					in = a.Get(c, "a")
					v = q.Get(c, "sq")
				})
				if err != nil {
					t.Errorf("Read: %v", err)
					return
				}
				if v != in*in {
					t.Errorf("inconsistent snapshot: q=%d in=%d", v, in)
					return
				}
			}
		}()
	}
	wg.Wait()
}
