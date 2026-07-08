package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// Default returns a Config populated with built-in defaults (matching the
// layout and options described in SPEC §10).
func Default() *Config {
	return &Config{
		Generate: Generate{
			Domain:     true,
			UseCases:   true,
			Ports:      true,
			Adapters:   AdaptersScaffold,
			Events:     true,
			Tests:      false,
			ImportLint: true,
			Cmd:        CmdPerContext,
		},
		Ports: Ports{
			DrivenDir:  "app/port/out",
			DrivingDir: "app/port/in",
		},
		Layout: map[string]Region{
			"domain":   {Dir: "internal/{context}/domain", Package: "domain"},
			"app":      {Dir: "internal/{context}/app/usecase", Package: "usecase"},
			"ports":    {Dir: "internal/{context}/app/port", Package: "port"},
			"adapters": {Dir: "internal/{context}/adapter", Package: "adapter"},
			"cmd":      {Dir: "cmd/{context}d", Package: "main"},
		},
		OutputOptions: OutputOptions{
			GeneratedMarker: DefaultMarker,
		},
	}
}

// Load reads, schema-validates, and decodes a sysgo.yaml file, layering it over
// the built-in defaults.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return Parse(data)
}

// Parse validates and decodes config bytes over the defaults.
func Parse(data []byte) (*Config, error) {
	if err := Validate(data); err != nil {
		return nil, err
	}
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyDefaults()
	if err := cfg.semanticCheck(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// applyDefaults fills empty required fields after unmarshalling.
func (c *Config) applyDefaults() {
	if c.Generate.Adapters == "" {
		c.Generate.Adapters = AdaptersScaffold
	}
	if c.Generate.Cmd == "" {
		c.Generate.Cmd = CmdPerContext
	}
	if c.Ports.DrivenDir == "" {
		c.Ports.DrivenDir = "app/port/out"
	}
	if c.Ports.DrivingDir == "" {
		c.Ports.DrivingDir = "app/port/in"
	}
	if c.OutputOptions.GeneratedMarker == "" {
		c.OutputOptions.GeneratedMarker = DefaultMarker
	}
	if c.Layout == nil {
		c.Layout = map[string]Region{}
	}
	// Add missing regions and backfill empty dir/package fields on regions that
	// were partially overridden (e.g. `layout: { domain: {} }`).
	for k, def := range Default().Layout {
		reg := c.Layout[k]
		if reg.Dir == "" {
			reg.Dir = def.Dir
		}
		if reg.Package == "" {
			reg.Package = def.Package
		}
		c.Layout[k] = reg
	}
}

// semanticCheck enforces constraints not expressible in JSON Schema.
func (c *Config) semanticCheck() error {
	if c.Module == "" {
		return fmt.Errorf("config: module is required")
	}
	if c.Source.API != nil && c.Source.File != "" {
		return fmt.Errorf("config: source.api and source.file are mutually exclusive")
	}
	if c.Source.API == nil && c.Source.File == "" {
		return fmt.Errorf("config: one of source.api or source.file is required")
	}
	switch c.Generate.Adapters {
	case AdaptersOff, AdaptersScaffold, AdaptersFull:
	default:
		return fmt.Errorf("config: generate.adapters must be one of off|scaffold|full, got %q", c.Generate.Adapters)
	}
	switch c.Generate.Cmd {
	case CmdPerContext, CmdOff, CmdMono:
	default:
		return fmt.Errorf("config: generate.cmd must be one of per-context|off|mono, got %q", c.Generate.Cmd)
	}
	return nil
}

// Validate checks config bytes against the embedded JSON Schema.
func Validate(data []byte) error {
	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("config: invalid YAML: %w", err)
	}
	doc = normalizeYAML(doc)

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("sysgo.schema.json", bytes.NewReader(schemaJSON)); err != nil {
		return fmt.Errorf("config: load schema: %w", err)
	}
	sch, err := compiler.Compile("sysgo.schema.json")
	if err != nil {
		return fmt.Errorf("config: compile schema: %w", err)
	}
	if err := sch.Validate(doc); err != nil {
		return fmt.Errorf("config: schema validation failed: %w", err)
	}
	return nil
}

// normalizeYAML converts map[any]any (from yaml.v3 generic decode) into
// map[string]any so the JSON Schema validator can traverse it.
func normalizeYAML(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			t[k] = normalizeYAML(val)
		}
		return t
	case map[any]any:
		m := make(map[string]any, len(t))
		for k, val := range t {
			m[fmt.Sprint(k)] = normalizeYAML(val)
		}
		return m
	case []any:
		for i, val := range t {
			t[i] = normalizeYAML(val)
		}
		return t
	default:
		return v
	}
}

// Marshal returns the YAML serialization of the config (used by `sysgo init`).
func Marshal(c *Config) ([]byte, error) {
	return yaml.Marshal(c)
}
