package text

import (
	"fmt"
	"sync"
	"testing"
)

func TestInternerEmptyIsZeroValue(t *testing.T) {
	in := NewInterner()
	if got := in.Intern(""); got != EmptySymbol {
		t.Fatalf("Intern(%q) = %d, want EmptySymbol (%d)", "", got, EmptySymbol)
	}
	if got := in.Lookup(EmptySymbol); got != "" {
		t.Fatalf("Lookup(EmptySymbol) = %q, want %q", got, "")
	}
	if got := in.Len(); got != 1 {
		t.Fatalf("Len() = %d, want 1 (empty string pre-interned)", got)
	}
}

func TestInternerStableAndDistinct(t *testing.T) {
	in := NewInterner()

	a1 := in.Intern("alpha")
	b := in.Intern("beta")
	a2 := in.Intern("alpha")

	if a1 != a2 {
		t.Errorf("Intern(%q) returned %d then %d; want stable", "alpha", a1, a2)
	}
	if a1 == b {
		t.Errorf("distinct strings interned to the same Symbol %d", a1)
	}
	if got := in.Lookup(a1); got != "alpha" {
		t.Errorf("Lookup(%d) = %q, want %q", a1, got, "alpha")
	}
	if got := in.Lookup(b); got != "beta" {
		t.Errorf("Lookup(%d) = %q, want %q", b, got, "beta")
	}
	if got := in.Len(); got != 3 { // "", alpha, beta
		t.Errorf("Len() = %d, want 3", got)
	}
}

func TestInternerLookupOutOfRangePanics(t *testing.T) {
	in := NewInterner()
	in.Intern("only")
	assertPanics(t, "Lookup out of range", func() {
		in.Lookup(Symbol(999))
	})
}

func TestInternerConcurrent(t *testing.T) {
	in := NewInterner()
	const goroutines = 16
	const words = 200

	var wg sync.WaitGroup
	got := make([]map[string]Symbol, goroutines)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			m := make(map[string]Symbol, words)
			for i := 0; i < words; i++ {
				s := fmt.Sprintf("id-%d", i)
				m[s] = in.Intern(s)
			}
			got[g] = m
		}(g)
	}
	wg.Wait()

	// Every goroutine must agree on the Symbol for each string.
	base := got[0]
	for g := 1; g < goroutines; g++ {
		for s, sym := range got[g] {
			if base[s] != sym {
				t.Fatalf("goroutine %d disagrees on %q: %d vs %d", g, s, sym, base[s])
			}
		}
	}
	// words distinct strings + the pre-interned empty string.
	if got := in.Len(); got != words+1 {
		t.Fatalf("Len() = %d, want %d", got, words+1)
	}
}
