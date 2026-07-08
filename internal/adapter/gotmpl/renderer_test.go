package gotmpl

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/internal/app/port"
	"github.com/gaarutyunov/sysgo/internal/config"
	"github.com/gaarutyunov/sysgo/internal/core/ir"
)

func proj() *ir.Project {
	return &ir.Project{
		Module: "github.com/acme/orders",
		Contexts: []*ir.Context{{
			Name:    "OrderContext",
			Package: "order",
			Entities: []*ir.Entity{{
				Name:      "Order",
				Aggregate: true,
				Fields:    []*ir.Field{{Name: "ID", GoType: "string", Tags: "`json:\"id\"`"}},
			}},
		}},
	}
}

func TestRenderFileSetAndPolicy(t *testing.T) {
	cfg := config.Default()
	cfg.Module = "github.com/acme/orders"
	cfg.Source.File = "x"
	r, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	files, err := r.Render(proj())
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]bool{}
	for _, f := range files {
		byPath[f.Path] = true
		if strings.HasSuffix(f.Path, "main.go") && !f.ScaffoldOnce {
			t.Fatal("main.go should be scaffold-once")
		}
		if f.Path == "internal/order/domain/order.go" && !f.Generated {
			t.Fatal("order.go should be marked generated")
		}
	}
	for _, want := range []string{
		"internal/order/domain/order.go",
		"internal/order/domain/order_factory.go",
		".go-arch-lint.yml",
		"cmd/order/main.go",
	} {
		if !byPath[want] {
			t.Fatalf("missing rendered file %q", want)
		}
	}
}

func TestUserTemplateOverride(t *testing.T) {
	cfg := config.Default()
	cfg.Module = "github.com/acme/orders"
	cfg.Source.File = "x"
	cfg.OutputOptions.UserTemplates = map[string]string{
		"domain/aggregate.go.tmpl": "{{ template \"header\" . }}\n// CUSTOM {{ .Entity.Name }}\ntype {{ .Entity.Name }} struct{}\n",
	}
	r, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	files, err := r.Render(proj())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Path == "internal/order/domain/order.go" {
			if !strings.Contains(string(f.Content), "CUSTOM Order") {
				t.Fatalf("override not applied:\n%s", f.Content)
			}
			return
		}
	}
	t.Fatal("order.go not rendered")
}

// projWithPort extends the base fixture with a use case and its driving port so
// DI mode emits a wire injector.
func projWithPort() *ir.Project {
	p := proj()
	ctx := p.Contexts[0]
	port := &ir.Port{Name: "PlaceOrderUseCase", Direction: ir.DirIn, Kind: ir.KindUseCase}
	ctx.DrivingPorts = []*ir.Port{port}
	ctx.UseCases = []*ir.UseCase{{
		Name:   "PlaceOrder",
		Input:  &ir.DTO{Name: "PlaceOrderInput"},
		Output: &ir.DTO{Name: "PlaceOrderOutput"},
		Port:   port,
	}}
	ctx.DrivenPorts = []*ir.Port{{Name: "OrderRepository", Direction: ir.DirOut, Kind: ir.KindRepository}}
	return p
}

// renderCfg renders the project with the given config and returns the file set
// keyed by path.
func renderCfg(t *testing.T, cfg *config.Config, p *ir.Project) map[string]port.File {
	t.Helper()
	cfg.Module = "github.com/acme/orders"
	cfg.Source.File = "x"
	r, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	files, err := r.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]port.File{}
	for _, f := range files {
		byPath[f.Path] = f
	}
	return byPath
}

func TestCmdPerContextNoDI(t *testing.T) {
	files := renderCfg(t, config.Default(), proj())
	main, ok := files["cmd/order/main.go"]
	if !ok {
		t.Fatal("per-context mode should emit cmd/order/main.go (no 'd' suffix)")
	}
	if !main.ScaffoldOnce {
		t.Fatal("binary main should be scaffold-once")
	}
	if !strings.Contains(string(main.Content), "cobra.Command") {
		t.Fatalf("main.go should build a cobra command:\n%s", main.Content)
	}
	if _, ok := files["internal/order/providers.go"]; ok {
		t.Fatal("DI disabled: no wire provider set expected")
	}
	if _, ok := files["cmd/order/wire.go"]; ok {
		t.Fatal("DI disabled: no wire.go expected")
	}
}

