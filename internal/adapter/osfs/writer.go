// Package osfs implements port.FileWriter over the local filesystem. It applies
// the generated/scaffold-once policy (SPEC §15), formats Go files with gofmt,
// creates directories, and prunes stale marker-bearing generated files.
package osfs

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gaarutyunov/sysgo/internal/app/port"
)

// Writer persists rendered files under a root directory.
type Writer struct {
	SkipFmt   bool
	SkipPrune bool
	Marker    string
}

// New constructs a Writer.
func New(skipFmt, skipPrune bool, marker string) *Writer {
	return &Writer{SkipFmt: skipFmt, SkipPrune: skipPrune, Marker: marker}
}

// Write implements port.FileWriter.
func (w *Writer) Write(root string, files []port.File) (port.WriteResult, error) {
	var res port.WriteResult
	keep := map[string]bool{}

	for _, f := range files {
		abs := filepath.Join(root, filepath.FromSlash(f.Path))
		keep[filepath.Clean(abs)] = true

		if f.ScaffoldOnce {
			if _, err := os.Stat(abs); err == nil {
				res.Skipped = append(res.Skipped, f.Path)
				continue
			}
		}

		content := f.Content
		if !w.SkipFmt && strings.HasSuffix(f.Path, ".go") {
			formatted, err := format.Source(content)
			if err != nil {
				return res, fmt.Errorf("osfs: gofmt %s: %w", f.Path, err)
			}
			content = formatted
		}

		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return res, fmt.Errorf("osfs: mkdir %s: %w", filepath.Dir(abs), err)
		}
		if err := os.WriteFile(abs, content, 0o644); err != nil {
			return res, fmt.Errorf("osfs: write %s: %w", f.Path, err)
		}
		res.Written = append(res.Written, f.Path)
	}

	if !w.SkipPrune {
		pruned, err := w.prune(root, keep)
		if err != nil {
			return res, err
		}
		res.Pruned = pruned
	}

	sort.Strings(res.Written)
	sort.Strings(res.Skipped)
	sort.Strings(res.Pruned)
	return res, nil
}

// prune removes generated (marker-bearing) files no longer in the keep set.
// Scaffold-once and hand-written files (without the marker) are never touched.
func (w *Writer) prune(root string, keep map[string]bool) ([]string, error) {
	if w.Marker == "" {
		return nil, nil
	}
	marker := []byte(w.Marker)
	var pruned []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		clean := filepath.Clean(path)
		if keep[clean] {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Contains(firstLines(data, 3), marker) {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		pruned = append(pruned, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("osfs: prune: %w", err)
	}
	return pruned, nil
}

// firstLines returns the first n lines of data (the generated marker must
// appear at the top of the file per the Go convention).
func firstLines(data []byte, n int) []byte {
	idx := 0
	for i := 0; i < n; i++ {
		j := bytes.IndexByte(data[idx:], '\n')
		if j < 0 {
			return data
		}
		idx += j + 1
	}
	return data[:idx]
}
