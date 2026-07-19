package contracts

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const bodyModel = `package API {
	import ScalarValues::*;
	import RESTProfile::*;
	item def Order {
		attribute id : String;
		attribute total : Real;
	}
	item def Receipt {
		attribute ref : String;
	}
	@REST { path = "/orders"; method = "POST"; successStatus = 201; }
	action placeOrder {
		in order : Order;
		out receipt : Receipt;
	}
	@REST { path = "/health"; method = "GET"; }
	action health;
}`

func buildBodyDoc(t *testing.T) *Document {
	t.Helper()
	m := engine.New().AddFile("api.sysml", bodyModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	return BuildDocument(m)
}

func TestRequestBodyFromInParam(t *testing.T) {
	doc := buildBodyDoc(t)
	op := doc.Paths["/orders"].Post
	if op.RequestBody == nil {
		t.Fatal("POST /orders has no requestBody")
	}
	if !op.RequestBody.Required {
		t.Error("requestBody should be required")
	}
	mt, ok := op.RequestBody.Content[jsonMediaType]
	if !ok {
		t.Fatalf("requestBody missing %s content", jsonMediaType)
	}
	if got := mt.Schema.Ref; got != "#/components/schemas/API.Order" {
		t.Errorf("requestBody schema $ref = %q, want API.Order", got)
	}
}

func TestResponseBodyFromOutParam(t *testing.T) {
	doc := buildBodyDoc(t)
	op := doc.Paths["/orders"].Post
	resp, ok := op.Responses["201"]
	if !ok {
		t.Fatal("no 201 response")
	}
	mt, ok := resp.Content[jsonMediaType]
	if !ok {
		t.Fatalf("201 response missing %s content", jsonMediaType)
	}
	if got := mt.Schema.Ref; got != "#/components/schemas/API.Receipt" {
		t.Errorf("response schema $ref = %q, want API.Receipt", got)
	}
}

func TestNoBodyWithoutParams(t *testing.T) {
	doc := buildBodyDoc(t)
	op := doc.Paths["/health"].Get
	if op.RequestBody != nil {
		t.Error("GET /health should have no requestBody")
	}
	if resp := op.Responses["200"]; resp != nil && resp.Content != nil {
		t.Error("GET /health success response should have no content")
	}
}

// TestBodiesInGeneratedCode verifies the emitted bodies survive oapi-codegen:
// the request object gains a Body field and the response is typed by the out
// param.
func TestBodiesInGeneratedCode(t *testing.T) {
	m := engine.New().AddFile("api.sysml", bodyModel).Build()
	src, err := GenerateServer(m, "api")
	if err != nil {
		t.Fatalf("GenerateServer: %v", err)
	}
	for _, want := range []string{
		"type PlaceOrderJSONRequestBody = APIOrder",
		"Body *PlaceOrderJSONRequestBody",
		"type PlaceOrder201JSONResponse APIReceipt",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing %q", want)
		}
	}
}

func TestBodyYAMLSerialization(t *testing.T) {
	doc := buildBodyDoc(t)
	yaml := doc.YAML()
	for _, want := range []string{"requestBody:", "application/json:", "$ref: '#/components/schemas/API.Order'"} {
		if !strings.Contains(yaml, want) {
			t.Errorf("openapi.yaml missing %q\n%s", want, yaml)
		}
	}
}