func TestCmdOffEmitsNoBinary(t *testing.T) {
	files := renderCfg(t, func() *config.Config {
		c := config.Default()
		c.Generate.Cmd.Mode = config.CmdOff
		return c
	}(), proj())
	for p := range files {
		if strings.HasPrefix(p, "cmd/") {
			t.Fatalf("off mode should emit no cmd/ files, got %q", p)
		}
	}
	if _, ok := files["internal/order/domain/order.go"]; !ok {
		t.Fatal("off mode should still emit domain files")
	}
}

// TestDIProviderSetWithoutCmd proves DI and cmd are orthogonal: providers.go is
// emitted even when the user wires the binary themselves (cmd off).
func TestDIProviderSetWithoutCmd(t *testing.T) {
	files := renderCfg(t, func() *config.Config {
		c := config.Default()
		c.Generate.DI.Enabled = true
		c.Generate.Cmd.Mode = config.CmdOff
		return c
	}(), projWithPort())
	ps, ok := files["internal/order/providers.go"]
	if !ok {
		t.Fatal("DI enabled should emit internal/order/providers.go regardless of cmd mode")
	}
	if !ps.Generated {
		t.Fatal("providers.go should be marked generated")
	}
	for p := range files {
		if strings.HasPrefix(p, "cmd/") {
			t.Fatalf("cmd off should emit no binary, got %q", p)
		}
	}
}

func TestCmdPerContextWithDI(t *testing.T) {
	files := renderCfg(t, func() *config.Config {
		c := config.Default()
		c.Generate.DI.Enabled = true
		return c
	}(), projWithPort())
	for _, want := range []string{
		"cmd/order/main.go",
		"cmd/order/wire.go",
		"internal/order/providers.go",
	} {
		if _, ok := files[want]; !ok {
			t.Fatalf("per-context+DI missing %q", want)
		}
	}
	wire := string(files["cmd/order/wire.go"].Content)
	if !strings.Contains(wire, "//go:build wireinject") {
		t.Fatalf("wire.go should carry the wireinject build tag:\n%s", wire)
	}
	if !strings.Contains(wire, "wire.Build(order.ProviderSet)") {
		t.Fatalf("wire.go should build the context ProviderSet:\n%s", wire)
	}
}

func TestCmdMonoWithDI(t *testing.T) {
	files := renderCfg(t, func() *config.Config {
		c := config.Default()
		c.Generate.DI.Enabled = true
		c.Generate.Cmd.Mode = config.CmdMono
		return c
	}(), projWithPort())
	if _, ok := files["cmd/orders/main.go"]; !ok {
		t.Fatal("mono mode should emit a single cmd/orders/main.go")
	}
	if _, ok := files["cmd/order/main.go"]; ok {
		t.Fatal("mono mode must not emit a per-context binary")
	}
	main := string(files["cmd/orders/main.go"].Content)
	if !strings.Contains(main, "cobra.Command") {
		t.Fatalf("mono main.go should build a cobra command:\n%s", main)
	}
}

func TestCmdCustomGroups(t *testing.T) {
	files := renderCfg(t, func() *config.Config {
		c := config.Default()
		c.Generate.DI.Enabled = true
		c.Generate.Cmd.Mode = config.CmdCustom
		c.Generate.Cmd.Groups = []config.CmdGroup{
			{Name: "commerce", Contexts: []string{"OrderContext"}},
		}
		return c
	}(), projWithPort())
	if _, ok := files["cmd/commerce/main.go"]; !ok {
		t.Fatal("custom mode should emit the group binary cmd/commerce/main.go")
	}
	if _, ok := files["cmd/commerce/wire.go"]; !ok {
		t.Fatal("custom mode with DI should emit cmd/commerce/wire.go")
	}
	if _, ok := files["cmd/order/main.go"]; ok {
		t.Fatal("custom mode must not emit per-context binaries")
	}
}

func TestCmdCustomUnknownContext(t *testing.T) {
	c := config.Default()
	c.Module = "github.com/acme/orders"
	c.Source.File = "x"
	c.Generate.Cmd.Mode = config.CmdCustom
	c.Generate.Cmd.Groups = []config.CmdGroup{{Name: "x", Contexts: []string{"NoSuchContext"}}}
	r, err := New(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Render(projWithPort()); err == nil {
		t.Fatal("expected error for unknown context in custom group")
	}
}

func TestSnake(t *testing.T) {
	cases := map[string]string{
		"Order":             "order",
		"OrderLine":         "order_line",
		"PlaceOrderUseCase": "place_order_use_case",
	}
	for in, want := range cases {
		if got := snake(in); got != want {
			t.Errorf("snake(%q) = %q, want %q", in, got, want)
		}
	}
}
