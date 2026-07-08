package gotmpl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gaarutyunov/sysgo/internal/app/port"
	"github.com/gaarutyunov/sysgo/internal/config"
	"github.com/gaarutyunov/sysgo/internal/core/gocode"
	"github.com/gaarutyunov/sysgo/internal/core/ir"
)

// This file implements the two orthogonal composition-root axes (SPEC §9.2.1):
//
//   generate.di   whether to wire with a DI toolkit (wire). When enabled sysgo
//                 emits a wire.ProviderSet per context and a wire injector per
//                 binary; when disabled the binary mains wire by hand.
//   generate.cmd  which binaries exist: one per context (per-context), one for
//                 everything (mono), one per context group (custom), or none
//                 (off). Every binary main is a cobra command.

// providerSetData drives templates/cmd/providers.go.tmpl.
type providerSetData struct {
	Marker    string
	Package   string
	Name      string
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

// binaryData drives templates/cmd/main.go.tmpl and templates/cmd/wire.go.tmpl.
type binaryData struct {
	Marker   string
	Package  string
	App      string
	Module   string
	DI       bool
	Multi    bool
	Imports  []importLine
	Contexts []binaryContext
}

// binaryContext is one context wired into a binary.
type binaryContext struct {
	Name         string // bounded-context name (OrderContext)
	Slug         string // package selector for the context's ProviderSet (order)
	Inject       string // exported injector suffix (Order); empty if no wire root
	RootType     string // driving port used as the injector's return type
	RootQual     string // qualified return type (orderin.PlaceOrderUseCase)
	PortInImport string // import path of the context's app/port/in package
}

// binary is one composition-root binary and the contexts it wires.
type binary struct {
	Name     string
	Contexts []*ir.Context
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

// appName derives a fallback application (and cmd directory) name from the
// module path's last segment.
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

// providerSetFile renders internal/<context>/providers.go (generated). It is
// emitted only when DI is enabled.
func (r *Renderer) providerSetFile(ctx *ir.Context, lay layout) (port.File, error) {
	s := slug(ctx.Name)
	provs, imps := r.buildProviders(ctx, lay)
	imps["wire"] = importAlias(config.WireImportPath, "wire")
	d := &providerSetData{
		Marker:    r.cfg.OutputOptions.GeneratedMarker,
		Package:   s,
		Name:      ctx.Name,
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

// binaries resolves the composition-root binaries selected by generate.cmd.
func (r *Renderer) binaries(p *ir.Project) ([]binary, error) {
	switch r.cfg.Generate.Cmd.Mode {
	case config.CmdOff:
		return nil, nil
	case config.CmdMono:
		return []binary{{Name: appName(p.Module), Contexts: p.Contexts}}, nil
	case config.CmdCustom:
		byName := map[string]*ir.Context{}
		for _, ctx := range p.Contexts {
			byName[ctx.Name] = ctx
			byName[slug(ctx.Name)] = ctx
		}
		var bins []binary
		for _, g := range r.cfg.Generate.Cmd.Groups {
			b := binary{Name: g.Name}
			for _, name := range g.Contexts {
				ctx, ok := byName[name]
				if !ok {
					return nil, fmt.Errorf("gotmpl: cmd group %q references unknown context %q", g.Name, name)
				}
				b.Contexts = append(b.Contexts, ctx)
			}
			bins = append(bins, b)
		}
		return bins, nil
	default: // config.CmdPerContext
		bins := make([]binary, 0, len(p.Contexts))
		for _, ctx := range p.Contexts {
			bins = append(bins, binary{Name: slug(ctx.Name), Contexts: []*ir.Context{ctx}})
		}
		return bins, nil
	}
}

// binaryContextView builds a binary's per-context view, resolving the wire
// injector target when DI is on and the context exposes a driving port.
func (r *Renderer) binaryContextView(ctx *ir.Context) binaryContext {
	lay := r.resolveLayout(ctx.Name)
	s := slug(ctx.Name)
	bc := binaryContext{Name: ctx.Name, Slug: s, PortInImport: lay.portInImport}
	if r.cfg.Generate.DI.Enabled && r.cfg.Generate.Ports && len(ctx.DrivingPorts) > 0 {
		dp := ctx.DrivingPorts[0]
		bc.Inject = gocode.GoName(s)
		bc.RootType = dp.Name
		bc.RootQual = s + "in." + dp.Name
	}
	return bc
}

// binaryFiles renders one binary's cobra main (scaffold-once) and, when DI is
// enabled and at least one context has an injectable root, its wire injector
// stub (scaffold-once, //go:build wireinject).
func (r *Renderer) binaryFiles(p *ir.Project, b binary) ([]port.File, error) {
	di := r.cfg.Generate.DI.Enabled
	scaffoldMarker := "// Code scaffolded by sysgo; edit freely (not regenerated)."

	views := make([]binaryContext, 0, len(b.Contexts))
	for _, ctx := range b.Contexts {
		views = append(views, r.binaryContextView(ctx))
	}

	// The no-DI scaffold is a minimal stdlib main; DI upgrades it to a
	// cobra + wire entrypoint. Keeping the default dependency-free means the
	// generated project compiles against the standard library alone.
	imports := []importLine{{Path: "log"}}
	if di {
		imports = append(imports, importLine{Path: "github.com/spf13/cobra"})
	}
	main := &binaryData{
		Marker:   scaffoldMarker,
		Package:  "main",
		App:      b.Name,
		Module:   p.Module,
		DI:       di,
		Multi:    len(b.Contexts) > 1,
		Imports:  imports,
		Contexts: views,
	}
	mainContent, err := r.exec("main.go.tmpl", main)
	if err != nil {
		return nil, err
	}
	files := []port.File{{
		Path:         "cmd/" + b.Name + "/main.go",
		Content:      mainContent,
		ScaffoldOnce: true,
	}}

	if !di {
		return files, nil
	}

	// wire.go injectors — only for contexts that have an injectable root.
	imps := map[string]importLine{"wire": importAlias(config.WireImportPath, "wire")}
	injectable := false
	for _, c := range views {
		if c.Inject == "" {
			continue
		}
		injectable = true
		imps[c.Slug] = importAlias(p.Module+"/internal/"+c.Slug, c.Slug)
		imps[c.Slug+"in"] = importLine{Alias: c.Slug + "in", Path: c.PortInImport}
	}
	if injectable {
		wire := &binaryData{
			Marker:   scaffoldMarker,
			Package:  "main",
			App:      b.Name,
			Imports:  sortedImports(imps),
			Contexts: views,
		}
		wireContent, err := r.exec("wire.go.tmpl", wire)
		if err != nil {
			return nil, err
		}
		files = append(files, port.File{
			Path:         "cmd/" + b.Name + "/wire.go",
			Content:      wireContent,
			ScaffoldOnce: true,
		})
	}

	return files, nil
}

// cmdFiles renders every composition-root binary for the project.
func (r *Renderer) cmdFiles(p *ir.Project) ([]port.File, error) {
	bins, err := r.binaries(p)
	if err != nil {
		return nil, err
	}
	var files []port.File
	for _, b := range bins {
		fs, err := r.binaryFiles(p, b)
		if err != nil {
			return nil, err
		}
		files = append(files, fs...)
	}
	return files, nil
}
