package gotmpl

import (
	"sort"
	"strings"

	"github.com/gaarutyunov/sysgo/internal/app/port"
	"github.com/gaarutyunov/sysgo/internal/config"
	"github.com/gaarutyunov/sysgo/internal/core/gocode"
	"github.com/gaarutyunov/sysgo/internal/core/ir"
)

// This file implements the composition-root emission modes selected by
// generate.cmd (SPEC §10):
//
//   per-context  one cmd/<context>d/main.go per bounded context (the default,
//                a microservice per context).
//   off          no cmd/ output; the user wires the application themselves.
//   mono         a single cobra + wire application (cmd/<app>/) plus a
//                configured wire ProviderSet per context.

// providerSetData drives templates/cmd/providers.go.tmpl.
type providerSetData struct {
	Marker    string
	Package   string
	Name      string
	App       string
	Imports   []importLine
	Providers []providerEntry
}

// providerEntry is one constructor in a wire.NewSet, with an optional leading
// comment and an optional interface binding.
type providerEntry struct {
	Comment string
	Ctor    string
	Bind    string
}

// monoData drives templates/cmd/mono_main.go.tmpl and templates/cmd/wire.go.tmpl.
type monoData struct {
	Marker   string
	Package  string
	App      string
	Module   string
	Imports  []importLine
	Contexts []monoContext
}

// monoContext is the mono root's per-context view.
type monoContext struct {
	Name         string // bounded-context name (OrderContext)
	Slug         string // package selector for the context's ProviderSet (order)
	Inject       string // exported injector suffix (Order); empty if no root port
	RootType     string // driving port used as the injector's return type
	RootQual     string // qualified return type (orderin.PlaceOrderUseCase)
	PortInImport string // import path of the context's app/port/in package
}

// importAlias records an import, aliasing it only when the package's own name
// differs from the selector the templates use.
func importAlias(path, sel string) importLine {
	if lastSeg(path) == sel {
		return importLine{Path: path}
	}
	return importLine{Alias: sel, Path: path}
}

