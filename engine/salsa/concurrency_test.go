package salsa

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func BenchmarkQueryHit(b *testing.B) {
	db := New()
	in := NewInput[string, int]("a")
	in.Set(db, "a", 1)
	q := NewQuery("sq", func(c *Ctx, _ string) int { return in.Get(c, "a") * 2 })
	_ = db.Read(context.Background(), func(c *Ctx) { q.Get(c, "x") })

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = db.Read(context.Background(), func(c *Ctx) { q.Get(c, "x") })
	}
}

func BenchmarkInvalidation(b *testing.B) {
	db := New()
	in := NewInput[string, int]("a")
	q := NewQuery("sq", func(c *Ctx, _ string) int { return in.Get(c, "a") * 2 })

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		in.Set(db, "a", i)
		_ = db.Read(context.Background(), func(c *Ctx) { q.Get(c, "x") })
	}
}

// TestConcurrentCancellation runs many reads concurrently, half with a
// pre-cancelled context, and confirms cancellation is honored under concurrency
// while uncancelled reads still succeed.
func TestConcurrentCancellation(t *testing.T) {
	db := New()
	in := NewInput[string, int]("a")
	in.Set(db, "x", 7)
	q := NewQuery("id", func(c *Ctx, k string) int { return in.Get(c, k) })

	const goroutines = 32
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			cancelled := g%2 == 0
			if cancelled {
				cancel()
			} else {
				defer cancel()
			}
			var v int
			err := db.Read(ctx, func(c *Ctx) { v = q.Get(c, "x") })
			switch {
			case cancelled && !errors.Is(err, context.Canceled):
				t.Errorf("goroutine %d: want context.Canceled, got %v", g, err)
			case !cancelled && err != nil:
				t.Errorf("goroutine %d: unexpected error %v", g, err)
			case !cancelled && v != 7:
				t.Errorf("goroutine %d: v = %d, want 7", g, v)
			}
		}(g)
	}
	wg.Wait()
}
