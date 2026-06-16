// Package mapping turns a resolved SysML element graph into the DDD IR. It
// applies the default heuristic mapping (SPEC §7.1) with explicit metadata and
// overlay-injected keys overriding heuristics (SPEC §7.2), plus type mapping
// (SPEC §8).
package mapping

import (
	"sort"
	"strings"

	"github.com/gaarutyunov/sysgo/internal/config"
	"github.com/gaarutyunov/sysgo/internal/core/gocode"
	"github.com/gaarutyunov/sysgo/internal/core/ir"
	"github.com/gaarutyunov/sysgo/internal/core/model"
)

// Diagnostic is a structured mapping note surfaced by `sysgo validate`.
type Diagnostic struct {
	ElementID    string
	DeclaredName string
	Rule         string
	Severity     string // info | warn | error
	Message      string
}

// Mapper builds the IR from a graph using the supplied configuration.
type Mapper struct {
	Cfg         *config.Config
	Diagnostics []Diagnostic
}

// New returns a Mapper bound to cfg.
func New(cfg *config.Config) *Mapper { return &Mapper{Cfg: cfg} }

// Build implements port.Builder.
func (m *Mapper) Build(g *model.Graph) (*ir.Project, error) {
	p := &ir.Project{Module: m.Cfg.Module}
	for _, root := range g.Roots {
		m.collectContexts(root, p)
	}
	// If no Package roots produced contexts, synthesize a default context from
	// any loose domain elements under roots.
	if len(p.Contexts) == 0 {
		ctx := &ir.Context{Name: "core", Package: "domain"}
		for _, root := range g.Roots {
			m.classify(root, ctx)
			for _, c := range descendants(root) {
				m.classify(c, ctx)
			}
		}
		if hasContent(ctx) {
			p.Contexts = append(p.Contexts, ctx)
		}
	}
	sortProject(p)
	return p, nil
}

// collectContexts walks the tree creating a Context per Package.
func (m *Mapper) collectContexts(e *model.Element, p *ir.Project) {
	if e.Type == "Package" || e.Type == "LibraryPackage" {
		ctx := &ir.Context{
			Name:    gocode.GoName(e.DeclaredName),
			Package: defaultPkg(e.DeclaredName),
		}
		for _, child := range descendants(e) {
			if child.Type == "Package" {
				continue // nested packages become their own contexts
			}
			m.classify(child, ctx)
		}
		if hasContent(ctx) {
			p.Contexts = append(p.Contexts, ctx)
		}
		// Recurse into nested packages.
		for _, child := range e.Owned {
			if child.Type == "Package" {
				m.collectContexts(child, p)
			}
		}
		return
	}
	for _, child := range e.Owned {
		m.collectContexts(child, p)
	}
}

// classify maps a single element into the context per stereotype/heuristic.
func (m *Mapper) classify(e *model.Element, ctx *ir.Context) {
	meta := m.resolveMeta(e)
	switch e.Type {
	case "PartDefinition":
		m.classifyPart(e, meta, ctx)
	case "AttributeDefinition":
		ctx.ValueObjects = append(ctx.ValueObjects, m.buildValueObject(e, meta))
	case "PortDefinition":
		m.classifyPort(e, meta, ctx)
	case "ActionDefinition":
		m.buildUseCase(e, meta, ctx)
	case "ItemDefinition":
		// DTO-like; emitted as a value object for v1 structure.
		ctx.ValueObjects = append(ctx.ValueObjects, m.buildValueObject(e, meta))
	case "RequirementDefinition":
		m.note(e, "requirement", "info", "requirement mapped to documentation only")
	}
}

func (m *Mapper) classifyPart(e *model.Element, meta ir.Metadata, ctx *ir.Context) {
	switch strings.ToLower(meta.Stereotype) {
	case "value-object", "value", "valueobject":
		ctx.ValueObjects = append(ctx.ValueObjects, m.buildValueObject(e, meta))
		return
	case "domain-service", "service":
		ctx.DomainServices = append(ctx.DomainServices, m.buildDomainService(e, meta))
		return
	case "event", "domain-event":
		ctx.Events = append(ctx.Events, m.buildEvent(e, meta))
		return
	case "aggregate", "entity":
		ent := m.buildEntity(e, meta)
		ent.Aggregate = strings.EqualFold(meta.Stereotype, "aggregate")
		ctx.Entities = append(ctx.Entities, ent)
		return
	}
	// Heuristic: identity present ⇒ entity/aggregate; else value object.
	if hasIdentity(e) {
		ent := m.buildEntity(e, meta)
		ent.Aggregate = true
		ctx.Entities = append(ctx.Entities, ent)
		m.note(e, "part-def", "info", "classified as aggregate (identity attribute present)")
	} else {
		ctx.ValueObjects = append(ctx.ValueObjects, m.buildValueObject(e, meta))
		m.note(e, "part-def", "info", "classified as value object (no identity)")
	}
}

