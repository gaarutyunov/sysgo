package config

import _ "embed"

//go:embed schema.json
var schemaJSON []byte

// SchemaJSON returns the embedded JSON Schema for sysgo.yaml.
func SchemaJSON() []byte { return schemaJSON }
