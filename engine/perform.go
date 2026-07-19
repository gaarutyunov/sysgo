package engine

import "github.com/gaarutyunov/sysgo/engine/hir"

// Perform is a resolved `perform` action reference within an action element.
type Perform struct {
	// Name is the local step name (`perform name : Target`), or "" for a direct
	// `perform Target` reference.
	Name string
	// TargetName is the referenced target as written in source.
	TargetName string

	target *hir.Symbol
	model  *Model
}

// IsResolved reports whether the performed target resolved to an element.
func (p Perform) IsResolved() bool { return p.target != nil }

// Target returns the resolved performed action element, if any.
func (p Perform) Target() (Element, bool) {
	if p.target == nil {
		return Element{}, false
	}
	return Element{sym: p.target, model: p.model}, true
}

// Performs returns the element's `perform` steps in declaration order (empty for
// a non-action or an action with no perform statements).
func (e Element) Performs() []Perform {
	out := make([]Perform, len(e.sym.Performs))
	for i, ps := range e.sym.Performs {
		out[i] = Perform{Name: ps.Name, TargetName: ps.Name0, target: ps.Target, model: e.model}
	}
	return out
}