func (m *Mapper) buildEntity(e *model.Element, meta ir.Metadata) *ir.Entity {
	return &ir.Entity{
		Name:   nameOf(e, meta),
		Fields: m.buildFields(e),
		Meta:   meta,
	}
}

func (m *Mapper) buildValueObject(e *model.Element, meta ir.Metadata) *ir.ValueObject {
	return &ir.ValueObject{
		Name:   nameOf(e, meta),
		Fields: m.buildFields(e),
		Meta:   meta,
	}
}

func (m *Mapper) buildEvent(e *model.Element, meta ir.Metadata) *ir.DomainEvent {
	return &ir.DomainEvent{
		Name:   nameOf(e, meta),
		Fields: m.buildFields(e),
		Meta:   meta,
	}
}

func (m *Mapper) buildDomainService(e *model.Element, meta ir.Metadata) *ir.DomainService {
	return &ir.DomainService{
		Name:    nameOf(e, meta),
		Methods: m.buildMethods(e),
		Meta:    meta,
	}
}

// buildFields maps the attribute/part usages owned by a definition into fields.
func (m *Mapper) buildFields(e *model.Element) []*ir.Field {
	var fields []*ir.Field
	for _, c := range e.Owned {
		switch c.Type {
		case "AttributeUsage", "PartUsage", "ItemUsage", "ReferenceUsage":
			fields = append(fields, m.buildField(c))
		}
	}
	return fields
}

func (m *Mapper) buildField(u *model.Element) *ir.Field {
	meta := m.resolveMeta(u)
	goType, _ := m.resolveGoType(u, meta)
	optional := isOptional(u)
	many := isMany(u)

	if many {
		goType = "[]" + goType
	}
	pointer := optional && !many && !meta.SkipOptionalPointer
	if pointer {
		goType = "*" + goType
	}
	name := nameOf(u, meta)
	tags := meta.Tags
	if tags == "" {
		tags = "`json:\"" + jsonTag(u.DeclaredName) + "\"`"
	}
	return &ir.Field{
		Name:     name,
		GoType:   goType,
		Optional: optional,
		Pointer:  pointer,
		Tags:     tags,
		Doc:      u.StringAttr("documentation"),
	}
}

// buildMethods maps owned action/feature elements into method signatures.
func (m *Mapper) buildMethods(e *model.Element) []*ir.Method {
	var methods []*ir.Method
	for _, c := range e.Owned {
		if c.Type == "ActionUsage" || c.Type == "OperationUsage" || c.Type == "ActionDefinition" {
			methods = append(methods, m.buildMethod(c))
		}
	}
	if len(methods) == 0 {
		// Provide at least one operation derived from the element itself.
		methods = append(methods, m.buildMethod(e))
	}
	return methods
}

func (m *Mapper) buildMethod(e *model.Element) *ir.Method {
	meta := m.resolveMeta(e)
	method := &ir.Method{Name: nameOf(e, meta), Doc: e.StringAttr("documentation")}
	for _, c := range directedFeatures(e) {
		fMeta := m.resolveMeta(c)
		goType, _ := m.resolveGoType(c, fMeta)
		p := &ir.Param{Name: gocode.Unexported(c.DeclaredName), GoType: goType}
		if direction(c) == "out" || direction(c) == "return" {
			method.Results = append(method.Results, p)
		} else {
			method.Params = append(method.Params, p)
		}
	}
	method.Results = append(method.Results, &ir.Param{Name: "err", GoType: "error"})
	return method
}

func (m *Mapper) classifyPort(e *model.Element, meta ir.Metadata, ctx *ir.Context) {
	dir, kind := portClass(e, meta)
	port := &ir.Port{
		Name:      nameOf(e, meta),
		Direction: dir,
		Kind:      kind,
		Methods:   m.buildPortMethods(e),
		Meta:      meta,
	}
	if dir == ir.DirIn {
		ctx.DrivingPorts = append(ctx.DrivingPorts, port)
	} else {
		ctx.DrivenPorts = append(ctx.DrivenPorts, port)
	}
}

