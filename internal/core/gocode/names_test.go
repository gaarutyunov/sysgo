package gocode

import "testing"

func TestGoName(t *testing.T) {
	cases := map[string]string{
		"order":       "Order",
		"order_line":  "OrderLine",
		"place order": "PlaceOrder",
		"id":          "ID",
		"http_api":    "HTTPAPI",
		"":            "X",
		"1foo":        "X1foo",
	}
	for in, want := range cases {
		if got := GoName(in); got != want {
			t.Errorf("GoName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJSONName(t *testing.T) {
	cases := map[string]string{
		"id":         "id",
		"OrderLine":  "orderLine",
		"order_line": "orderLine",
	}
	for in, want := range cases {
		if got := JSONName(in); got != want {
			t.Errorf("JSONName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnexportedReserved(t *testing.T) {
	if got := Unexported("type"); got != "type_" {
		t.Errorf("Unexported(type) = %q", got)
	}
}

func TestZeroValue(t *testing.T) {
	cases := map[string]string{
		"string": `""`,
		"int64":  "0",
		"bool":   "false",
		"*Order": "nil",
		"[]Line": "nil",
		"error":  "nil",
		"Money":  "Money{}",
	}
	for in, want := range cases {
		if got := ZeroValue(in); got != want {
			t.Errorf("ZeroValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPackageSelector(t *testing.T) {
	if got := PackageSelector("money.Money"); got != "money" {
		t.Errorf("got %q", got)
	}
	if got := PackageSelector("[]domain.Order"); got != "domain" {
		t.Errorf("got %q", got)
	}
	if got := PackageSelector("string"); got != "" {
		t.Errorf("got %q", got)
	}
}
