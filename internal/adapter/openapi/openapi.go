// Package openapi implements port.ContractEmitter: it emits an OpenAPI 3.1
// contract document (openapi.yaml) from a SysML v2 textual-notation source.
//
// This is the first gen/* consumer wired into `sysgo generate`. Unlike the DDD
// pipeline (which loads the SysML v2 API JSON into internal/core), contract
// generators consume the engine's resolved model directly, per OVERVIEW.md F4:
// the engine parses the .sysml source natively (ENGINE.md §5b) and gen/contracts
// builds the document from the resolved model.
package openapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gaarutyunov/sysgo/engine"
	"github.com/gaarutyunov/sysgo/gen/contracts"
	"github.com/gaarutyunov/sysgo/internal/app/port"
)

// DefaultOutPath is where the emitted document is written, relative to the
// generate output root.
const DefaultOutPath = "openapi.yaml"

// Emitter builds openapi.yaml from a SysML textual source file.
type Emitter struct {
	// SysMLPath is the path to the .sysml textual-notation source.
	SysMLPath string
	// OutPath is the output path (relative to the generate root). Defaults to
	// DefaultOutPath when empty.
	OutPath string
}

// New constructs an Emitter for the given .sysml source path.
func New(sysmlPath string) *Emitter {
	return &Emitter{SysMLPath: sysmlPath, OutPath: DefaultOutPath}
}

// Emit implements port.ContractEmitter. It parses and resolves the SysML source
// with the engine, fails on any diagnostic, then builds a deterministic
// OpenAPI 3.1 document.
func (e *Emitter) Emit(_ context.Context) ([]port.File, error) {
	if e.SysMLPath == "" {
		return nil, fmt.Errorf("openapi: no sysml source configured (set source.sysml)")
	}
	src, err := os.ReadFile(e.SysMLPath)
	if err != nil {
		return nil, fmt.Errorf("openapi: read sysml source: %w", err)
	}
	m := engine.New().AddFile(filepath.Base(e.SysMLPath), string(src)).Build()
	if diags := m.Diagnostics(); len(diags) > 0 {
		return nil, fmt.Errorf("openapi: sysml model has %d diagnostic(s): %s", len(diags), diags[0].Message)
	}
	doc := contracts.BuildDocument(m)

	out := e.OutPath
	if out == "" {
		out = DefaultOutPath
	}
	return []port.File{{
		Path:      out,
		Content:   []byte(doc.YAML()),
		Generated: true,
	}}, nil
}
