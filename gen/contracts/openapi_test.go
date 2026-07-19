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
	// item defs with attributes → component schemas; actions/operations excluded.
	if got := doc.SchemaNames(); !reflect.DeepEqual(got, []string{"API.LineItem", "API.Order"}) {
		t.Fatalf("schema names = %v, want [API.LineItem API.Order]", got)
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

func TestYAMLDeterministicAndRoundTrips(t *testing.T) {
	doc := buildDoc(t)
	a := doc.YAML()
	b := doc.YAML()
	if a != b {
		t.Fatal("YAML output not deterministic")
	}
	// Sanity substrings.
	for _, want := range []string{"openapi:", "3.1.0", "paths:", "/orders:", "post:", "operationId: placeOrder", "components:", "schemas:", "API.Order:"} {
		if !strings.Contains(a, want) {
			t.Errorf("YAML missing %q:\n%s", want, a)
		}
	}
	// Round-trips back into an equivalent document.
	var back Document
	if err := yaml.Unmarshal([]byte(a), &back); err != nil {
		t.Fatalf("re-parse failed: %v", err)
	}
	if back.OpenAPI != "3.1.0" {
		t.Errorf("round-trip openapi = %q", back.OpenAPI)
	}
	if back.Paths["/orders"].Post.OperationID != "placeOrder" {
		t.Errorf("round-trip lost the POST operation")
	}
	if _, ok := back.Components.Schemas["API.Order"]; !ok {
		t.Errorf("round-trip lost the Order schema")
	}
}

func TestEmptyModelDocument(t *testing.T) {
	m := engine.New().AddFile("m.sysml", "package M {}").Build()
	doc := BuildDocument(m)
	if doc.OpenAPI != "3.1.0" || doc.Paths != nil || doc.Components != nil {
		t.Errorf("empty model doc = %+v, want just openapi+info", doc)
	}
}
