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

// Decode parses a JSON document produced by SysML v2 tooling. It accepts:
//   - a flat array of element objects (the raw API element serialization),
//   - the API envelope array of {"payload": element, "identity": {...}} objects
//     emitted by the pilot's SysML2JSON / the REST API bulk format,
//   - or an object wrapping the array under "elements"/"content".
func Decode(data []byte) ([]map[string]any, error) {
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err == nil {
		return unwrap(arr), nil
	}
	var wrapped struct {
		Elements []map[string]any `json:"elements"`
		Content  []map[string]any `json:"content"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, fmt.Errorf("sysmlfile: decode: %w", err)
	}
	if len(wrapped.Elements) > 0 {
		return unwrap(wrapped.Elements), nil
	}
	if len(wrapped.Content) > 0 {
		return unwrap(wrapped.Content), nil
	}
	return nil, fmt.Errorf("sysmlfile: no elements found in document")
}

// unwrap extracts the element from each API envelope object {"payload": {...}}.
// Plain element objects (without a payload wrapper) pass through unchanged.
func unwrap(arr []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(arr))
	for _, e := range arr {
		if payload, ok := e["payload"].(map[string]any); ok {
			out = append(out, payload)
			continue
		}
		out = append(out, e)
	}
	return out
}
