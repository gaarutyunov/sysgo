package gotmpl

import (
	"sort"
	"strings"

	"github.com/gaarutyunov/sysgo/internal/config"
	"github.com/gaarutyunov/sysgo/internal/core/gocode"
	"github.com/gaarutyunov/sysgo/internal/core/ir"
)

// layout holds the resolved per-region directories, packages, and import paths
// for one bounded context.
type layout struct {
	domainDir, domainPkg   string
	portInDir, portInPkg   string
	portOutDir, portOutPkg string
	usecaseDir, usecasePkg string
	adapterDir             string
	cmdDir, cmdPkg         string
	domainImport           string
	portInImport           string
	portOutImport          string
}

// slug derives the {context} interpolation value from a context name, dropping
// a trailing "Context" suffix (OrderContext -> order).
func slug(ctxName string) string {
	n := strings.TrimSuffix(ctxName, "Context")
	n = strings.ToLower(gocode.GoName(n))
	if n == "" {
		return "core"
	}
	return n
}

func (r *Renderer) resolveLayout(ctxName string) layout {
	s := slug(ctxName)
	interp := func(region string) (dir, pkg string) {
		reg := r.cfg.Layout[region]
		dir = strings.ReplaceAll(reg.Dir, "{context}", s)
		pkg = reg.Package
		return
	}
	base := "internal/" + s

	domainDir, domainPkg := interp("domain")
	usecaseDir, usecasePkg := interp("app")
	adapterDir, _ := interp("adapters")
	cmdDir, cmdPkg := interp("cmd")

	portInDir := base + "/" + strings.TrimPrefix(r.cfg.Ports.DrivingDir, "/")
	portOutDir := base + "/" + strings.TrimPrefix(r.cfg.Ports.DrivenDir, "/")

	lay := layout{
		domainDir:  domainDir,
		domainPkg:  domainPkg,
		portInDir:  portInDir,
		portInPkg:  lastSeg(portInDir),
		portOutDir: portOutDir,
		portOutPkg: lastSeg(portOutDir),
		usecaseDir: usecaseDir,
		usecasePkg: usecasePkg,
		adapterDir: adapterDir,
		cmdDir:     cmdDir,
		cmdPkg:     cmdPkg,
	}
	lay.domainImport = r.cfg.Module + "/" + domainDir
	lay.portInImport = r.cfg.Module + "/" + portInDir
	lay.portOutImport = r.cfg.Module + "/" + portOutDir

	// repository-in-domain moves driven interfaces into the domain region.
	if r.cfg.Ports.RepositoryInDomain {
		lay.portOutDir = domainDir
		lay.portOutPkg = domainPkg
		lay.portOutImport = lay.domainImport
	}
	return lay
}

func lastSeg(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// typeSets indexes the cross-region type names of a context so the qualifier
// can decide which package selector (if any) prefixes a type.
type typeSets struct {
	domain  set
	portIn  set
	portOut set
	extern  map[string]config.ImportSpec
}

type set struct {
	names map[string]bool
	sel   string
	path  string
}

func (r *Renderer) buildTypeSets(ctx *ir.Context, lay layout) *typeSets {
	ts := &typeSets{
		domain:  set{names: map[string]bool{}, sel: lay.domainPkg, path: lay.domainImport},
		portIn:  set{names: map[string]bool{}, sel: lay.portInPkg, path: lay.portInImport},
		portOut: set{names: map[string]bool{}, sel: lay.portOutPkg, path: lay.portOutImport},
		extern:  map[string]config.ImportSpec{},
	}
	for _, e := range ctx.Entities {
		ts.domain.names[e.Name] = true
	}
	for _, v := range ctx.ValueObjects {
		ts.domain.names[v.Name] = true
	}
	for _, ev := range ctx.Events {
		ts.domain.names[ev.Name] = true
	}
	for _, s := range ctx.DomainServices {
		ts.domain.names[s.Name] = true
	}
	for _, p := range ctx.DrivingPorts {
		ts.portIn.names[p.Name] = true
	}
	for _, uc := range ctx.UseCases {
		ts.portIn.names[uc.Input.Name] = true
		ts.portIn.names[uc.Output.Name] = true
	}
	for _, p := range ctx.DrivenPorts {
		ts.portOut.names[p.Name] = true
	}
	for _, ai := range r.cfg.AdditionalImports {
		key := ai.Alias
		if key == "" {
			key = lastSeg(ai.Package)
		}
		ts.extern[key] = ai
	}
	return ts
}

// qualifier produces qualified Go types for one file and records imports.
func (ts *typeSets) qualifier(selfPkg string) *qualifier {
	return &qualifier{ts: ts, self: selfPkg, imports: map[string]importLine{}}
}

type qualifier struct {
	ts      *typeSets
	self    string
	imports map[string]importLine
}

// add records an import by path (alias only when it differs from the package).
func (q *qualifier) add(path, alias string) {
	if path == "" {
		return
	}
	if alias != "" && alias == lastSeg(path) {
		alias = ""
	}
	if existing, ok := q.imports[path]; ok && existing.Alias != "" {
		return
	}
	q.imports[path] = importLine{Alias: alias, Path: path}
}

// lines returns the collected imports sorted by path.
func (q *qualifier) lines() []importLine {
	out := make([]importLine, 0, len(q.imports))
	for _, l := range q.imports {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// qualify returns the package-qualified form of a Go type for the current file,
// recording any import it requires.
func (q *qualifier) qualify(goType string) string {
	prefix, base := splitPrefix(goType)
	if base == "" {
		return goType
	}
	// Already-qualified external type (pkg.Type).
	if i := strings.Index(base, "."); i > 0 {
		selpart := base[:i]
		if spec, ok := q.ts.extern[selpart]; ok {
			q.add(spec.Package, spec.Alias)
		}
		return goType
	}
	for _, s := range []set{q.ts.domain, q.ts.portIn, q.ts.portOut} {
		if s.names[base] {
			if q.self == s.sel {
				return prefix + base
			}
			q.add(s.path, "")
			return prefix + s.sel + "." + base
		}
	}
	return goType
}

// splitPrefix separates a leading "*" or "[]" from the base type.
func splitPrefix(t string) (prefix, base string) {
	switch {
	case strings.HasPrefix(t, "[]"):
		return "[]", t[2:]
	case strings.HasPrefix(t, "*"):
		return "*", t[1:]
	default:
		return "", t
	}
}
