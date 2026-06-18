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
	qn := m.typeName(u)
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
// It supports the simplified inline form ("type": "String") and the canonical
// graph form (a FeatureTyping relationship whose "type" references the type
// element by @id).
func (m *Mapper) typeName(u *model.Element) string {
	if t := u.StringAttr("type"); t != "" {
		return t
	}
	if t := u.StringAttr("declaredType"); t != "" {
		return t
	}
	// Canonical form: resolve the FeatureTyping relationship's type reference.
	for _, relID := range refList(u.Raw["ownedRelationship"]) {
		rel := m.elem(relID)
		if rel == nil || !strings.Contains(rel.Type, "FeatureTyping") {
			continue
		}
		if tid := refID(rel.Raw["type"]); tid != "" {
			if te := m.elem(tid); te != nil {
				return te.QualifiedName()
			}
		}
	}
	return ""
}

// elem dereferences an element @id against the graph.
func (m *Mapper) elem(id string) *model.Element {
	if m.g == nil {
		return nil
	}
	return m.g.Elements[id]
}

// isLibrary reports whether an element belongs to the standard library and so
// must not be generated (only used for type resolution).
func isLibrary(e *model.Element) bool {
	return e.Type == "LibraryPackage" || e.BoolAttr("isLibraryElement")
}

// refID extracts an @id from a {"@id": "..."} reference or bare string.
func refID(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if s, ok := t["@id"].(string); ok {
			return s
		}
	}
	return ""
}

// refList normalizes a reference field into a slice of ids.
func refList(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if id := refID(item); id != "" {
				out = append(out, id)
			}
		}
		return out
	default:
		if id := refID(v); id != "" {
			return []string{id}
		}
	}
	return nil
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
	if opt, _, ok := multiplicity(u); ok {
		return opt
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
	if _, many, ok := multiplicity(u); ok {
		return many
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

// multiplicity inspects a feature's owned MultiplicityRange (canonical form),
// returning (optional, many, found). A plain Multiplicity (default 1..1) is
// ignored. Bounds are LiteralInteger values or a LiteralInfinity ("*").
func multiplicity(u *model.Element) (optional, many, found bool) {
	for _, c := range u.Owned {
		if !strings.HasSuffix(c.Type, "MultiplicityRange") {
			continue
		}
		bounds := collectBounds(c)
		switch len(bounds) {
		case 0:
			return false, false, true
		case 1:
			b := bounds[0]
			return false, b.inf || b.val > 1, true
		default:
			lo, hi := bounds[0], bounds[len(bounds)-1]
			optional = !lo.inf && lo.val == 0
			many = hi.inf || hi.val > 1
			return optional, many, true
		}
	}
	return false, false, false
}

type bound struct {
	val int
	inf bool
}

// collectBounds gathers the literal bounds owned by a MultiplicityRange in
// document order (descending through memberships one level if needed).
func collectBounds(mr *model.Element) []bound {
	var out []bound
	var visit func(e *model.Element, depth int)
	visit = func(e *model.Element, depth int) {
		for _, c := range e.Owned {
			switch {
			case strings.HasSuffix(c.Type, "LiteralInfinity"):
				out = append(out, bound{inf: true})
			case strings.HasSuffix(c.Type, "LiteralInteger"):
				out = append(out, bound{val: asInt(c.Raw["value"])})
			default:
				if depth < 2 {
					visit(c, depth+1)
				}
			}
		}
	}
	visit(mr, 0)
	return out
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
