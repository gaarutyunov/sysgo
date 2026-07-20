package openapi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sysmlSrc = `package API {
	import ScalarValues::*;
	import RESTProfile::*;
	item def Order {
		attribute id : String;
		attribute total : Real;
	}
	@REST { path = "/orders"; method = "POST"; successStatus = 201; }
	action placeOrder {
		in order : Order;
	}
	@REST { path = "/orders"; method = "GET"; }
	action listOrders;
}`

func writeSysML(t *testing.T, src string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "api.sysml")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEmit(t *testing.T) {
	files, err := New(writeSysML(t, sysmlSrc)).Emit(context.Background())
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	f := files[0]
	if f.Path != "openapi.yaml" {
		t.Errorf("path = %q, want openapi.yaml", f.Path)
	}
	if !f.Generated {
		t.Errorf("Generated = false, want true")
	}
	s := string(f.Content)
	for _, want := range []string{"openapi:", "3.1.0", "API.Order", "/orders"} {
		if !strings.Contains(s, want) {
			t.Errorf("openapi.yaml missing %q\n---\n%s", want, s)
		}
	}
}

func TestEmitErrors(t *testing.T) {
	if _, err := (&Emitter{}).Emit(context.Background()); err == nil {
		t.Error("want error for empty SysMLPath")
	}
	missing := filepath.Join(t.TempDir(), "nope.sysml")
	if _, err := New(missing).Emit(context.Background()); err == nil {
		t.Error("want error for missing source file")
	}
	// A source with unresolved references must surface as a diagnostic error.
	bad := writeSysML(t, `package Bad { item def X { attribute y : Nonexistent; } }`)
	if _, err := New(bad).Emit(context.Background()); err == nil {
		t.Error("want error for model with diagnostics")
	}
}
