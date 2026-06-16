// Package sysmlfile implements port.ModelLoader over a pre-exported JSON file
// (the SysML v2 API serialization), for offline/CI use.
package sysmlfile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// Loader reads a JSON array of SysML elements from a file.
type Loader struct {
	Path string
}

// New returns a Loader for the given path.
func New(path string) *Loader { return &Loader{Path: path} }

// Load implements port.ModelLoader.
func (l *Loader) Load(_ context.Context) ([]map[string]any, error) {
	data, err := os.ReadFile(l.Path)
	if err != nil {
		return nil, fmt.Errorf("sysmlfile: read %s: %w", l.Path, err)
	}
	return Decode(data)
}

// Decode parses a JSON document that is either a flat array of element objects
// or an object with an "elements"/"content" array.
func Decode(data []byte) ([]map[string]any, error) {
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr, nil
	}
	var wrapped struct {
		Elements []map[string]any `json:"elements"`
		Content  []map[string]any `json:"content"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, fmt.Errorf("sysmlfile: decode: %w", err)
	}
	if len(wrapped.Elements) > 0 {
		return wrapped.Elements, nil
	}
	if len(wrapped.Content) > 0 {
		return wrapped.Content, nil
	}
	return nil, fmt.Errorf("sysmlfile: no elements found in document")
}
