package project

import (
	"embed"
	"io/fs"
	"sort"
	"strings"

	"github.com/gaarutyunov/sysgo/engine/hir"
)

//go:embed stdlib/*.sysml
var stdlibFS embed.FS

// StdlibUnits returns the embedded standard-library source units, sorted by
// path for a deterministic load order.
func StdlibUnits() []hir.Unit {
	entries, err := fs.ReadDir(stdlibFS, "stdlib")
	if err != nil {
		// The embed directive guarantees the directory exists at build time.
		panic("project: embedded stdlib missing: " + err.Error())
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sysml") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	units := make([]hir.Unit, 0, len(names))
	for _, name := range names {
		data, err := stdlibFS.ReadFile("stdlib/" + name)
		if err != nil {
			panic("project: reading embedded stdlib " + name + ": " + err.Error())
		}
		units = append(units, hir.Unit{Key: "stdlib/" + name, Source: string(data)})
	}
	return units
}

// Workspace is a set of user source files resolved together with the embedded
// standard library.
type Workspace struct {
	files map[string]string
	order []string
}

// New returns an empty workspace. The standard library is loaded implicitly on
// every analysis.
func New() *Workspace {
	return &Workspace{files: make(map[string]string)}
}

// AddFile adds or replaces the source for a file key.
func (w *Workspace) AddFile(key, source string) {
	if _, exists := w.files[key]; !exists {
		w.order = append(w.order, key)
	}
	w.files[key] = source
}

// units returns the standard library followed by the user files, in insertion
// order.
func (w *Workspace) units() []hir.Unit {
	units := StdlibUnits()
	for _, key := range w.order {
		units = append(units, hir.Unit{Key: key, Source: w.files[key]})
	}
	return units
}

// Analyze resolves every user file against the standard library and returns the
// combined model, resolved references, and diagnostics.
func (w *Workspace) Analyze() *hir.Result {
	return hir.AnalyzeUnits(w.units())
}
