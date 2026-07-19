package project

import (
	"testing"

	"github.com/gaarutyunov/sysgo/engine/hir"
)

func TestProfileUnitsResolve(t *testing.T) {
	units := ProfileUnits()
	if len(units) < 1 {
		t.Fatal("no profile units embedded")
	}
	// The REST profile resolves cleanly alongside the stdlib.
	all := append(StdlibUnits(), units...)
	r := hir.AnalyzeUnits(all)
	if len(r.Diagnostics) != 0 {
		t.Fatalf("profile diagnostics: %v", r.Diagnostics)
	}
	for _, qn := range [][]string{{"RESTProfile", "REST"}, {"RESTProfile", "Api"}, {"RESTProfile", "ErrorModel"}} {
		if r.Model.Resolve(r.Model.Root, qn) == nil {
			t.Errorf("%v not resolvable", qn)
		}
	}
}

func TestWorkspaceLoadsProfile(t *testing.T) {
	w := New()
	w.AddFile("m.sysml", "package M {\n\timport RESTProfile::*;\n\tattribute m : HttpMethod;\n}")
	r := w.Analyze()
	if len(r.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %v", r.Diagnostics)
	}
	// A user model resolves profile names through the auto-loaded profile.
	if r.Model.Resolve(r.Model.Root, []string{"RESTProfile", "REST"}) == nil {
		t.Error("RESTProfile::REST not resolvable in workspace")
	}
}
