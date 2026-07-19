package engine

import (
	"github.com/gaarutyunov/sysgo/engine/hir"
	"github.com/gaarutyunov/sysgo/engine/text"
)

// Accept is a resolved `accept` statement within an action: a signal acceptance
// or a durable timer (`accept after …` / `accept at …`).
type Accept struct {
	Mode string // "signal", "after", or "at"
	Ref  string // the accepted reference/value as written

	// Range is the accept statement's source range, for ordering timer steps
	// against activity steps in a workflow body.
	Range text.TextRange

	target *hir.Symbol
	model  *Model
}

// IsTimer reports whether the accept is a durable timer (after / at).
func (a Accept) IsTimer() bool { return a.Mode == "after" || a.Mode == "at" }

// Target returns the resolved signal/duration reference element, if any (absent
// for a literal timer value).
func (a Accept) Target() (Element, bool) {
	if a.target == nil {
		return Element{}, false
	}
	return Element{sym: a.target, model: a.model}, true
}

// Accepts returns the element's `accept` statements in declaration order.
func (e Element) Accepts() []Accept {
	out := make([]Accept, len(e.sym.Accepts))
	for i, a := range e.sym.Accepts {
		out[i] = Accept{Mode: a.Mode, Ref: a.Ref, Range: a.Range, target: a.Target, model: e.model}
	}
	return out
}