func sortedImports(m map[string]importLine) []importLine {
	out := make([]importLine, 0, len(m))
	for _, l := range m {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// appName derives the mono application (and cmd directory) name from the module
// path's last segment.
func appName(module string) string {
	seg := strings.ToLower(gocode.GoName(lastSeg(module)))
	if seg == "" {
		return "app"
	}
	return seg
}

// buildProviders assembles the wire provider entries for one context, honoring
// which artifacts generate.* actually emits, and reports the imports they need.
func (r *Renderer) buildProviders(ctx *ir.Context, lay layout) ([]providerEntry, map[string]importLine) {
	var provs []providerEntry
	imps := map[string]importLine{}
	use := func(sel, path string) { imps[sel] = importAlias(path, sel) }

	adapters := r.cfg.Generate.Adapters != config.AdaptersOff
	ports := r.cfg.Generate.Ports
	adapterImport := func(sub string) string { return r.cfg.Module + "/" + lay.adapterDir + "/" + sub }

	// Driven adapters (repository/gateway), bound to their port/out interfaces.
	if adapters {
		first := true
		for _, dp := range ctx.DrivenPorts {
			sel, path := "gateway", adapterImport("out/gateway")
			if dp.Kind == ir.KindRepository {
				sel, path = "repository", adapterImport("out/repository")
			}
			use(sel, path)
			e := providerEntry{Ctor: sel + ".New" + dp.Name + "Impl"}
			if first {
				e.Comment = "driven adapters, bound to app/port/out"
				first = false
			}
			if ports {
				use("portout", lay.portOutImport)
				e.Bind = "wire.Bind(new(portout." + dp.Name + "), new(*" + sel + "." + dp.Name + "Impl))"
			}
			provs = append(provs, e)
		}
	}

	// Interactors, bound to their port/in use-case interfaces.
	if r.cfg.Generate.UseCases {
		first := true
		for _, uc := range ctx.UseCases {
			use("usecase", r.cfg.Module+"/"+lay.usecaseDir)
			e := providerEntry{Ctor: "usecase.New" + uc.Name + "Interactor"}
			if first {
				e.Comment = "interactors, bound to app/port/in"
				first = false
			}
			if ports && uc.Port != nil {
				use("portin", lay.portInImport)
				e.Bind = "wire.Bind(new(portin." + uc.Port.Name + "), new(*usecase." + uc.Name + "Interactor))"
			}
			provs = append(provs, e)
		}
	}

	// Driving adapters (HTTP handlers); wire supplies their port/in argument
	// from the bound interactor above.
	if adapters {
		first := true
		for _, dp := range ctx.DrivingPorts {
			use("httpadapter", adapterImport("in/http"))
			e := providerEntry{Ctor: "httpadapter.New" + dp.Name + "Handler"}
			if first {
				e.Comment = "driving adapters (HTTP)"
				first = false
			}
			provs = append(provs, e)
		}
	}

	return provs, imps
}

// providerSetFile renders internal/<context>/providers.go (generated).
func (r *Renderer) providerSetFile(ctx *ir.Context, lay layout, app string) (port.File, error) {
	s := slug(ctx.Name)
	provs, imps := r.buildProviders(ctx, lay)
	imps["wire"] = importAlias(config.WireImportPath, "wire")
	d := &providerSetData{
		Marker:    r.cfg.OutputOptions.GeneratedMarker,
		Package:   s,
		Name:      ctx.Name,
		App:       app,
		Imports:   sortedImports(imps),
		Providers: provs,
	}
	content, err := r.exec("providers.go.tmpl", d)
	if err != nil {
		return port.File{}, err
	}
	return port.File{
		Path:      "internal/" + s + "/providers.go",
		Content:   content,
		Generated: true,
	}, nil
}

// monoContexts builds the mono root's per-context view for every context.
func (r *Renderer) monoContexts(p *ir.Project) []monoContext {
	out := make([]monoContext, 0, len(p.Contexts))
	for _, ctx := range p.Contexts {
		lay := r.resolveLayout(ctx.Name)
		s := slug(ctx.Name)
		mc := monoContext{Name: ctx.Name, Slug: s, PortInImport: lay.portInImport}
		// A context gets a wire injector when it exposes a driving port whose
		// interface exists to serve as the injector's concrete return type.
		if r.cfg.Generate.Ports && len(ctx.DrivingPorts) > 0 {
			dp := ctx.DrivingPorts[0]
			mc.Inject = gocode.GoName(s)
			mc.RootType = dp.Name
			mc.RootQual = s + "in." + dp.Name
		}
		out = append(out, mc)
	}
	return out
}

// monoRootFiles renders the single cobra root (cmd/<app>/main.go) and, when at
// least one context has an injectable root, the wire injector stub
// (cmd/<app>/wire.go). Both are scaffold-once.
func (r *Renderer) monoRootFiles(p *ir.Project) ([]port.File, error) {
	app := appName(p.Module)
	mctx := r.monoContexts(p)
	scaffoldMarker := "// Code scaffolded by sysgo; edit freely (not regenerated)."

	main := &monoData{
		Marker:   scaffoldMarker,
		Package:  "main",
		App:      app,
		Module:   p.Module,
		Imports:  []importLine{{Path: "log"}, {Path: "github.com/spf13/cobra"}},
		Contexts: mctx,
	}
	mainContent, err := r.exec("mono_main.go.tmpl", main)
	if err != nil {
		return nil, err
	}
	files := []port.File{{
		Path:         "cmd/" + app + "/main.go",
		Content:      mainContent,
		ScaffoldOnce: true,
	}}

	// wire.go injectors — only for contexts that have an injectable root.
	imps := map[string]importLine{"wire": importAlias(config.WireImportPath, "wire")}
	injectable := false
	for _, c := range mctx {
		if c.Inject == "" {
			continue
		}
		injectable = true
		imps[c.Slug] = importAlias(p.Module+"/internal/"+c.Slug, c.Slug)
		imps[c.Slug+"in"] = importLine{Alias: c.Slug + "in", Path: c.PortInImport}
	}
	if injectable {
		wire := &monoData{
			Marker:   scaffoldMarker,
			Package:  "main",
			App:      app,
			Imports:  sortedImports(imps),
			Contexts: mctx,
		}
		wireContent, err := r.exec("wire.go.tmpl", wire)
		if err != nil {
			return nil, err
		}
		files = append(files, port.File{
			Path:         "cmd/" + app + "/wire.go",
			Content:      wireContent,
			ScaffoldOnce: true,
		})
	}

	return files, nil
}
