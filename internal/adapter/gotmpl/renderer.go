// Package gotmpl implements port.Renderer using Go text/template with an
// embedded default template set (overridable per SPEC §12). It resolves
// cross-region type qualification and per-file imports so generated code
// compiles, and tags each file with the generated/scaffold-once policy.
package gotmpl

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/gaarutyunov/sysgo/internal/app/port"
	"github.com/gaarutyunov/sysgo/internal/config"
	"github.com/gaarutyunov/sysgo/internal/core/ir"
)

// Renderer renders an IR project into Go source files.
type Renderer struct {
	cfg  *config.Config
	tmpl *template.Template
}

// New parses the embedded default templates plus any user overrides.
func New(cfg *config.Config) (*Renderer, error) {
	t := template.New("sysgo").Funcs(funcMap())
	t, err := t.ParseFS(templatesFS,
		"templates/*.tmpl",
		"templates/domain/*.tmpl",
		"templates/app/*.tmpl",
		"templates/adapter/*.tmpl",
		"templates/cmd/*.tmpl",
	)
	if err != nil {
		return nil, fmt.Errorf("gotmpl: parse templates: %w", err)
	}
	r := &Renderer{cfg: cfg, tmpl: t}
	if err := r.applyOverrides(); err != nil {
		return nil, err
	}
	return r, nil
}

// applyOverrides loads user-template overrides (same-name semantics).
func (r *Renderer) applyOverrides() error {
	for name, src := range r.cfg.OutputOptions.UserTemplates {
		content, err := loadTemplateSource(src)
		if err != nil {
			return fmt.Errorf("gotmpl: user template %q: %w", name, err)
		}
		base := name
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		if _, err := r.tmpl.New(base).Parse(string(content)); err != nil {
			return fmt.Errorf("gotmpl: parse user template %q: %w", name, err)
		}
	}
	return nil
}

func loadTemplateSource(src string) ([]byte, error) {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return nil, fmt.Errorf("remote templates not supported offline: %s", src)
	}
	if b, err := os.ReadFile(src); err == nil {
		return b, nil
	}
	// Treat as inline template content.
	return []byte(src), nil
}

// tmplData is the data passed to every template.
type tmplData struct {
	Marker  string
	Package string
	Imports []importLine
	Module  string
	Name    string

	Entity      *ir.Entity
	ValueObject *ir.ValueObject
	Event       *ir.DomainEvent
	Service     *ir.DomainService
	Port        *ir.Port
	UseCase     *ir.UseCase

	PortInPkg  string
	PortOutPkg string
}

type importLine struct {
	Alias string
	Path  string
}

