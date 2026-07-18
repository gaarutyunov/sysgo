package text

import "sync"

// Symbol is a stable, comparable handle for a string interned by an [Interner].
// Comparing two Symbols is an integer comparison; two Symbols are equal iff the
// strings they were interned from are equal. A Symbol is only meaningful
// relative to the [Interner] that produced it.
//
// The zero value is [EmptySymbol], the interned empty string, for every
// Interner created with [NewInterner].
type Symbol uint32

// EmptySymbol is the Symbol of the empty string. It is pre-interned by
// [NewInterner], so the zero value of [Symbol] always resolves to "".
const EmptySymbol Symbol = 0

// Interner assigns a stable [Symbol] to each distinct string and resolves a
// Symbol back to its string. It is safe for concurrent use by multiple
// goroutines.
type Interner struct {
	mu   sync.RWMutex
	ids  map[string]Symbol
	strs []string
}

// NewInterner returns an empty Interner. The empty string is pre-interned as
// [EmptySymbol] so that the zero [Symbol] value is always valid.
func NewInterner() *Interner {
	return &Interner{
		ids:  map[string]Symbol{"": EmptySymbol},
		strs: []string{""},
	}
}

// Intern returns the [Symbol] for s, assigning a new one the first time s is
// seen. Interning the same string again returns the same Symbol.
func (in *Interner) Intern(s string) Symbol {
	// Fast path: already interned.
	in.mu.RLock()
	sym, ok := in.ids[s]
	in.mu.RUnlock()
	if ok {
		return sym
	}

	in.mu.Lock()
	defer in.mu.Unlock()
	// Re-check: another goroutine may have interned s between the two locks.
	if sym, ok := in.ids[s]; ok {
		return sym
	}
	sym = Symbol(len(in.strs))
	in.strs = append(in.strs, s)
	in.ids[s] = sym
	return sym
}

// Lookup returns the string that sym was interned from. It panics if sym did
// not come from this Interner (i.e. is out of range).
func (in *Interner) Lookup(sym Symbol) string {
	in.mu.RLock()
	defer in.mu.RUnlock()
	if int(sym) >= len(in.strs) {
		panic("text: Symbol out of range for this Interner")
	}
	return in.strs[sym]
}

// Len reports the number of distinct strings interned, including the
// pre-interned empty string.
func (in *Interner) Len() int {
	in.mu.RLock()
	defer in.mu.RUnlock()
	return len(in.strs)
}
