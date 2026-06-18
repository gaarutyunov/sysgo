package gotmpl

import (
	"strings"
	"text/template"

	"github.com/gaarutyunov/sysgo/internal/core/gocode"
	"github.com/gaarutyunov/sysgo/internal/core/ir"
)

// funcMap is the template helper set (SPEC §12).
func funcMap() template.FuncMap {
	return template.FuncMap{
		"goName":           gocode.GoName,
		"exported":         gocode.GoName,
		"unexported":       gocode.Unexported,
		"receiver":         gocode.Receiver,
		"zeroValue":        gocode.ZeroValue,
		"comment":          gocode.Comment,
		"sig":              methodSig,
		"zeroReturn":       zeroReturn,
		"comparableFields": comparableFields,
		"lower":            strings.ToLower,
	}
}

// methodSig renders a Go method signature: Name(p1 T1, ...) (R1, ...).
func methodSig(m *ir.Method) string {
	var b strings.Builder
	b.WriteString(m.Name)
	b.WriteString("(")
	for i, p := range m.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.Name)
		b.WriteString(" ")
		b.WriteString(p.GoType)
	}
	b.WriteString(")")
	switch len(m.Results) {
	case 0:
	case 1:
		b.WriteString(" ")
		b.WriteString(m.Results[0].GoType)
	default:
		b.WriteString(" (")
		for i, r := range m.Results {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(r.GoType)
		}
		b.WriteString(")")
	}
	return b.String()
}

// zeroReturn renders a return statement producing zero values, substituting a
// not-implemented error for any error result.
func zeroReturn(m *ir.Method) string {
	if len(m.Results) == 0 {
		return "return"
	}
	parts := make([]string, 0, len(m.Results))
	for _, r := range m.Results {
		if r.GoType == "error" {
			parts = append(parts, `errors.New("not implemented")`)
		} else {
			parts = append(parts, gocode.ZeroValue(r.GoType))
		}
	}
	return "return " + strings.Join(parts, ", ")
}

// comparableFields reports whether all fields are comparable with ==.
func comparableFields(fields []*ir.Field) bool {
	for _, f := range fields {
		t := f.GoType
		if strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "map[") || strings.HasPrefix(t, "func") {
			return false
		}
	}
	return true
}
