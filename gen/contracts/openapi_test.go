package contracts

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/gaarutyunov/sysgo/engine"
)

const apiModel = `package API {
	import ScalarValues::*;
	import RESTProfile::*;
	item def Order {
		attribute id : String;
		attribute total : Real;
	}
	item def LineItem {
		attribute sku : String;
	}
	@REST { path = "/orders"; method = "POST"; successStatus = 201; }
	action placeOrder {
		in order : Order;
	}
	@REST { path = "/orders"; method = "GET"; }
	action listOrders;
}`

func buildDoc(t *testing.T) *Document {
	t.Helper()
	m := engine.New().AddFile("api.sysml", apiModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model has diagnostics: %v", d)
	}
	return BuildDocument(m)
}

func TestDocumentBasics(t *testing.T) {
	doc := buildDoc(t)
	if doc.OpenAPI != "3.1.0" {
		t.Errorf("openapi = %q, want 3.1.0", doc.OpenAPI)
	}
	if doc.Info.Title == "" || doc.Info.Version == "" {
		t.Errorf("info incomplete: %+v", doc.Info)
	}
}

func TestComponentsFromItemDefs(t *testing.T) {
	doc := buildDoc(t)
	// item defs with attributes → component schemas; the default RFC 9457
	// error schema is added because the operations use it; operations excluded.
	if got := doc.SchemaNames(); !reflect.DeepEqual(got, []string{"API.LineItem", "API.Order", "ProblemDetails"}) {
		t.Fatalf("schema names = %v, want [API.LineItem API.Order ProblemDetails]", got)
	}
	order := doc.Components.Schemas["API.Order"]
	if order.Type != "object" {
		t.Fatalf("Order schema type = %q, want object", order.Type)
	}
	if order.Properties["id"].Type != "string" || order.Properties["total"].Type != "number" {
		t.Errorf("Order properties wrong: %+v", order.Properties)
	}
}

func TestPathsFromRESTOperations(t *testing.T) {
	doc := buildDoc(t)
	item := doc.Paths["/orders"]
	if item == nil {
		t.Fatalf("no /orders path (%v)", doc.Paths)
	}
	if item.Post == nil || item.Post.OperationID != "placeOrder" {
		t.Errorf("POST /orders operationId wrong: %+v", item.Post)
	}
	if _, ok := item.Post.Responses["201"]; !ok {
		t.Errorf("POST /orders missing 201 response: %+v", item.Post.Responses)
	}
	if item.Get == nil || item.Get.OperationID != "listOrders" {
		t.Errorf("GET /orders operationId wrong: %+v", item.Get)
	}
	if _, ok := item.Get.Responses["200"]; !ok { // default status
		t.Errorf("GET /orders missing default 200 response: %+v", item.Get.Responses)
	}
}

func TestDefaultErrorResponse(t *testing.T) {
	doc := buildDoc(t)
	// RFC 9457 Problem Details schema is present.
	pd := doc.Components.Schemas["ProblemDetails"]
	if pd == nil || pd.Type != "object" {
		t.Fatalf("ProblemDetails schema = %+v, want object", pd)
	}
	for _, f := range []string{"type", "title", "status", "detail", "instance"} {
		if _, ok := pd.Properties[f]; !ok {
			t.Errorf("ProblemDetails missing field %q", f)
		}
	}
	// Each operation has a default error response referencing it via
	// application/problem+json.
	op := doc.Paths["/orders"].Post
	resp, ok := op.Responses["default"]
	if !ok {
		t.Fatalf("POST /orders has no default response: %+v", op.Responses)
	}
	mt, ok := resp.Content["application/problem+json"]
	if !ok {
		t.Fatalf("default response missing problem+json content: %+v", resp.Content)
	}
	if mt.Schema.Ref != "#/components/schemas/ProblemDetails" {
		t.Errorf("error schema ref = %q, want #/components/schemas/ProblemDetails", mt.Schema.Ref)
	}
}

func TestErrorModelOverride(t *testing.T) {
	src := `package API {
	import ScalarValues::*;
	import RESTProfile::*;
	item def MyError { attribute code : Integer; }
	@REST { path = "/x"; method = "GET"; }
	@ErrorModel { schemaRef = "MyError"; }
	action getX;
}`
	m := engine.New().AddFile("api.sysml", src).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("diagnostics: %v", d)
	}
	doc := BuildDocument(m)

	resp := doc.Paths["/x"].Get.Responses["default"]
	ref := resp.Content["application/problem+json"].Schema.Ref
	if ref != "#/components/schemas/MyError" {
		t.Errorf("override error ref = %q, want #/components/schemas/MyError", ref)
	}
	// The default Problem Details schema is NOT added when every operation
	// overrides its error model.
	if _, ok := doc.Components.Schemas["ProblemDetails"]; ok {
		t.Error("ProblemDetails should not be present when overridden")
	}
}

func TestYAMLDeterministicAndRoundTrips(t *testing.T) {
	doc := buildDoc(t)
	a := doc.YAML()
	b := doc.YAML()
	if a != b {
		t.Fatal("YAML output not deterministic")
	}
	for _, want := range []string{
		"openapi:", "3.1.0", "paths:", "/orders:", "post:", "operationId: placeOrder",
		"components:", "schemas:", "API.Order:", "application/problem+json:", "ProblemDetails:",
	} {
		if !strings.Contains(a, want) {
			t.Errorf("YAML missing %q:\n%s", want, a)
		}
	}
	var back Document
	if err := yaml.Unmarshal([]byte(a), &back); err != nil {
		t.Fatalf("re-parse failed: %v", err)
	}
	if back.Paths["/orders"].Post.OperationID != "placeOrder" {
		t.Errorf("round-trip lost the POST operation")
	}
	if _, ok := back.Components.Schemas["API.Order"]; !ok {
		t.Errorf("round-trip lost the Order schema")
	}
	if back.Paths["/orders"].Post.Responses["default"].Content["application/problem+json"] == nil {
		t.Errorf("round-trip lost the error response content")
	}
}

func TestEmptyModelDocument(t *testing.T) {
	m := engine.New().AddFile("m.sysml", "package M {}").Build()
	doc := BuildDocument(m)
	if doc.OpenAPI != "3.1.0" || doc.Paths != nil || doc.Components != nil {
		t.Errorf("empty model doc = %+v, want just openapi+info", doc)
	}
}
