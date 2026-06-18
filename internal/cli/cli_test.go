package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const model = `[
  {"@id":"p","@type":"Package","declaredName":"OrderContext","ownedElement":[{"@id":"o"}]},
  {"@id":"o","@type":"PartDefinition","declaredName":"Order","ownedElement":[{"@id":"id"}]},
  {"@id":"id","@type":"AttributeUsage","declaredName":"id","type":"String"}
]`

func writeProject(t *testing.T) (dir, cfgPath string) {
	t.Helper()
	dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "model.json"), []byte(model), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath = filepath.Join(dir, "sysgo.yaml")
	cfg := "module: github.com/acme/orders\nsource:\n  file: ./model.json\n"
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, cfgPath
}

func run(t *testing.T, args ...string) string {
	t.Helper()
	cmd := NewRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute %v: %v\n%s", args, err, buf.String())
	}
	return buf.String()
}

func TestVersionCmd(t *testing.T) {
	out := run(t, "version")
	if !strings.Contains(out, "sysgo") {
		t.Fatalf("version output = %q", out)
	}
}

func TestGenerateCmd(t *testing.T) {
	dir, cfgPath := writeProject(t)
	out := run(t, "generate", "-c", cfgPath, "--out", dir)
	if !strings.Contains(out, "wrote") {
		t.Fatalf("generate output = %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "internal/order/domain/order.go")); err != nil {
		t.Fatalf("expected generated file: %v", err)
	}
}

func TestValidateCmd(t *testing.T) {
	_, cfgPath := writeProject(t)
	out := run(t, "validate", "-c", cfgPath)
	if !strings.Contains(out, "Order (aggregate)") {
		t.Fatalf("validate output = %q", out)
	}
}

func TestInitCmd(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	run(t, "init")
	for _, f := range []string{"sysgo.yaml", "overlay.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Fatalf("init did not create %s: %v", f, err)
		}
	}
}
