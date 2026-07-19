package ast

import (
	"strings"

	"github.com/gaarutyunov/sysgo/engine/cst"
	"github.com/gaarutyunov/sysgo/engine/parser"
)

// Transition is a `transition … first … then …` state transition inside a
// `state def` body. Every clause is optional except, for a well-formed
// transition, the `then` target: an optional name, a `first` source state, an
// `accept` trigger, an `if` guard, a `do` effect, and a `then` target.
type Transition struct{ node cst.Node }

func (t Transition) Syntax() cst.Node { return t.node }

// transitionParts is the resolved role of each of a transition's qualified
// names and its guard, mapped by the keyword that precedes each in source.
type transitionParts struct {
	name    QualifiedName
	source  QualifiedName
	trigger QualifiedName
	effect  QualifiedName
	target  QualifiedName

	hasName, hasSource, hasTrigger, hasEffect, hasTarget bool

	guard          string // `if` condition text, if any
	hasGuard       bool
	triggerLiteral string // literal `accept after N` value, if the trigger is a literal
	hasTriggerLit  bool
}

// parts walks the transition's significant children once, assigning each
// QualifiedName (and the guard/trigger Expr) to its role by tracking the most
// recent clause keyword.
func (t Transition) parts() transitionParts {
	var p transitionParts
	kw := "" // most recent clause keyword: "", first, accept, if, do, then
	for _, c := range t.node.Children() {
		switch n := c.(type) {
		case cst.Token:
			if kindIsTrivia(n.Kind()) {
				continue
			}
			if parser.SyntaxKind(n.Kind()) == parser.KindIdent {
				switch n.Text() {
				case "transition", "after", "at":
					// leading keyword / timer modifier — not a role marker
				case "first", "accept", "if", "do", "then":
					kw = n.Text()
				}
			}
		case cst.Node:
			switch kindOf(n) {
			case parser.KindQualifiedName:
				qn := QualifiedName{n}
				switch kw {
				case "":
					p.name, p.hasName = qn, true
				case "first":
					p.source, p.hasSource = qn, true
				case "accept":
					p.trigger, p.hasTrigger = qn, true
				case "do":
					p.effect, p.hasEffect = qn, true
				case "then":
					p.target, p.hasTarget = qn, true
				}
			case parser.KindExpr:
				switch kw {
				case "accept":
					p.triggerLiteral, p.hasTriggerLit = strings.TrimSpace(Expr{n}.Text()), true
				case "if":
					p.guard, p.hasGuard = strings.TrimSpace(Expr{n}.Text()), true
				}
			}
		}
	}
	return p
}

// Name returns the transition's declared name, if it has one.
func (t Transition) Name() (QualifiedName, bool) {
	p := t.parts()
	return p.name, p.hasName
}

// Source returns the transition's source state (`first S`), if present.
func (t Transition) Source() (QualifiedName, bool) {
	p := t.parts()
	return p.source, p.hasSource
}

// Target returns the transition's target state (`then S`), if present.
func (t Transition) Target() (QualifiedName, bool) {
	p := t.parts()
	return p.target, p.hasTarget
}

// Trigger returns the transition's `accept` trigger reference, if present. The
// boolean is false for a bare `accept after N` literal timer (see TriggerText).
func (t Transition) Trigger() (QualifiedName, bool) {
	p := t.parts()
	return p.trigger, p.hasTrigger
}

// TriggerText returns the trigger as written — a signal name, or the literal
// value of an `accept after N` timer — and whether any trigger is present.
func (t Transition) TriggerText() (string, bool) {
	p := t.parts()
	switch {
	case p.hasTrigger:
		return p.trigger.String(), true
	case p.hasTriggerLit:
		return p.triggerLiteral, true
	default:
		return "", false
	}
}

// Guard returns the transition's `if` condition text, if present.
func (t Transition) Guard() (string, bool) {
	p := t.parts()
	return p.guard, p.hasGuard
}

// Effect returns the transition's `do` effect action reference, if present.
func (t Transition) Effect() (QualifiedName, bool) {
	p := t.parts()
	return p.effect, p.hasEffect
}

// Transitions returns the declaration body's state transitions in order.
func (d Declaration) Transitions() []Transition {
	b, ok := d.Body()
	if !ok {
		return nil
	}
	var out []Transition
	for _, c := range childNodes(b.node) {
		if kindOf(c) == parser.KindTransition {
			out = append(out, Transition{c})
		}
	}
	return out
}
