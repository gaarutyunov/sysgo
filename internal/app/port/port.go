// Package port declares the driven ports (interfaces) of the sysgo generator
// itself. sysgo is a hexagonal application: the pipeline depends only on these
// interfaces, and concrete adapters (sysmlfile, sysmlapi, overlay, gotmpl,
// osfs) are wired in at the composition root.
package port

import (
	"context"

	"github.com/gaarutyunov/sysgo/internal/core/ir"
	"github.com/gaarutyunov/sysgo/internal/core/model"
)

// ModelLoader fetches/reads a SysML v2 element graph as a flat list of decoded
// element objects.
type ModelLoader interface {
	Load(ctx context.Context) ([]map[string]any, error)
}

// OverlayEngine applies overlay actions to the raw model JSON before IR build.
type OverlayEngine interface {
	Apply(elements []map[string]any) ([]map[string]any, error)
}

// File is a single unit of generator output.
type File struct {
	// Path is relative to the output root.
	Path string
	// Content is the rendered (not yet formatted) file content.
	Content []byte
	// Generated marks files carrying the DO NOT EDIT marker (always overwritten).
	Generated bool
	// ScaffoldOnce marks files written only if absent (never overwritten).
	ScaffoldOnce bool
}

// Renderer turns the IR into a set of output files using templates.
type Renderer interface {
	Render(p *ir.Project) ([]File, error)
}

// FileWriter persists rendered files, applying the generated/scaffold-once
// policy, optional formatting, and pruning of stale generated files.
type FileWriter interface {
	Write(root string, files []File) (WriteResult, error)
}

// WriteResult reports what the writer did, for diagnostics and CI freshness.
type WriteResult struct {
	Written []string
	Skipped []string
	Pruned  []string
}

// Builder turns a resolved graph into the IR (the mapping stage).
type Builder interface {
	Build(g *model.Graph) (*ir.Project, error)
}
