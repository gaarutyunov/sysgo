package mapping

import (
	"strings"

	"github.com/gaarutyunov/sysgo/internal/config"
	"github.com/gaarutyunov/sysgo/internal/core/gocode"
	"github.com/gaarutyunov/sysgo/internal/core/ir"
	"github.com/gaarutyunov/sysgo/internal/core/model"
)

// scalarMap is the built-in SysML ScalarValues/ISQ → Go type mapping (SPEC §8).
var scalarMap = map[string]string{
	"Real":        "float64",
	"Rational":    "float64",
	"Number":      "float64",
	"Integer":     "int64",
	"Natural":     "uint64",
	"Boolean":     "bool",
	"String":      "string",
	"Complex":     "complex128",
	"ScalarValue": "string",
}

// descendants returns all transitive children of e (excluding e), depth-first.
func descendants(e *model.Element) []*model.Element {
	var out []*model.Element
	var walk func(n *model.Element)
	walk = func(n *model.Element) {
		for _, c := range n.Owned {
			out = append(out, c)
			walk(c)
		}
	}
	walk(e)
	return out
}

// defaultPkg derives a lowercase Go package name from a declared name.
func defaultPkg(name string) string {
	n := strings.ToLower(gocode.GoName(name))
	if n == "" {
		return "domain"
	}
	return n
}

// nameOf returns the Go identifier for an element, honoring an x-go-name override.
func nameOf(e *model.Element, meta ir.Metadata) string {
	if meta.GoName != "" {
		return meta.GoName
	}
	return gocode.GoName(e.DeclaredName)
}

// resolveMeta reads generation hints from an element's raw JSON. Overlay
// actions inject x-go-*/x-ddd-* keys here; in-model metadata uses the same keys.
func (m *Mapper) resolveMeta(e *model.Element) ir.Metadata {
	meta := ir.Metadata{
		GoType:      e.StringAttr("x-go-type"),
		GoName:      e.StringAttr("x-go-name"),
		Tags:        e.StringAttr("x-go-tags"),
		Stereotype:  e.StringAttr("x-ddd-stereotype"),
		TargetDir:   e.StringAttr("x-ddd-target-dir"),
		TargetLayer: e.StringAttr("x-ddd-target-layer"),
	}
	meta.SkipOptionalPointer = e.BoolAttr("x-go-skip-optional-pointer")
	if imp := e.StringAttr("x-go-type-import"); imp != "" {
		meta.Imports = append(meta.Imports, imp)
		// Register the selector→import so the renderer can emit the import.
		if sel := gocode.PackageSelector(meta.GoType); sel != "" {
			m.registerImport(sel, imp)
		}
	}
	return meta
}

// registerImport folds a discovered selector→import into additional-imports so
// the renderer's import resolver can find it.
func (m *Mapper) registerImport(selector, path string) {
	for _, ai := range m.Cfg.AdditionalImports {
		if ai.Package == path {
			return
		}
	}
	m.Cfg.AdditionalImports = append(m.Cfg.AdditionalImports, config.ImportSpec{
		Package: path, Alias: selector,
	})
}

// resolveGoType resolves the Go type for a typed feature, applying precedence:
// x-go-type > config type-mapping > built-in scalar map > sibling Go type.
func (m *Mapper) resolveGoType(u *model.Element, meta ir.Metadata) (string, string) {
	if meta.GoType != "" {
		return meta.GoType, ""
	}
	qn := typeName(u)
	if qn == "" {
		return "any", ""
	}
	if tm, ok := m.Cfg.TypeMapping[qn]; ok {
		if tm.Import != "" {
			if sel := gocode.PackageSelector(tm.Type); sel != "" {
				m.registerImport(sel, tm.Import)
			}
		}
		return tm.Type, tm.Import
	}
	base := baseName(qn)
	if tm, ok := m.Cfg.TypeMapping[base]; ok {
		if tm.Import != "" {
			if sel := gocode.PackageSelector(tm.Type); sel != "" {
				m.registerImport(sel, tm.Import)
			}
		}
		return tm.Type, tm.Import
	}
	if g, ok := scalarMap[base]; ok {
		return g, ""
	}
	// Otherwise assume a sibling generated domain type.
	return gocode.GoName(base), ""
}