// Render implements port.Renderer.
func (r *Renderer) Render(p *ir.Project) ([]port.File, error) {
	var files []port.File
	for _, ctx := range p.Contexts {
		fs, err := r.renderContext(p, ctx)
		if err != nil {
			return nil, err
		}
		files = append(files, fs...)
	}
	if r.cfg.Generate.Cmd == config.CmdMono {
		mono, err := r.monoRootFiles(p)
		if err != nil {
			return nil, err
		}
		files = append(files, mono...)
	}
	if r.cfg.Generate.ImportLint {
		files = append(files, r.archLintFile())
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func (r *Renderer) renderContext(p *ir.Project, ctx *ir.Context) ([]port.File, error) {
	lay := r.resolveLayout(ctx.Name)
	sets := r.buildTypeSets(ctx, lay)

	var files []port.File
	emit := func(tmplName, pkg, dir, base string, generated, scaffold bool, q *qualifier, build func(d *tmplData)) error {
		d := &tmplData{
			Marker:     r.cfg.OutputOptions.GeneratedMarker,
			Package:    pkg,
			Module:     p.Module,
			Name:       ctx.Name,
			PortInPkg:  lay.portInPkg,
			PortOutPkg: lay.portOutPkg,
		}
		if !generated {
			d.Marker = "// Code scaffolded by sysgo; edit freely (not regenerated)."
		}
		build(d)
		d.Imports = q.lines()
		content, err := r.exec(tmplName, d)
		if err != nil {
			return err
		}
		files = append(files, port.File{
			Path:         dir + "/" + base,
			Content:      content,
			Generated:    generated,
			ScaffoldOnce: scaffold,
		})
		return nil
	}

	if r.cfg.Generate.Domain {
		for _, e := range ctx.Entities {
			q := sets.qualifier(lay.domainPkg)
			ent := qualifyEntity(e, q)
			tmpl := "entity.go.tmpl"
			if e.Aggregate {
				tmpl = "aggregate.go.tmpl"
			}
			if err := emit(tmpl, lay.domainPkg, lay.domainDir, snake(e.Name)+".go", true, false, q,
				func(d *tmplData) { d.Entity = ent }); err != nil {
				return nil, err
			}
			if e.Aggregate {
				fq := sets.qualifier(lay.domainPkg)
				if err := emit("factory.go.tmpl", lay.domainPkg, lay.domainDir, snake(e.Name)+"_factory.go", true, false, fq,
					func(d *tmplData) { d.Entity = ent }); err != nil {
					return nil, err
				}
			}
		}
		for _, vo := range ctx.ValueObjects {
			q := sets.qualifier(lay.domainPkg)
			v := qualifyVO(vo, q)
			if err := emit("value_object.go.tmpl", lay.domainPkg, lay.domainDir, snake(vo.Name)+".go", true, false, q,
				func(d *tmplData) { d.ValueObject = v }); err != nil {
				return nil, err
			}
		}
		if r.cfg.Generate.Events {
			for _, ev := range ctx.Events {
				q := sets.qualifier(lay.domainPkg)
				e := qualifyEvent(ev, q)
				if err := emit("domain_event.go.tmpl", lay.domainPkg, lay.domainDir, snake(ev.Name)+".go", true, false, q,
					func(d *tmplData) { d.Event = e }); err != nil {
					return nil, err
				}
			}
		}
		for _, ds := range ctx.DomainServices {
			q := sets.qualifier(lay.domainPkg)
			s := qualifyService(ds, q)
			if err := emit("domain_service.go.tmpl", lay.domainPkg, lay.domainDir, snake(ds.Name)+".go", true, false, q,
				func(d *tmplData) { d.Service = s }); err != nil {
				return nil, err
			}
			iq := sets.qualifier(lay.domainPkg)
			si := qualifyService(ds, iq)
			iq.add("errors", "")
			if err := emit("domain_service_impl.go.tmpl", lay.domainPkg, lay.domainDir, snake(ds.Name)+"_impl.go", false, true, iq,
				func(d *tmplData) { d.Service = si }); err != nil {
				return nil, err
			}
		}
	}

	if r.cfg.Generate.Ports {
		ucByPort := map[*ir.Port]*ir.UseCase{}
		for _, uc := range ctx.UseCases {
			ucByPort[uc.Port] = uc
		}
		for _, p := range ctx.DrivingPorts {
			q := sets.qualifier(lay.portInPkg)
			pp := qualifyPort(p, q)
			var ucv *ir.UseCase
			if uc, ok := ucByPort[p]; ok {
				ucv = qualifyUseCase(uc, q)
			}
			if err := emit("port_in.go.tmpl", lay.portInPkg, lay.portInDir, snake(p.Name)+".go", true, false, q,
				func(d *tmplData) { d.Port = pp; d.UseCase = ucv }); err != nil {
				return nil, err
			}
		}
		for _, p := range ctx.DrivenPorts {
			q := sets.qualifier(lay.portOutPkg)
			pp := qualifyPort(p, q)
			if err := emit("port_out.go.tmpl", lay.portOutPkg, lay.portOutDir, snake(p.Name)+".go", true, false, q,
				func(d *tmplData) { d.Port = pp }); err != nil {
				return nil, err
			}
		}
	}

	if r.cfg.Generate.UseCases {
		for _, uc := range ctx.UseCases {
			q := sets.qualifier(lay.usecasePkg)
			// Only the port methods are rendered here; qualifying the DTO field
			// internals would pull in unused imports.
			ucCopy := *uc
			ucCopy.Port = qualifyPort(uc.Port, q)
			ucv := &ucCopy
			q.add(lay.portInImport, "")
			q.add("errors", "")
			if err := emit("usecase.go.tmpl", lay.usecasePkg, lay.usecaseDir, snake(uc.Name)+".go", false, true, q,
				func(d *tmplData) { d.UseCase = ucv }); err != nil {
				return nil, err
			}
		}
	}

	if r.cfg.Generate.Adapters != config.AdaptersOff {
		for _, p := range ctx.DrivenPorts {
			q := sets.qualifier(adapterSubPkg(p))
			pp := qualifyPort(p, q)
			q.add(lay.portOutImport, "")
			q.add("errors", "")
			tmpl := "gateway_impl.go.tmpl"
			sub := "out/gateway"
			if p.Kind == ir.KindRepository {
				tmpl = "repo_impl.go.tmpl"
				sub = "out/repository"
			}
			if err := emit(tmpl, adapterSubPkg(p), lay.adapterDir+"/"+sub, snake(p.Name)+".go", false, true, q,
				func(d *tmplData) { d.Port = pp }); err != nil {
				return nil, err
			}
		}
		for _, p := range ctx.DrivingPorts {
			q := sets.qualifier("http")
			pp := qualifyPort(p, q)
			q.add(lay.portInImport, "")
			q.add("net/http", "")
			if err := emit("http_handler.go.tmpl", "http", lay.adapterDir+"/in/http", snake(p.Name)+".go", false, true, q,
				func(d *tmplData) { d.Port = pp }); err != nil {
				return nil, err
			}
		}
	}

	// Composition root. The generate.cmd mode selects the shape: a per-context
	// microservice main (default), a per-context wire ProviderSet feeding the
	// mono root (emitted once in Render), or nothing at all.
	switch r.cfg.Generate.Cmd {
	case config.CmdOff:
		// User wires the application themselves.
	case config.CmdMono:
		pf, err := r.providerSetFile(ctx, lay, appName(p.Module))
		if err != nil {
			return nil, err
		}
		files = append(files, pf)
	default: // config.CmdPerContext
		cq := sets.qualifier(lay.cmdPkg)
		cq.add("log", "")
		if err := emit("main.go.tmpl", lay.cmdPkg, lay.cmdDir, "main.go", false, true, cq,
			func(d *tmplData) {}); err != nil {
			return nil, err
		}
	}

	return files, nil
}

func (r *Renderer) exec(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := r.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, fmt.Errorf("gotmpl: execute %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

// adapterSubPkg returns the Go package name for a driven port's adapter.
func adapterSubPkg(p *ir.Port) string {
	if p.Kind == ir.KindRepository {
		return "repository"
	}
	return "gateway"
}

// archLintFile emits an import-lint ruleset asserting the Dependency Rule.
func (r *Renderer) archLintFile() port.File {
	content := `# Code generated by sysgo; DO NOT EDIT.
# go-arch-lint ruleset asserting the Dependency Rule: dependencies point
# inward (adapter -> app -> domain); domain depends on nothing.
version: 3
allow:
  depOnAnyVendor: true
components:
  domain: { in: internal/**/domain/** }
  app:    { in: internal/**/app/** }
  adapter: { in: internal/**/adapter/** }
  cmd:    { in: cmd/** }
`
	if r.cfg.Generate.Cmd == config.CmdMono {
		// The mono wire ProviderSet lives at the context root and, like cmd,
		// composes every inner region.
		content += "  wiring: { in: internal/*/*.go }\n"
	}
	content += `deps:
  domain:
    mayDependOn: []
  app:
    mayDependOn: [domain]
  adapter:
    mayDependOn: [app, domain]
  cmd:
    mayDependOn: [domain, app, adapter]
`
	if r.cfg.Generate.Cmd == config.CmdMono {
		content += `  wiring:
    mayDependOn: [domain, app, adapter]
`
	}
	return port.File{Path: ".go-arch-lint.yml", Content: []byte(content)}
}
