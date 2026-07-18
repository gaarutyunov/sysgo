package hir

import (
	"fmt"
	"testing"
)

func manyUnits(n int) []Unit {
	units := make([]Unit, n)
	for i := 0; i < n; i++ {
		units[i] = Unit{
			Key:    fmt.Sprintf("f%d.sysml", i),
			Source: fmt.Sprintf("package P%d {\n\tpart def X%d;\n}", i, i),
		}
	}
	return units
}

// TestAnalyzeUnitsConcurrentParseDeterministic checks that the concurrent
// parse in AnalyzeUnits yields a correct, deterministic combined model (run
// under -race to catch data races in the parse fan-out).
func TestAnalyzeUnitsConcurrentParseDeterministic(t *testing.T) {
	units := manyUnits(24)

	r1 := AnalyzeUnits(units)
	if len(r1.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %v", r1.Diagnostics)
	}
	for i := 0; i < 24; i++ {
		if _, ok := r1.Model.Root.Member(fmt.Sprintf("P%d", i)); !ok {
			t.Errorf("missing package P%d", i)
		}
	}

	// Determinism: repeated analysis produces the same top-level shape.
	r2 := AnalyzeUnits(units)
	if got1, got2 := len(r1.Model.Root.Children()), len(r2.Model.Root.Children()); got1 != got2 || got1 != 24 {
		t.Errorf("child counts = %d / %d, want 24", got1, got2)
	}
	order1 := make([]string, 0, 24)
	for _, c := range r1.Model.Root.Children() {
		order1 = append(order1, c.Name)
	}
	for i, c := range r2.Model.Root.Children() {
		if c.Name != order1[i] {
			t.Errorf("non-deterministic order at %d: %q vs %q", i, c.Name, order1[i])
		}
	}
}

func BenchmarkAnalyzeUnits(b *testing.B) {
	units := manyUnits(50)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AnalyzeUnits(units)
	}
}
