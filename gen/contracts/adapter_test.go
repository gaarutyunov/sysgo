package contracts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

// adapterModel exercises all four request/response body combinations and the
// operation → domain-action `perform` binding.
const adapterModel = `package API {
	import ScalarValues::*;
	import RESTProfile::*;
	item def Order { attribute id : String; }
	item def Receipt { attribute ref : String; }
	action def PlaceOrderUC { in order : Order; out receipt : Receipt; }
	action def CancelUC { in order : Order; }
	action def StatusUC { out receipt : Receipt; }
	action def PingUC;
	@REST { path = "/orders"; method = "POST"; successStatus = 201; }
	action placeOrder { in order : Order; out receipt : Receipt; perform PlaceOrderUC; }
	@REST { path = "/cancel"; method = "POST"; }
	action cancel { in order : Order; perform CancelUC; }
	@REST { path = "/status"; method = "GET"; }
	action status { out receipt : Receipt; perform StatusUC; }
	@REST { path = "/ping"; method = "GET"; }
	action ping { perform PingUC; }
}`

func genAdapter(t *testing.T) string {
	t.Helper()
	m := engine.New().AddFile("api.sysml", adapterModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	src, err := GenerateAdapter(m, "api")
	if err != nil {
		t.Fatalf("GenerateAdapter: %v", err)
	}
	return src
}

func TestAdapterShape(t *testing.T) {
	src := genAdapter(t)
	for _, want := range []string{
		"type Domain interface",
		"CancelUC(ctx context.Context, in APIOrder) error",
		"PingUC(ctx context.Context) error",
		"PlaceOrderUC(ctx context.Context, in APIOrder) (APIReceipt, error)",
		"StatusUC(ctx context.Context) (APIReceipt, error)",
		"func NewAdapter(domain Domain) Adapter",
		"var _ StrictServerInterface = Adapter{}",
		// in+out: call domain with the body, wrap out in the typed response
		"out, err := a.domain.PlaceOrderUC(ctx, *request.Body)",
		"return PlaceOrder201JSONResponse(out), nil",
		// in, no out: pass the body, return the empty typed response
		"if err := a.domain.CancelUC(ctx, *request.Body); err != nil",
		"return Cancel200Response{}, nil",
		// no in, out
		"out, err := a.domain.StatusUC(ctx)",
		"return Status200JSONResponse(out), nil",
		// no in, no out
		"if err := a.domain.PingUC(ctx); err != nil",
		"return Ping200Response{}, nil",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated adapter missing %q\n---\n%s", want, src)
		}
	}
}

// TestAdapterUnwiredOperation checks an operation with no `perform` still gets a
// method (so the interface stays satisfied) that reports the missing binding.
func TestAdapterUnwiredOperation(t *testing.T) {
	src := `package API {
	import RESTProfile::*;
	@REST { path = "/x"; method = "GET"; }
	action doThing;
}`
	m := engine.New().AddFile("api.sysml", src).Build()
	out, err := GenerateAdapter(m, "api")
	if err != nil {
		t.Fatalf("GenerateAdapter: %v", err)
	}
	if !strings.Contains(out, "has no domain action to perform") {
		t.Errorf("expected an unwired stub method:\n%s", out)
	}
}

// TestAdapterCompilesWithServer generates the server and adapter into one
// package and builds them together, proving the adapter implements the
// generated StrictServerInterface (the `var _` assertion) and every response
// type name matches oapi-codegen's output.
func TestAdapterCompilesWithServer(t *testing.T) {
	m := engine.New().AddFile("api.sysml", adapterModel).Build()
	server, err := GenerateServer(m, "api")
	if err != nil {
		t.Fatalf("GenerateServer: %v", err)
	}
	adapter, err := GenerateAdapter(m, "api")
	if err != nil {
		t.Fatalf("GenerateAdapter: %v", err)
	}

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "server.go"), server)
	mustWrite(t, filepath.Join(dir, "adapter.go"), adapter)
	ginReq := "github.com/gin-gonic/gin " + moduleVersion(t, "github.com/gin-gonic/gin")
	mod := "module gen\n\ngo 1.26.5\n\nrequire " + ginReq + "\n"
	mustWrite(t, filepath.Join(dir, "go.mod"), mod)
	if sum, err := os.ReadFile(parentGoSum(t)); err == nil {
		mustWrite(t, filepath.Join(dir, "go.sum"), string(sum))
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("server+adapter did not compile: %v\n%s\n--- adapter ---\n%s", err, out, adapter)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
