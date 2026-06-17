// Package overlay implements an OpenAPI-Overlay-style engine applied to the
// SysML model JSON (a flat array of elements) before IR build. It supports the
// update/remove/copy actions over a focused JSONPath filter subset plus a
// friendlier selector-sugar layer (SPEC §11).
package overlay

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Document is a parsed overlay file.
type Document struct {
	Overlay string   `yaml:"overlay"`
	Info    Info     `yaml:"info"`
	Actions []Action `yaml:"actions"`
}

// Info carries overlay metadata.
type Info struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

// Action is a single overlay operation. Exactly one of update/remove/copy
// applies per action.
type Action struct {
	Target string         `yaml:"target"`
	Update map[string]any `yaml:"update,omitempty"`
	Remove bool           `yaml:"remove,omitempty"`
	Copy   string         `yaml:"copy,omitempty"`

	// Description is informational.
	Description string `yaml:"description,omitempty"`
}

// Engine applies a Document to the model element array.
type Engine struct {
	Doc *Document
}

// Load reads and parses an overlay file.
func Load(path string) (*Engine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("overlay: read %s: %w", path, err)
	}
	return Parse(data)
}

// Parse parses overlay bytes.
func Parse(data []byte) (*Engine, error) {
	var doc Document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("overlay: parse: %w", err)
	}
	return &Engine{Doc: &doc}, nil
}

// Apply implements port.OverlayEngine. It returns a new element slice with all
// actions applied in order.
func (e *Engine) Apply(elements []map[string]any) ([]map[string]any, error) {
	if e == nil || e.Doc == nil {
		return elements, nil
	}
	work := elements
	for i, a := range e.Doc.Actions {
		sel, err := compile(a.Target)
		if err != nil {
			return nil, fmt.Errorf("overlay: action %d: %w", i, err)
		}
		switch {
		case a.Remove:
			work = applyRemove(work, sel)
		case a.Copy != "":
			work = applyCopy(work, sel, a.Copy)
		case a.Update != nil:
			applyUpdate(work, sel, a.Update)
		default:
			return nil, fmt.Errorf("overlay: action %d: no update/remove/copy specified", i)
		}
	}
	return work, nil
}

// applyUpdate recursively merges update into every matched element.
func applyUpdate(elements []map[string]any, sel matcher, update map[string]any) {
	for _, el := range elements {
		if sel.match(el) {
			mergeInto(el, update)
		}
	}
}

// applyRemove drops matched elements (and prunes references to them).
func applyRemove(elements []map[string]any, sel matcher) []map[string]any {
	removed := map[string]bool{}
	out := make([]map[string]any, 0, len(elements))
	for _, el := range elements {
		if sel.match(el) {
			if id, ok := el["@id"].(string); ok {
				removed[id] = true
			}
			continue
		}
		out = append(out, el)
	}
	if len(removed) > 0 {
		pruneRefs(out, removed)
	}
	return out
}

// applyCopy duplicates matched elements, applying the copy target's update
// (Overlay v1.1.0 copy semantics, simplified: deep-copy + new @id suffix).
func applyCopy(elements []map[string]any, sel matcher, suffix string) []map[string]any {
	var dupes []map[string]any
	for _, el := range elements {
		if sel.match(el) {
			c := deepCopy(el)
			if id, ok := c["@id"].(string); ok {
				c["@id"] = id + ":" + suffix
				c["elementId"] = c["@id"]
			}
			dupes = append(dupes, c)
		}
	}
	return append(elements, dupes...)
}

// mergeInto recursively merges src into dst.
func mergeInto(dst, src map[string]any) {
	for k, v := range src {
		if sub, ok := v.(map[string]any); ok {
			if existing, ok := dst[k].(map[string]any); ok {
				mergeInto(existing, sub)
				continue
			}
		}
		dst[k] = v
	}
}

// pruneRefs removes ownedRelationship/ownedElement references pointing at
// removed ids so the graph stays consistent.
func pruneRefs(elements []map[string]any, removed map[string]bool) {
	for _, el := range elements {
		for _, key := range []string{"ownedRelationship", "ownedElement", "ownedRelatedElement"} {
			arr, ok := el[key].([]any)
			if !ok {
				continue
			}
			filtered := make([]any, 0, len(arr))
			for _, item := range arr {
				if id := refID(item); id != "" && removed[id] {
					continue
				}
				filtered = append(filtered, item)
			}
			el[key] = filtered
		}
	}
}

func refID(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if s, ok := t["@id"].(string); ok {
			return s
		}
	}
	return ""
}

func deepCopy(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = deepCopyValue(v)
	}
	return out
}

// deepCopyValue recursively clones maps and slices; scalars are returned as-is.
func deepCopyValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return deepCopy(t)
	case []any:
		cp := make([]any, len(t))
		for i, item := range t {
			cp[i] = deepCopyValue(item)
		}
		return cp
	default:
		return v
	}
}