// buildPortMethods turns a port's directed items into a single interface method
// (in items → params, out items → results), per SPEC §7.1.
func (m *Mapper) buildPortMethods(e *model.Element) []*ir.Method {
	method := &ir.Method{Name: gocode.GoName(strings.TrimSuffix(e.DeclaredName, "Port"))}
	if method.Name == "" {
		method.Name = "Invoke"
	}
	for _, c := range e.Owned {
		if !isItem(c) {
			continue
		}
		fMeta := m.resolveMeta(c)
		goType, _ := m.resolveGoType(c, fMeta)
		p := &ir.Param{Name: gocode.Unexported(c.DeclaredName), GoType: goType}
		switch direction(c) {
		case "out", "return":
			method.Results = append(method.Results, p)
		default:
			method.Params = append(method.Params, p)
		}
	}
	method.Results = append(method.Results, &ir.Param{Name: "err", GoType: "error"})
	return []*ir.Method{method}
}

func (m *Mapper) buildUseCase(e *model.Element, meta ir.Metadata, ctx *ir.Context) {
	name := nameOf(e, meta)
	in := &ir.DTO{Name: name + "Input"}
	out := &ir.DTO{Name: name + "Output"}
	for _, c := range directedFeatures(e) {
		fMeta := m.resolveMeta(c)
		goType, _ := m.resolveGoType(c, fMeta)
		f := &ir.Field{
			Name:   nameOf(c, fMeta),
			GoType: goType,
			Tags:   "`json:\"" + jsonTag(c.DeclaredName) + "\"`",
		}
		if direction(c) == "out" || direction(c) == "return" {
			out.Fields = append(out.Fields, f)
		} else {
			in.Fields = append(in.Fields, f)
		}
	}
	port := &ir.Port{
		Name:      name + "UseCase",
		Direction: ir.DirIn,
		Kind:      ir.KindUseCase,
		Methods: []*ir.Method{{
			Name:    name,
			Params:  []*ir.Param{{Name: "input", GoType: in.Name}},
			Results: []*ir.Param{{Name: "output", GoType: out.Name}, {Name: "err", GoType: "error"}},
		}},
	}
	uc := &ir.UseCase{Name: name, Input: in, Output: out, Port: port, Meta: meta}
	ctx.UseCases = append(ctx.UseCases, uc)
	ctx.DrivingPorts = append(ctx.DrivingPorts, port)
}

func (m *Mapper) note(e *model.Element, rule, sev, msg string) {
	m.Diagnostics = append(m.Diagnostics, Diagnostic{
		ElementID: e.ID, DeclaredName: e.DeclaredName, Rule: rule, Severity: sev, Message: msg,
	})
}

// ---- IR ordering for byte-stable output ----

func sortProject(p *ir.Project) {
	sort.Slice(p.Contexts, func(i, j int) bool { return p.Contexts[i].Name < p.Contexts[j].Name })
	for _, c := range p.Contexts {
		sort.Slice(c.Entities, func(i, j int) bool { return c.Entities[i].Name < c.Entities[j].Name })
		sort.Slice(c.ValueObjects, func(i, j int) bool { return c.ValueObjects[i].Name < c.ValueObjects[j].Name })
		sort.Slice(c.DomainServices, func(i, j int) bool { return c.DomainServices[i].Name < c.DomainServices[j].Name })
		sort.Slice(c.Events, func(i, j int) bool { return c.Events[i].Name < c.Events[j].Name })
		sort.Slice(c.UseCases, func(i, j int) bool { return c.UseCases[i].Name < c.UseCases[j].Name })
		sort.Slice(c.DrivingPorts, func(i, j int) bool { return c.DrivingPorts[i].Name < c.DrivingPorts[j].Name })
		sort.Slice(c.DrivenPorts, func(i, j int) bool { return c.DrivenPorts[i].Name < c.DrivenPorts[j].Name })
	}
}

func hasContent(c *ir.Context) bool {
	return len(c.Entities)+len(c.ValueObjects)+len(c.DomainServices)+
		len(c.UseCases)+len(c.DrivenPorts)+len(c.DrivingPorts)+len(c.Events) > 0
}
