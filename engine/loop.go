package engine

import (
	"github.com/gaarutyunov/sysgo/engine/hir"
	"github.com/gaarutyunov/sysgo/engine/text"
)

// Loop is a resolved `loop <count> times <activity>` repetition within an
// action: repeat the target activity Count times.
type Loop struct {
	Count      string // repeat count as written (int literal or param name)
	TargetName string // the repeated activity reference as written

	target *hir.Symbol
	model  *Model
	rng    text.TextRange
}

// Range returns the loop's source range, for ordering it against other steps.
func (l Loop) Range() text.TextRange { return l.rng }

// Target returns the resolved repeated activity, if it resolved.
func (l Loop) Target() (Element, bool) {
	if l.target == nil {
		return Element{}, false
	}
	return Element{sym: l.target, model: l.model}, true
}

// Loops returns the element's `loop` repetitions in declaration order (empty for
// a non-action or an action with no loops).
func (e Element) Loops() []Loop {
	out := make([]Loop, len(e.sym.Loops))
	for i, lp := range e.sym.Loops {
		out[i] = Loop{
			Count:      lp.Count,
			TargetName: lp.TargetName,
			target:     lp.Target,
			model:      e.model,
			rng:        lp.Range,
		}
	}
	return out
}