// typeName extracts the SysML qualified type name referenced by a typed feature.
func typeName(u *model.Element) string {
	if t := u.StringAttr("type"); t != "" {
		return t
	}
	if t := u.StringAttr("declaredType"); t != "" {
		return t
	}
	// Resolve a FeatureTyping relationship: child rel with a "type" reference.
	for _, c := range u.Owned {
		if strings.Contains(c.Type, "Typing") || strings.Contains(c.Type, "Specialization") {
			if t := c.StringAttr("typeName"); t != "" {
				return t
			}
		}
	}
	return ""
}

func baseName(qn string) string {
	if i := strings.LastIndex(qn, "::"); i >= 0 {
		return qn[i+2:]
	}
	return qn
}

// hasIdentity reports whether a part def carries an identity attribute.
func hasIdentity(e *model.Element) bool {
	if e.BoolAttr("x-ddd-aggregate") || e.BoolAttr("isAggregate") {
		return true
	}
	for _, c := range e.Owned {
		if c.Type != "AttributeUsage" && c.Type != "ReferenceUsage" {
			continue
		}
		if c.BoolAttr("isIdentity") || c.BoolAttr("x-ddd-identity") {
			return true
		}
		switch strings.ToLower(c.DeclaredName) {
		case "id", "identifier", "uuid", "guid":
			return true
		}
	}
	return false
}

func isOptional(u *model.Element) bool {
	if u.BoolAttr("optional") || u.BoolAttr("isOptional") {
		return true
	}
	if v, ok := u.Attr("lowerBound"); ok {
		return asInt(v) == 0
	}
	return false
}

func isMany(u *model.Element) bool {
	if u.BoolAttr("many") || u.BoolAttr("isMany") {
		return true
	}
	if v, ok := u.Attr("upperBound"); ok {
		switch t := v.(type) {
		case string:
			return t == "*"
		default:
			return asInt(v) < 0 || asInt(v) > 1
		}
	}
	return false
}

func asInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return 0
}

// direction returns the flow direction of an item/feature: in|out|inout|return.
func direction(e *model.Element) string {
	if d := e.StringAttr("direction"); d != "" {
		return strings.ToLower(d)
	}
	if d := e.StringAttr("FeatureDirection"); d != "" {
		return strings.ToLower(d)
	}
	return "in"
}

// directedFeatures returns the directed parameter features of an action/op.
func directedFeatures(e *model.Element) []*model.Element {
	var out []*model.Element
	for _, c := range e.Owned {
		switch c.Type {
		case "AttributeUsage", "ItemUsage", "PartUsage", "ReferenceUsage", "ParameterUsage":
			if _, ok := c.Attr("direction"); ok {
				out = append(out, c)
			} else if c.Type == "ParameterUsage" {
				out = append(out, c)
			}
		}
	}
	return out
}

func isItem(e *model.Element) bool {
	switch e.Type {
	case "ItemUsage", "AttributeUsage", "ReferenceUsage", "PortUsage":
		_, ok := e.Attr("direction")
		return ok
	}
	return false
}

// portClass resolves a port's direction and kind, metadata overriding heuristics.
func portClass(e *model.Element, meta ir.Metadata) (ir.PortDir, ir.PortKind) {
	switch strings.ToLower(meta.Stereotype) {
	case "driving-port", "driving", "in":
		return ir.DirIn, ir.KindUseCase
	case "driven-port", "driven", "out":
		return ir.DirOut, kindByName(e.DeclaredName)
	case "repository", "repo":
		return ir.DirOut, ir.KindRepository
	case "gateway":
		return ir.DirOut, ir.KindGateway
	}
	name := strings.ToLower(e.DeclaredName)
	if strings.Contains(name, "repository") || strings.Contains(name, "repo") {
		return ir.DirOut, ir.KindRepository
	}
	// Default heuristic: ports are driven gateways unless told otherwise.
	return ir.DirOut, ir.KindGateway
}

func kindByName(name string) ir.PortKind {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "repository"), strings.Contains(n, "repo"):
		return ir.KindRepository
	case strings.Contains(n, "gateway"):
		return ir.KindGateway
	default:
		return ir.KindGateway
	}
}

// jsonTag derives a json tag value from a declared name (lowerCamel).
func jsonTag(name string) string {
	return gocode.JSONName(name)
}
