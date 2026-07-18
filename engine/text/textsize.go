package text

import (
	"fmt"
	"math"
)

// TextSize is a byte offset or byte length within a source file. It mirrors
// rowan's u32-based TextSize: positions are measured in UTF-8 bytes, not runes,
// so a TextSize indexes directly into the source string.
//
// The u32 range caps a single source file at 4 GiB, which is deliberate — it
// keeps positions compact in the CST arena (ENGINE §5).
type TextSize uint32

// SizeOf returns the length of s as a [TextSize]. It panics if s is longer than
// [math.MaxUint32] bytes.
func SizeOf(s string) TextSize {
	if uint64(len(s)) > math.MaxUint32 {
		panic("text: string length exceeds TextSize range")
	}
	return TextSize(len(s))
}

// Add returns s+other, panicking on overflow. Overflow means a computed
// position exceeded the 4 GiB file cap, which is a bug rather than a runtime
// condition to handle.
func (s TextSize) Add(other TextSize) TextSize {
	sum := s + other
	if sum < s {
		panic("text: TextSize addition overflow")
	}
	return sum
}

// Sub returns s-other, panicking on underflow (other > s). Underflow indicates
// an inverted range or a position computed out of order.
func (s TextSize) Sub(other TextSize) TextSize {
	if other > s {
		panic("text: TextSize subtraction underflow")
	}
	return s - other
}

// TextRange is a half-open byte range [Start, End) within a source file, with
// Start <= End. The zero value is the empty range at offset 0. TextRange is an
// immutable value type.
type TextRange struct {
	Start TextSize
	End   TextSize
}

// NewRange returns the range [start, end). It panics if end < start.
func NewRange(start, end TextSize) TextRange {
	if end < start {
		panic(fmt.Sprintf("text: invalid TextRange [%d, %d): end before start", start, end))
	}
	return TextRange{Start: start, End: end}
}

// RangeAt returns the range [offset, offset+length). It panics if
// offset+length overflows [TextSize].
func RangeAt(offset, length TextSize) TextRange {
	return TextRange{Start: offset, End: offset.Add(length)}
}

// EmptyRange returns the empty range [at, at).
func EmptyRange(at TextSize) TextRange {
	return TextRange{Start: at, End: at}
}

// Len returns the length of the range, End-Start.
func (r TextRange) Len() TextSize {
	return r.End - r.Start
}

// IsEmpty reports whether the range covers no bytes (Start == End).
func (r TextRange) IsEmpty() bool {
	return r.Start == r.End
}

// Contains reports whether offset lies within the half-open range, i.e.
// Start <= offset < End. An empty range contains no offset.
func (r TextRange) Contains(offset TextSize) bool {
	return r.Start <= offset && offset < r.End
}

// ContainsInclusive reports whether offset lies within the closed range, i.e.
// Start <= offset <= End. Unlike [TextRange.Contains], the End boundary counts,
// which is the useful test for a cursor position that may sit just past the
// last byte.
func (r TextRange) ContainsInclusive(offset TextSize) bool {
	return r.Start <= offset && offset <= r.End
}

// ContainsRange reports whether other is fully contained in r, i.e.
// r.Start <= other.Start and other.End <= r.End.
func (r TextRange) ContainsRange(other TextRange) bool {
	return r.Start <= other.Start && other.End <= r.End
}

// Intersect returns the overlap of r and other and whether they overlap. Ranges
// that merely touch at a boundary (e.g. [1,3) and [3,5)) do not overlap, so ok
// is false and the returned range is the zero value.
func (r TextRange) Intersect(other TextRange) (TextRange, bool) {
	start := r.Start
	if other.Start > start {
		start = other.Start
	}
	end := r.End
	if other.End < end {
		end = other.End
	}
	if start >= end {
		return TextRange{}, false
	}
	return TextRange{Start: start, End: end}, true
}

// Cover returns the smallest range containing both r and other (their bounding
// union), regardless of whether they overlap.
func (r TextRange) Cover(other TextRange) TextRange {
	start := r.Start
	if other.Start < start {
		start = other.Start
	}
	end := r.End
	if other.End > end {
		end = other.End
	}
	return TextRange{Start: start, End: end}
}

// String renders the range as "[Start, End)".
func (r TextRange) String() string {
	return fmt.Sprintf("[%d, %d)", r.Start, r.End)
}
