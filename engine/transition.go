package engine

import "github.com/gaarutyunov/sysgo/engine/hir"

// Transition is a resolved `transition … first … then …` state transition
// within a state def: an optional name, source and target states, an optional
// trigger signal and effect action, and the guard condition as written.
// gen/temporal consumes these to build selector-loop state machines.
type Transition struct {
	Name       string // transition name, or "" if anonymous
	SourceName string // source state reference as written
	TargetName string // target state reference as written
	Trigger    string // trigger reference/value as written, or ""
	Guard      string // `if` condition text, or ""
	EffectName string // effect action reference as written, or ""

	source *hir.Symbol
	target *hir.Symbol
	effect *hir.Symbol
	tsig   *hir.Symbol
	model  *Model
}

// Source returns the resolved source state, if the transition named one that
// resolved.
func (t Transition) Source() (Element, bool) {
	if t.source == nil {
		return Element{}, false
	}
	return Element{sym: t.source, model: t.model}, true
}

// Target returns the resolved target state, if it resolved.
func (t Transition) Target() (Element, bool) {
	if t.target == nil {
		return Element{}, false
	}
	return Element{sym: t.target, model: t.model}, true
}

// TriggerElement returns the resolved trigger signal, if the transition had an
// `accept` trigger that resolved (absent for a literal timer value).
func (t Transition) TriggerElement() (Element, bool) {
	if t.tsig == nil {
		return Element{}, false
	}
	return Element{sym: t.tsig, model: t.model}, true
}

// Effect returns the resolved effect action, if the transition had a `do`
// effect that resolved.
func (t Transition) Effect() (Element, bool) {
	if t.effect == nil {
		return Element{}, false
	}
	return Element{sym: t.effect, model: t.model}, true
}

// Transitions returns the element's state transitions in declaration order
// (empty for a non-state or a state def with no transitions).
func (e Element) Transitions() []Transition {
	out := make([]Transition, len(e.sym.Transitions))
	for i, tr := range e.sym.Transitions {
		out[i] = Transition{
			Name:       tr.Name,
			SourceName: tr.SourceName,
			TargetName: tr.TargetName,
			Trigger:    tr.TriggerRef,
			Guard:      tr.Guard,
			EffectName: tr.EffectName,
			source:     tr.Source,
			target:     tr.Target,
			effect:     tr.Effect,
			tsig:       tr.Trigger,
			model:      e.model,
		}
	}
	return out
}

// States returns the element's directly declared `state` members in order —
// the vertices a state machine transitions between.
func (e Element) States() []Element {
	var out []Element
	for _, c := range e.Children() {
		if c.DeclarationKeyword() == "state" {
			out = append(out, c)
		}
	}
	return out
}
