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
		"cmd/orderd/main.go",
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
// the mono root emits a wire injector.
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

// render is a small helper: build a renderer for the given cmd mode and return
// the rendered file set keyed by path.
func render(t *testing.T, cmdMode string, p *ir.Project) map[string]port.File {
	t.Helper()
	cfg := config.Default()
	cfg.Module = "github.com/acme/orders"
	cfg.Source.File = "x"
	cfg.Generate.Cmd = cmdMode
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

func TestCmdPerContextEmitsService(t *testing.T) {
	files := render(t, config.CmdPerContext, proj())
	if _, ok := files["cmd/orderd/main.go"]; !ok {
		t.Fatal("per-context mode should emit cmd/orderd/main.go")
	}
	if _, ok := files["internal/order/providers.go"]; ok {
		t.Fatal("per-context mode should not emit a wire provider set")
	}
}

func TestCmdOffEmitsNoRoot(t *testing.T) {
	files := render(t, config.CmdOff, proj())
	for p := range files {
		if strings.HasPrefix(p, "cmd/") {
			t.Fatalf("off mode should emit no cmd/ files, got %q", p)
		}
		if strings.HasSuffix(p, "/providers.go") {
			t.Fatalf("off mode should emit no provider set, got %q", p)
		}
	}
	// Domain and other artifacts are still emitted.
	if _, ok := files["internal/order/domain/order.go"]; !ok {
		t.Fatal("off mode should still emit domain files")
	}
}

func TestCmdMonoEmitsCobraWire(t *testing.T) {
	files := render(t, config.CmdMono, projWithPort())
	for _, want := range []string{
		"cmd/orders/main.go",
		"cmd/orders/wire.go",
		"internal/order/providers.go",
	} {
		if _, ok := files[want]; !ok {
			t.Fatalf("mono mode missing %q", want)
		}
	}
	if _, ok := files["cmd/orderd/main.go"]; ok {
		t.Fatal("mono mode must not emit the per-context microservice main")
	}

	// The provider set is generated (regenerated); the roots are scaffold-once.
	if !files["internal/order/providers.go"].Generated {
		t.Fatal("providers.go should be marked generated")
	}
	if !files["cmd/orders/main.go"].ScaffoldOnce || !files["cmd/orders/wire.go"].ScaffoldOnce {
		t.Fatal("mono root files should be scaffold-once")
	}

	providers := string(files["internal/order/providers.go"].Content)
	for _, want := range []string{
		"wire.NewSet(",
		`"github.com/goforj/wire"`,
	} {
		if !strings.Contains(providers, want) {
			t.Fatalf("providers.go missing %q:\n%s", want, providers)
		}
	}

	main := string(files["cmd/orders/main.go"].Content)
	if !strings.Contains(main, "cobra.Command") {
		t.Fatalf("mono main.go should build a cobra command:\n%s", main)
	}

	wire := string(files["cmd/orders/wire.go"].Content)
	if !strings.Contains(wire, "//go:build wireinject") {
		t.Fatalf("wire.go should carry the wireinject build tag:\n%s", wire)
	}
	if !strings.Contains(wire, "wire.Build(order.ProviderSet)") {
		t.Fatalf("wire.go should build the context ProviderSet:\n%s", wire)
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
