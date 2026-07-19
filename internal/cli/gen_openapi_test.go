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
