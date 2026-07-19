package contracts

import (
	"bytes"
	"encoding/json"
)

// Schema is a subset of JSON Schema 2020-12 sufficient for the item-definition
// mapping. Fields are ordered for stable JSON/YAML output.
type Schema struct {
	Ref        string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type       string             `json:"type,omitempty" yaml:"type,omitempty"`
	Format     string             `json:"format,omitempty" yaml:"format,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required   []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Items      *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	Enum       []string           `json:"enum,omitempty" yaml:"enum,omitempty"`
}

// JSON renders the schema as indented JSON. Map keys (properties) are emitted in
// sorted order by encoding/json, so the output is deterministic.
func (s *Schema) JSON() string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	// Encode error is impossible for this in-memory value graph.
	_ = enc.Encode(s)
	return buf.String()
}
