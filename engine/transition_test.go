package engine

import "testing"

const transitionModel = `package App {
	import ScalarValues::*;
	item def GoSignal;
	action def OpenGate;
	state def TrafficLight {
		state Red;
		state Green;
		transition RedToGreen
			first Red
			accept GoSignal
			if clear
			do OpenGate
			then Green;
		transition first Green then Red;
	}
}`

func TestTransitionsResolved(t *testing.T) {
	m := New().AddFile("app.sysml", transitionModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	tl, ok := m.Lookup("App::TrafficLight")
	if !ok {
		t.Fatal("TrafficLight not found")
	}

	// States exposed.
	states := tl.States()
	if len(states) != 2 {
		t.Fatalf("states = %d, want 2", len(states))
	}
	if states[0].Name() != "Red" || states[1].Name() != "Green" {
		t.Errorf("states = %q, %q, want Red, Green", states[0].Name(), states[1].Name())
	}

	// Transitions exposed.
	trs := tl.Transitions()
	if len(trs) != 2 {
		t.Fatalf("transitions = %d, want 2", len(trs))
	}

	full := trs[0]
	if full.Name != "RedToGreen" {
		t.Errorf("name = %q, want RedToGreen", full.Name)
	}
	if full.SourceName != "Red" || full.TargetName != "Green" {
		t.Errorf("edge = %q → %q, want Red → Green", full.SourceName, full.TargetName)
	}
	if full.Trigger != "GoSignal" {
		t.Errorf("trigger = %q, want GoSignal", full.Trigger)
	}
	if full.Guard != "clear" {
		t.Errorf("guard = %q, want clear", full.Guard)
	}
	if full.EffectName != "OpenGate" {
		t.Errorf("effect = %q, want OpenGate", full.EffectName)
	}
	if src, ok := full.Source(); !ok || src.QualifiedName() != "App::TrafficLight::Red" {
		t.Errorf("source = %+v (%v), want TrafficLight::Red", src, ok)
	}
	if tgt, ok := full.Target(); !ok || tgt.QualifiedName() != "App::TrafficLight::Green" {
		t.Errorf("target = %+v (%v), want TrafficLight::Green", tgt, ok)
	}
	if sig, ok := full.TriggerElement(); !ok || sig.QualifiedName() != "App::GoSignal" {
		t.Errorf("trigger element = %+v (%v), want App::GoSignal", sig, ok)
	}
	if eff, ok := full.Effect(); !ok || eff.QualifiedName() != "App::OpenGate" {
		t.Errorf("effect element = %+v (%v), want App::OpenGate", eff, ok)
	}

	// Minimal transition: no name, no trigger/guard/effect.
	min := trs[1]
	if min.Name != "" {
		t.Errorf("min name = %q, want empty", min.Name)
	}
	if min.SourceName != "Green" || min.TargetName != "Red" {
		t.Errorf("min edge = %q → %q, want Green → Red", min.SourceName, min.TargetName)
	}
	if min.Trigger != "" || min.Guard != "" || min.EffectName != "" {
		t.Errorf("min should have no trigger/guard/effect, got %+v", min)
	}
	if _, ok := min.TriggerElement(); ok {
		t.Error("min transition should have no trigger element")
	}
}

func TestNoTransitions(t *testing.T) {
	m := New().AddFile("app.sysml", transitionModel).Build()
	og, _ := m.Lookup("App::OpenGate")
	if len(og.Transitions()) != 0 {
		t.Error("OpenGate should have no transitions")
	}
	if len(og.States()) != 0 {
		t.Error("OpenGate should have no states")
	}
}

func TestTransitionTimerTrigger(t *testing.T) {
	src := `package App {
	state def Timeout {
		state Waiting;
		state Done;
		transition first Waiting accept after 30 then Done;
	}
}`
	m := New().AddFile("app.sysml", src).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	to, _ := m.Lookup("App::Timeout")
	trs := to.Transitions()
	if len(trs) != 1 {
		t.Fatalf("transitions = %d, want 1", len(trs))
	}
	tr := trs[0]
	if tr.SourceName != "Waiting" || tr.TargetName != "Done" {
		t.Errorf("edge = %q → %q, want Waiting → Done", tr.SourceName, tr.TargetName)
	}
	// A literal `accept after 30` trigger has a written value but no symbol.
	if tr.Trigger != "30" {
		t.Errorf("trigger = %q, want 30", tr.Trigger)
	}
	if _, ok := tr.TriggerElement(); ok {
		t.Error("literal timer trigger should have no resolved element")
	}
}
