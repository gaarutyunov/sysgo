package parser

import (
	"os"
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine/cst"
)

func TestParseSysmlDefinitions(t *testing.T) {
	for _, src := range []string{
		"part def P;",
		"attribute def Money {}",
		"item def ChargeRequest {}",
		"port def PaymentPort {}",
		"action def PlaceOrder;",
		"metadata def M;",
		"enum def Color {}",
		"connection def C;",
	} {
		tree := roundTrip(t, src)
		k := nodeKinds(tree.Root())
		if countKind(k, "Def") != 1 {
			t.Errorf("Parse(%q): Def count = %d, want 1 (%v)", src, countKind(k, "Def"), k)
		}
	}
}

func TestParseSysmlUsages(t *testing.T) {
	for _, src := range []string{
		"part p : P;",
		"attribute mass : Real;",
		"port pay : PaymentPort;",
		"item charge : ChargeRequest;",
	} {
		tree := roundTrip(t, src)
		if countKind(nodeKinds(tree.Root()), "Usage") != 1 {
			t.Errorf("Parse(%q): want one Usage, got %v", src, nodeKinds(tree.Root()))
		}
	}
}

func TestParseDirectedUsages(t *testing.T) {
	// Directed usages: a bare direction modifier with no noun keyword still
	// parses as a Usage.
	for _, src := range []string{
		"in item receipt : Receipt;",
		"out item charge : ChargeRequest;",
		"in order : Order;",
	} {
		tree := roundTrip(t, src)
		if countKind(nodeKinds(tree.Root()), "Usage") != 1 {
			t.Errorf("Parse(%q): want one Usage, got %v", src, nodeKinds(tree.Root()))
		}
	}
}

func TestParseMultiplicity(t *testing.T) {
	for _, src := range []string{
		"part lines : LineItem[*];",
		"attribute xs : Real[0..1];",
		"part p : P[1];",
	} {
		tree := roundTrip(t, src)
		if countKind(nodeKinds(tree.Root()), "Multiplicity") != 1 {
			t.Errorf("Parse(%q): want one Multiplicity, got %v", src, nodeKinds(tree.Root()))
		}
	}
}

func TestParseAnnotations(t *testing.T) {
	for _, src := range []string{
		"@Approved part def X;",
		"#command action a;",
	} {
		tree := roundTrip(t, src)
		if countKind(nodeKinds(tree.Root()), "Annotation") != 1 {
			t.Errorf("Parse(%q): want one Annotation, got %v", src, nodeKinds(tree.Root()))
		}
	}
}

func TestParseSysmlDefGolden(t *testing.T) {
	tree := Parse("part def P;")
	want := strings.Join([]string{
		`SourceFile [0, 11)`,
		`  Def [0, 11)`,
		`    Ident [0, 4) "part"`,
		`    Whitespace [4, 5) " "`,
		`    Ident [5, 8) "def"`,
		`    QualifiedName [8, 10)`,
		`      Name [8, 10)`,
		`        Whitespace [8, 9) " "`,
		`        Ident [9, 10) "P"`,
		`    Semicolon [10, 11) ";"`,
		``,
	}, "\n")
	if got := cst.Print(tree.Root(), Namer); got != want {
		t.Errorf("Print mismatch:\n got:\n%s\nwant:\n%s", got, want)
	}
}

// TestParseExampleModel parses the committed SysML example end-to-end: it must
// round-trip losslessly and parse cleanly (no error nodes).
func TestParseExampleModel(t *testing.T) {
	const path = "../../examples/order/OrderContext.sysml"
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	tree := Parse(string(src))
	if got := tree.Root().Text(); got != string(src) {
		t.Fatalf("example round-trip mismatch")
	}
	k := nodeKinds(tree.Root())
	if n := countKind(k, "ErrorNode"); n != 0 {
		t.Errorf("example parsed with %d error node(s); grammar gap", n)
	}
	// Sanity: the model has several definitions and usages.
	if countKind(k, "Def") < 5 {
		t.Errorf("Def count = %d, want >= 5", countKind(k, "Def"))
	}
	if countKind(k, "Usage") < 5 {
		t.Errorf("Usage count = %d, want >= 5", countKind(k, "Usage"))
	}
}
