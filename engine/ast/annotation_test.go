package ast

import (
	"testing"

	"github.com/gaarutyunov/sysgo/engine/parser"
)

func firstAnnotation(t *testing.T, src string) Annotation {
	t.Helper()
	sf := New(parser.Parse(src))
	for _, m := range sf.Members() {
		d, ok := m.(Declaration)
		if !ok {
			continue
		}
		anns := d.Annotations()
		if len(anns) > 0 {
			return anns[0]
		}
	}
	t.Fatalf("no annotation found in %q", src)
	return Annotation{}
}

func TestAnnotationAssignments(t *testing.T) {
	a := firstAnnotation(t, "@REST { path = \"/orders\"; method = POST; successStatus = 201; } part def Order;")

	name, ok := a.Name()
	if !ok || name.String() != "REST" {
		t.Fatalf("annotation name = %q (%v), want REST", name.String(), ok)
	}
	asn := a.Assignments()
	if len(asn) != 3 {
		t.Fatalf("assignments = %d, want 3 (%+v)", len(asn), asn)
	}
	want := map[string]string{
		"path":          `"/orders"`,
		"method":        "POST",
		"successStatus": "201",
	}
	for _, a := range asn {
		if want[a.Name] != a.Value {
			t.Errorf("assignment %q = %q, want %q", a.Name, a.Value, want[a.Name])
		}
	}
	// Order is preserved.
	if asn[0].Name != "path" || asn[1].Name != "method" || asn[2].Name != "successStatus" {
		t.Errorf("assignment order = %v", []string{asn[0].Name, asn[1].Name, asn[2].Name})
	}
}

func TestAnnotationQualifiedValue(t *testing.T) {
	a := firstAnnotation(t, "@REST { method = HttpMethod::POST; } part def O;")
	asn := a.Assignments()
	if len(asn) != 1 || asn[0].Name != "method" || asn[0].Value != "HttpMethod::POST" {
		t.Errorf("assignment = %+v, want method=HttpMethod::POST", asn)
	}
}

func TestBodylessAnnotationNoAssignments(t *testing.T) {
	a := firstAnnotation(t, "@Approved part def X;")
	if _, ok := a.Body(); ok {
		t.Error("bodyless annotation should have no body")
	}
	if asn := a.Assignments(); asn != nil {
		t.Errorf("assignments = %v, want nil", asn)
	}
}
