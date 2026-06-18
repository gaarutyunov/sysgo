package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseValid(t *testing.T) {
	cfg, err := Parse([]byte(`
module: github.com/acme/orders
source:
  file: ./model.json
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Module != "github.com/acme/orders" {
		t.Fatalf("module = %q", cfg.Module)
	}
	if cfg.Generate.Adapters != AdaptersScaffold {
		t.Fatalf("default adapters = %q", cfg.Generate.Adapters)
	}
	if cfg.Layout["domain"].Dir != "internal/{context}/domain" {
		t.Fatalf("default layout missing")
	}
}

func TestParsePartialLayoutOverride(t *testing.T) {
	// A region present but with empty fields must still be fully defaulted, and
	// regions omitted entirely must be added.
	cfg, err := Parse([]byte(`
module: github.com/acme/orders
source:
  file: ./model.json
layout:
  domain: {}
  app: { package: app }
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := cfg.Layout["domain"]; got.Dir != "internal/{context}/domain" || got.Package != "domain" {
		t.Fatalf("empty domain region not defaulted: %+v", got)
	}
	if got := cfg.Layout["app"]; got.Package != "app" || got.Dir != "internal/{context}/app/usecase" {
		t.Fatalf("partial app region not backfilled: %+v", got)
	}
	if _, ok := cfg.Layout["cmd"]; !ok {
		t.Fatal("omitted cmd region not added")
	}
}

func TestParseMissingModule(t *testing.T) {
	if _, err := Parse([]byte(`source: { file: x }`)); err == nil {
		t.Fatal("expected schema/semantic error for missing module")
	}
}

func TestParseRejectsUnknownKey(t *testing.T) {
	_, err := Parse([]byte(`
module: m
source: { file: x }
bogus-key: true
`))
	if err == nil {
		t.Fatal("expected schema error for additional property")
	}
}

func TestParseMutuallyExclusiveSource(t *testing.T) {
	_, err := Parse([]byte(`
module: m
source:
  file: x
  api: { base-url: y, project: z }
`))
	if err == nil {
		t.Fatal("expected mutually-exclusive source error")
	}
}

func TestParseInvalidAdapters(t *testing.T) {
	_, err := Parse([]byte(`
module: m
source: { file: x }
generate: { adapters: sometimes }
`))
	if err == nil {
		t.Fatal("expected enum violation for adapters")
	}
}

// TestEmbeddedSchemaMatchesPublished guards against drift between the embedded
// schema and the published schema/sysgo.schema.json.
func TestEmbeddedSchemaMatchesPublished(t *testing.T) {
	// Locate repo root from this test file's directory.
	wd, _ := os.Getwd()
	published := filepath.Join(wd, "..", "..", "schema", "sysgo.schema.json")
	data, err := os.ReadFile(published)
	if err != nil {
		t.Skipf("published schema not found: %v", err)
	}
	if string(data) != string(SchemaJSON()) {
		t.Fatal("schema/sysgo.schema.json differs from embedded schema.json")
	}
}
