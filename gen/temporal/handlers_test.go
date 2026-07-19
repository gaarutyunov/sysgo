package temporal

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const handlerModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def Order {
		attribute id : String;
	}
	@Signal { name = "cancel"; }
	action def CancelOrder {
		in order : Order;
	}
	@Query { name = "status"; }
	action def OrderStatus {
		in order : Order;
	}
	action def Plain;
}`

func genHandlers(t *testing.T) string {
	t.Helper()
	m := engine.New().AddFile("app.sysml", handlerModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	src, err := GenerateHandlers(m, "gen")
	if err != nil {
		t.Fatalf("GenerateHandlers: %v", err)
	}
	return src
}

func TestSignalsAndQueriesClassification(t *testing.T) {
	m := engine.New().AddFile("app.sysml", handlerModel).Build()
	sigs := Signals(m)
	if len(sigs) != 1 || sigs[0].Element.Name() != "CancelOrder" || sigs[0].Name != "cancel" {
		t.Errorf("signals = %+v, want one CancelOrder/cancel", sigs)
	}
	queries := Queries(m)
	if len(queries) != 1 || queries[0].Element.Name() != "OrderStatus" || queries[0].Name != "status" {
		t.Errorf("queries = %+v, want one OrderStatus/status", queries)
	}
}

func TestGeneratedHandlersCompile(t *testing.T) {
	mustCompile(t, genHandlers(t))
}

func TestHandlerInterfaceShape(t *testing.T) {
	n := norm(genHandlers(t))
	for _, want := range []string{
		"type Order struct",
		"type Signals interface",
		"CancelOrder(ctx context.Context, order Order) error",
		"type Queries interface",
		"OrderStatus(ctx context.Context, order Order) error",
		"SignalNames = map[string]string",
		`"CancelOrder": "cancel"`,
		"QueryNames = map[string]string",
		`"OrderStatus": "status"`,
	} {
		if !strings.Contains(n, want) {
			t.Errorf("generated source missing %q; got:\n%s", want, genHandlers(t))
		}
	}
}

func TestHandlersDeterministic(t *testing.T) {
	if a, b := genHandlers(t), genHandlers(t); a != b {
		t.Error("GenerateHandlers output is not deterministic")
	}
}

func TestPlainActionNotAHandler(t *testing.T) {
	m := engine.New().AddFile("app.sysml", handlerModel).Build()
	for _, h := range append(Signals(m), Queries(m)...) {
		if h.Element.Name() == "Plain" {
			t.Error("Plain action wrongly classified as a signal/query handler")
		}
	}
}
