// Package app contains the generator's application core: the pipeline that
// orchestrates load → overlay → IR → render → emit, depending only on the
// driven ports declared in the port subpackage.
package app

import (
	"context"
	"fmt"

	"github.com/gaarutyunov/sysgo/internal/app/port"
	"github.com/gaarutyunov/sysgo/internal/core/ir"
	"github.com/gaarutyunov/sysgo/internal/core/model"
)

// Pipeline wires the stages together. Each field is an interface so adapters
// are swappable and the pipeline is unit-testable.
type Pipeline struct {
	Loader   port.ModelLoader
	Overlay  port.OverlayEngine // may be nil
	Builder  port.Builder
	Renderer port.Renderer
	Writer   port.FileWriter
}

// LoadIR runs load → overlay → IR build and returns the resolved IR project.
func (p *Pipeline) LoadIR(ctx context.Context) (*ir.Project, *model.Graph, error) {
	elements, err := p.Loader.Load(ctx)
	if err != nil {
		return nil, nil, err
	}
	if p.Overlay != nil {
		elements, err = p.Overlay.Apply(elements)
		if err != nil {
			return nil, nil, err
		}
	}
	g, err := model.Build(elements)
	if err != nil {
		return nil, nil, fmt.Errorf("build graph: %w", err)
	}
	proj, err := p.Builder.Build(g)
	if err != nil {
		return nil, nil, fmt.Errorf("build IR: %w", err)
	}
	return proj, g, nil
}

// Generate runs the full pipeline and writes the output under root.
func (p *Pipeline) Generate(ctx context.Context, root string) (port.WriteResult, error) {
	proj, _, err := p.LoadIR(ctx)
	if err != nil {
		return port.WriteResult{}, err
	}
	files, err := p.Renderer.Render(proj)
	if err != nil {
		return port.WriteResult{}, fmt.Errorf("render: %w", err)
	}
	res, err := p.Writer.Write(root, files)
	if err != nil {
		return port.WriteResult{}, fmt.Errorf("write: %w", err)
	}
	return res, nil
}
