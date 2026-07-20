package contracts

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/gaarutyunov/sysgo/engine"
)

// TestToOpenAPI3 verifies the document is built directly as a valid kin-openapi
// *openapi3.T (schemas, paths, request body) with its internal refs resolvable —
// no YAML round-trip.
func TestToOpenAPI3(t *testing.T) {
	m := engine.New().AddFile("api.sysml", apiModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model has diagnostics: %v", d)
	}
	spec := BuildDocument(m).ToOpenAPI3()

	if err := openapi3.NewLoader().ResolveRefsIn(spec, nil); err != nil {
		t.Fatalf("resolve refs: %v", err)
	}
	if err := spec.Validate(context.Background()); err != nil {
		t.Fatalf("validate: %v", err)
	}

	if spec.OpenAPI != "3.1.0" {
		t.Errorf("openapi = %q, want 3.1.0", spec.OpenAPI)
	}
	if spec.Info == nil || spec.Info.Title == "" {
		t.Error("info not set")
	}
	if spec.Components == nil || len(spec.Components.Schemas) == 0 {
		t.Fatal("no component schemas")
	}
	pi := spec.Paths.Find("/orders")
	if pi == nil || pi.Post == nil {
		t.Fatal("POST /orders missing")
	}
	if pi.Post.RequestBody == nil || pi.Post.RequestBody.Value == nil {
		t.Error("POST /orders missing request body")
	}
	if pi.Post.Responses == nil {
		t.Error("POST /orders missing responses")
	}
}
