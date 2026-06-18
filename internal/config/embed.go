package config

import (
	"bytes"
	_ "embed"
)

//go:embed schema.json
var schemaJSON []byte

// SchemaJSON returns a copy of the embedded JSON Schema for sysgo.yaml so
// callers cannot mutate the package-level slice.
func SchemaJSON() []byte { return bytes.Clone(schemaJSON) }
