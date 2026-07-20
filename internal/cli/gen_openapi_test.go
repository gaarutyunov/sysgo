package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const genOpenAPIModel = `package Catalog {
	import ScalarValues::*;
	item def Product {
		attribute id : String;
		attribute name : String;
		attribute price : Real;
	}
	item def Category {
		attribute id : String;
		attribute label : String;
	}
}`

func TestGenOpenAPI(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "catalog.sysml")
	if err := os.WriteFile(model, []byte(genOpenAPIModel), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "openapi.yaml")
	if _, err := runGen(t, "gen", "openapi", model, "--out", out); err != nil {
		t.Fatalf("gen openapi: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read generated document: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"openapi: 3.1",
		"Catalog.Product",
		"Catalog.Category",
		"type: object",
		"price:",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("openapi.yaml missing %q:\n%s", want, got)
		}
	}
}

// TestGenOpenAPIDeterministic checks two runs produce identical output — the
// property the drift check relies on.
func TestGenOpenAPIDeterministic(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "catalog.sysml")
	if err := os.WriteFile(model, []byte(genOpenAPIModel), 0o644); err != nil {
		t.Fatal(err)
	}
	read := func() string {
		out := filepath.Join(t.TempDir(), "openapi.yaml")
		if _, err := runGen(t, "gen", "openapi", model, "--out", out); err != nil {
			t.Fatalf("gen openapi: %v", err)
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	if a, b := read(), read(); a != b {
		t.Errorf("non-deterministic output:\n--- run 1 ---\n%s\n--- run 2 ---\n%s", a, b)
	}
}

// TestGenOpenAPIDiagnostics fails (non-nil error) when the model has
// diagnostics, so CI generation surfaces model errors.
func TestGenOpenAPIDiagnostics(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "bad.sysml")
	// Attribute typed by an undefined type -> unresolved diagnostic.
	if err := os.WriteFile(model, []byte(`package App {
	item def Thing { attribute x : Nonexistent; }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "openapi.yaml")
	if _, err := runGen(t, "gen", "openapi", model, "--out", out); err == nil {
		t.Error("expected an error for a model with diagnostics, got nil")
	}
}

const genOpenAPIRESTModel = `package Catalog {
	import ScalarValues::*;
	import RESTProfile::*;
	item def Product {
		attribute id : String;
		attribute name : String;
	}
	@REST { path = "/products"; method = "POST"; successStatus = 201; }
	action def CreateProduct {
		in product : Product;
		out created : Product;
	}
}`

// TestGenOpenAPIServer generates the gin server Go code directly from the model
// (no openapi.yaml intermediate) and checks the server surface is present.
func TestGenOpenAPIServer(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "catalog.sysml")
	if err := os.WriteFile(model, []byte(genOpenAPIRESTModel), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "server.gen.go")
	if _, err := runGen(t, "gen", "openapi", model, "--server", "--out", out, "--package", "api"); err != nil {
		t.Fatalf("gen openapi --server: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read generated server: %v", err)
	}
	got := string(data)
	for _, want := range []string{"package api", "ServerInterface", "RegisterHandlers"} {
		if !strings.Contains(got, want) {
			t.Errorf("generated server missing %q", want)
		}
	}
	// The server mode must not leave a stale openapi.yaml lying around.
	if _, err := os.Stat(filepath.Join(dir, "openapi.yaml")); !os.IsNotExist(err) {
		t.Errorf("--server should not write openapi.yaml (stat err = %v)", err)
	}
}

// TestGenOpenAPIModels generates only the model types.
func TestGenOpenAPIModels(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "catalog.sysml")
	if err := os.WriteFile(model, []byte(genOpenAPIRESTModel), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "models.gen.go")
	if _, err := runGen(t, "gen", "openapi", model, "--models", "--out", out, "--package", "api"); err != nil {
		t.Fatalf("gen openapi --models: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "package api") {
		t.Errorf("generated models missing package declaration:\n%s", got)
	}
	if strings.Contains(got, "ServerInterface") {
		t.Errorf("--models should not emit the server interface")
	}
}
