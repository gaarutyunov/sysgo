package contracts

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"

	// Blank import keeps gin — the web framework the generated gin server needs
	// — in the module graph and cache so TestGeneratedServerCompiles can build
	// the generated code from cache.
	_ "github.com/gin-gonic/gin"
)

const serverModel = `package API {
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

func genServer(t *testing.T, pkg string) string {
	t.Helper()
	m := engine.New().AddFile("api.sysml", serverModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	src, err := GenerateServer(m, pkg)
	if err != nil {
		t.Fatalf("GenerateServer: %v", err)
	}
	return src
}

// TestGenerateServerShape checks the generated source is valid Go and exposes
// the full server/client surface oapi-codegen is configured to emit.
func TestGenerateServerShape(t *testing.T) {
	src := genServer(t, "api")
	if _, err := parser.ParseFile(token.NewFileSet(), "gen.go", src, parser.AllErrors); err != nil {
		t.Fatalf("generated code is not valid Go: %v", err)
	}
	for _, want := range []string{
		"package api",
		"type ServerInterface interface",
		"type StrictServerInterface interface",
		"func RegisterHandlers",
		"type Client struct",
		"type APIOrder struct",
		"type ProblemDetails struct",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated server missing %q", want)
		}
	}
}

// TestGenerateServerError surfaces generation failures (invalid config etc.).
func TestGenerateModelsShape(t *testing.T) {
	m := engine.New().AddFile("api.sysml", serverModel).Build()
	src, err := GenerateModels(m, "models")
	if err != nil {
		t.Fatalf("GenerateModels: %v", err)
	}
	if !strings.Contains(src, "type APIOrder struct") {
		t.Errorf("models missing Order struct:\n%s", src)
	}
	if strings.Contains(src, "gin-gonic/gin") {
		t.Error("models-only output should not import gin")
	}
}

// TestGeneratedModelsCompile compiles the models-only output — stdlib-only, so
// it builds in a throwaway module with no external dependencies.
func TestGeneratedModelsCompile(t *testing.T) {
	m := engine.New().AddFile("api.sysml", serverModel).Build()
	src, err := GenerateModels(m, "models")
	if err != nil {
		t.Fatalf("GenerateModels: %v", err)
	}
	dir := t.TempDir()
	writeModule(t, dir, src, "")
	compileModule(t, dir, src)
}

// TestGeneratedServerCompiles compiles the full gin strict-server + client +
// models output, verifying the oapi-codegen integration produces buildable Go.
// The generated code's only external dependency is gin, which the blank import
// above keeps in the module cache, so the build resolves offline.
func TestGeneratedServerCompiles(t *testing.T) {
	src := genServer(t, "api")
	ginReq := "github.com/gin-gonic/gin " + moduleVersion(t, "github.com/gin-gonic/gin")
	dir := t.TempDir()
	writeModule(t, dir, src, ginReq)
	compileModule(t, dir, src)
}

// writeModule writes src as gen.go plus a go.mod (optionally requiring extra
// modules) into dir. When a require is given, the parent module's go.sum is
// copied so transitive hashes resolve from cache without network.
func writeModule(t *testing.T, dir, src, require string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "gen.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := "module gen\n\ngo 1.26.5\n"
	if require != "" {
		mod += "\nrequire " + require + "\n"
		if sum, err := os.ReadFile(parentGoSum(t)); err == nil {
			_ = os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
}

// compileModule runs `go build` in dir, resolving any requires from the module cache.
func compileModule(t *testing.T, dir, src string) {
	t.Helper()
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated code did not compile: %v\n%s\n--- source ---\n%s", err, out, src)
	}
}

// moduleVersion returns the selected version of module path in the current
// (parent) module.
func moduleVersion(t *testing.T, path string) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Version}}", path).Output()
	if err != nil {
		t.Fatalf("go list -m %s: %v", path, err)
	}
	return strings.TrimSpace(string(out))
}

// parentGoSum returns the path to the current module's go.sum.
func parentGoSum(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	return filepath.Join(filepath.Dir(strings.TrimSpace(string(out))), "go.sum")
}
