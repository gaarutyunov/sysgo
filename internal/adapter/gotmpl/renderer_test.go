package gotmpl

import (
	"strings"
	"testing"

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
