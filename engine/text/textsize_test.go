package text

import (
	"math"
	"testing"
)

// assertPanics fails the test if fn does not panic. Shared across the package's
// tests.
func assertPanics(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected panic, got none", what)
		}
	}()
	fn()
}

func TestSizeOf(t *testing.T) {
	// Byte length, not rune count: "é" is two UTF-8 bytes.
	tests := []struct {
		in   string
		want TextSize
	}{
		{"", 0},
		{"abc", 3},
		{"é", 2},
		{"héllo", 6},
	}
	for _, tt := range tests {
		if got := SizeOf(tt.in); got != tt.want {
			t.Errorf("SizeOf(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestTextSizeAddSub(t *testing.T) {
	if got := TextSize(5).Add(3); got != 8 {
		t.Errorf("5.Add(3) = %d, want 8", got)
	}
	if got := TextSize(5).Sub(3); got != 2 {
		t.Errorf("5.Sub(3) = %d, want 2", got)
	}
	assertPanics(t, "Add overflow", func() {
		TextSize(math.MaxUint32).Add(1)
	})
	assertPanics(t, "Sub underflow", func() {
		TextSize(3).Sub(5)
	})
}

func TestNewRangeAndConstructors(t *testing.T) {
	r := NewRange(2, 5)
	if r.Start != 2 || r.End != 5 {
		t.Errorf("NewRange(2,5) = %v, want [2,5)", r)
	}
	if got := RangeAt(2, 3); got != (TextRange{2, 5}) {
		t.Errorf("RangeAt(2,3) = %v, want [2,5)", got)
	}
	if got := EmptyRange(4); got != (TextRange{4, 4}) {
		t.Errorf("EmptyRange(4) = %v, want [4,4)", got)
	}
	assertPanics(t, "NewRange end<start", func() {
		NewRange(5, 2)
	})
}

func TestTextRangeLenEmpty(t *testing.T) {
	if got := NewRange(2, 5).Len(); got != 3 {
		t.Errorf("Len [2,5) = %d, want 3", got)
	}
	if !EmptyRange(4).IsEmpty() {
		t.Error("EmptyRange(4).IsEmpty() = false, want true")
	}
	if NewRange(2, 5).IsEmpty() {
		t.Error("[2,5).IsEmpty() = true, want false")
	}
}

func TestTextRangeContains(t *testing.T) {
	r := NewRange(2, 5)
	tests := []struct {
		off        TextSize
		half, incl bool
	}{
		{1, false, false},
		{2, true, true},
		{4, true, true},
		{5, false, true}, // End is excluded by Contains, included by ContainsInclusive
		{6, false, false},
	}
	for _, tt := range tests {
		if got := r.Contains(tt.off); got != tt.half {
			t.Errorf("[2,5).Contains(%d) = %v, want %v", tt.off, got, tt.half)
		}
		if got := r.ContainsInclusive(tt.off); got != tt.incl {
			t.Errorf("[2,5).ContainsInclusive(%d) = %v, want %v", tt.off, got, tt.incl)
		}
	}
	// An empty range contains no offset, not even its own position.
	if EmptyRange(3).Contains(3) {
		t.Error("EmptyRange(3).Contains(3) = true, want false")
	}
}

func TestTextRangeContainsRange(t *testing.T) {
	outer := NewRange(2, 8)
	tests := []struct {
		inner TextRange
		want  bool
	}{
		{NewRange(2, 8), true},  // equal
		{NewRange(3, 5), true},  // strictly inside
		{NewRange(2, 5), true},  // shares start
		{NewRange(5, 8), true},  // shares end
		{NewRange(1, 5), false}, // spills left
		{NewRange(5, 9), false}, // spills right
		{EmptyRange(4), true},   // empty, inside
	}
	for _, tt := range tests {
		if got := outer.ContainsRange(tt.inner); got != tt.want {
			t.Errorf("[2,8).ContainsRange(%v) = %v, want %v", tt.inner, got, tt.want)
		}
	}
}

func TestTextRangeIntersect(t *testing.T) {
	tests := []struct {
		a, b   TextRange
		want   TextRange
		wantOK bool
	}{
		{NewRange(2, 6), NewRange(4, 8), NewRange(4, 6), true}, // overlap
		{NewRange(2, 6), NewRange(0, 3), NewRange(2, 3), true}, // overlap left
		{NewRange(2, 6), NewRange(2, 6), NewRange(2, 6), true}, // equal
		{NewRange(1, 3), NewRange(3, 5), TextRange{}, false},   // touch, no overlap
		{NewRange(1, 2), NewRange(5, 9), TextRange{}, false},   // disjoint
	}
	for _, tt := range tests {
		got, ok := tt.a.Intersect(tt.b)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("%v.Intersect(%v) = (%v, %v), want (%v, %v)",
				tt.a, tt.b, got, ok, tt.want, tt.wantOK)
		}
		// Intersect must be symmetric.
		got2, ok2 := tt.b.Intersect(tt.a)
		if ok2 != tt.wantOK || got2 != tt.want {
			t.Errorf("%v.Intersect(%v) asymmetric: (%v,%v) vs (%v,%v)",
				tt.b, tt.a, got2, ok2, tt.want, tt.wantOK)
		}
	}
}

func TestTextRangeCover(t *testing.T) {
	tests := []struct {
		a, b, want TextRange
	}{
		{NewRange(2, 4), NewRange(6, 8), NewRange(2, 8)}, // disjoint bounding
		{NewRange(2, 6), NewRange(4, 8), NewRange(2, 8)}, // overlapping
		{NewRange(2, 8), NewRange(4, 6), NewRange(2, 8)}, // nested
	}
	for _, tt := range tests {
		if got := tt.a.Cover(tt.b); got != tt.want {
			t.Errorf("%v.Cover(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestTextRangeString(t *testing.T) {
	if got := NewRange(2, 5).String(); got != "[2, 5)" {
		t.Errorf("String() = %q, want %q", got, "[2, 5)")
	}
}
