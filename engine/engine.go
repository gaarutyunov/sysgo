package engine

import (
	"strings"

	"github.com/gaarutyunov/sysgo/engine/hir"
	"github.com/gaarutyunov/sysgo/engine/project"
	"github.com/gaarutyunov/sysgo/engine/text"
)

// Engine builds resolved models from source. Construct one with [New], add
// source files, then call [Engine.Build].
type Engine struct {
	ws *project.Workspace
}

// New returns an engine with the standard library loaded.
func New() *Engine {
	return &Engine{ws: project.New()}
}

// AddFile adds (or replaces) a source file. It returns the engine for chaining.
func (e *Engine) AddFile(key, source string) *Engine {
	e.ws.AddFile(key, source)
	return e
}

// Build resolves all added files against the standard library and returns the
// resolved [Model].
func (e *Engine) Build() *Model {
	return &Model{res: e.ws.Analyze()}
}

// Model is a fully resolved, read-only view of the analyzed sources.
type Model struct {
	res *hir.Result
}

// Root returns the implicit global namespace whose children are the top-level
// packages and declarations of every analyzed file (including the standard
// library).
func (m *Model) Root() Element {
	return Element{sym: m.res.Model.Root, model: m}
}

// Lookup resolves a qualified name ("A::B::C") from the global namespace.
func (m *Model) Lookup(qualified string) (Element, bool) {
	if qualified == "" {
		return Element{}, false
	}
	segs := strings.Split(qualified, "::")
	s := m.res.Model.Resolve(m.res.Model.Root, segs)
	if s == nil {
		return Element{}, false
	}
	return Element{sym: s, model: m}, true
}

// Diagnostics returns the resolution diagnostics for the model (unresolved
// imports and relationship targets).
func (m *Model) Diagnostics() []Diagnostic {
	out := make([]Diagnostic, len(m.res.Diagnostics))
	for i, d := range m.res.Diagnostics {
		out[i] = Diagnostic{Message: d.Message, Range: d.Range}
	}
	return out
}

// Diagnostic is a resolution problem tied to a source range.
type Diagnostic struct {
	Message string
	Range   text.TextRange
}
