// Package gocode holds Go identifier and source helpers shared by the mapping
// stage and the template FuncMap, so naming decisions are made in exactly one
// place.
package gocode

import (
	"strings"
	"unicode"
)

// commonInitialisms are upper-cased wholesale to match Go conventions.
var commonInitialisms = map[string]bool{
	"ID": true, "URL": true, "HTTP": true, "API": true, "JSON": true,
	"SQL": true, "UUID": true, "DTO": true, "DB": true, "IO": true,
}

// GoName converts an arbitrary declared name into an exported Go identifier.
func GoName(s string) string {
	parts := splitWords(s)
	var b strings.Builder
	for _, p := range parts {
		up := strings.ToUpper(p)
		if commonInitialisms[up] {
			b.WriteString(up)
			continue
		}
		r := []rune(p)
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	out := b.String()
	if out == "" {
		return "X"
	}
	if unicode.IsDigit(rune(out[0])) {
		return "X" + out
	}
	return out
}

// Unexported lower-cases the first run of the exported name.
func Unexported(s string) string {
	n := GoName(s)
	if n == "" {
		return n
	}
	r := []rune(n)
	// Lowercase a leading initialism run or just the first rune.
	r[0] = unicode.ToLower(r[0])
	out := string(r)
	if isReserved(out) {
		return out + "_"
	}
	return out
}

// JSONName returns a lowerCamelCase JSON field name derived from a declared
// name, without applying Go initialism upcasing (so "id" stays "id").
func JSONName(s string) string {
	words := splitWords(s)
	var b strings.Builder
	for i, w := range words {
		lw := strings.ToLower(w)
		if i == 0 {
			b.WriteString(lw)
			continue
		}
		r := []rune(lw)
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	if b.Len() == 0 {
		return "field"
	}
	return b.String()
}

// Receiver returns a short receiver identifier for a type name.
func Receiver(typeName string) string {
	n := GoName(typeName)
	if n == "" {
		return "x"
	}
	return strings.ToLower(n[:1])
}

// Comment formats text as a Go doc comment block prefixed with name.
func Comment(name, text string) string {
	if text == "" {
		return "// " + name
	}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = "// " + l
	}
	return strings.Join(lines, "\n")
}

// ZeroValue returns the Go zero value literal for a type.
func ZeroValue(goType string) string {
	switch {
	case goType == "":
		return "nil"
	case goType == "string":
		return `""`
	case goType == "bool":
		return "false"
	case strings.HasPrefix(goType, "*"), strings.HasPrefix(goType, "[]"),
		strings.HasPrefix(goType, "map["), strings.HasPrefix(goType, "chan "),
		goType == "any", goType == "error", goType == "interface{}":
		return "nil"
	case isNumeric(goType):
		return "0"
	default:
		return goType + "{}"
	}
}

func isNumeric(t string) bool {
	switch t {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "byte", "rune", "uintptr":
		return true
	}
	return false
}

// PackageSelector returns the package selector of a qualified Go type like
// "money.Money" -> "money", or "" if the type is unqualified or a builtin.
func PackageSelector(goType string) string {
	t := strings.TrimLeft(goType, "*[]")
	t = strings.TrimPrefix(t, "map[")
	if i := strings.LastIndex(t, "."); i > 0 {
		sel := t[:i]
		// Guard against slice/map remnants.
		if j := strings.LastIndexAny(sel, "[] "); j >= 0 {
			sel = sel[j+1:]
		}
		return sel
	}
	return ""
}

func splitWords(s string) []string {
	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	var prev rune
	for i, r := range s {
		switch {
		case r == '_' || r == '-' || r == ' ' || r == '.' || r == ':':
			flush()
		case i > 0 && unicode.IsUpper(r) && unicode.IsLower(prev):
			flush()
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
		prev = r
	}
	flush()
	return words
}

func isReserved(s string) bool {
	switch s {
	case "break", "case", "chan", "const", "continue", "default", "defer",
		"else", "fallthrough", "for", "func", "go", "goto", "if", "import",
		"interface", "map", "package", "range", "return", "select", "struct",
		"switch", "type", "var", "any":
		return true
	}
	return false
}
