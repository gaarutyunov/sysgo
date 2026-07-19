package engine

import "github.com/gaarutyunov/sysgo/engine/hir"

// Succession is a resolved `first A then B` control edge within an action.
type Succession struct {
	SourceName string // source reference as written, or "" for a bare `then`
	TargetName string // target reference as written
	Guard      string // `if` condition text, or "" for an unguarded edge

	source *hir.Symbol
	target *hir.Symbol
	model  *Model
}

// Source returns the resolved source element, if the edge had an explicit
// `first` reference that resolved.
func (s Succession) Source() (Element, bool) {
	if s.source == nil {
		return Element{}, false
	}
	return Element{sym: s.source, model: s.model}, true
}

// Target returns the resolved target element, if it resolved.
func (s Succession) Target() (Element, bool) {
	if s.target == nil {
		return Element{}, false
	}
	return Element{sym: s.target, model: s.model}, true
}

// Successions returns the element's `first … then` control edges in declaration
// order (empty for a non-action or an action with no successions).
func (e Element) Successions() []Succession {
	out := make([]Succession, len(e.sym.Successions))
	for i, se := range e.sym.Successions {
		out[i] = Succession{
			SourceName: se.SourceName,
			TargetName: se.TargetName,
			Guard:      se.Guard,
			source:     se.Source,
			target:     se.Target,
			model:      e.model,
		}
	}
	return out
}
