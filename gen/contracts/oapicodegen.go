package contracts

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"

	"github.com/gaarutyunov/sysgo/engine"
)

// ServerConfig is sysgo's fixed-default oapi-codegen configuration (OPENAPI §7,
// C7/C9): a gin strict-server with models and a matching client, targeting
// OpenAPI 3.1. packageName names the generated Go package.
func ServerConfig(packageName string) codegen.Configuration {
	return codegen.Configuration{
		PackageName: packageName,
		Generate: codegen.GenerateOptions{
			GinServer: true,
			Strict:    true,
			Models:    true,
			Client:    true,
		},
		// Emit every component schema as a model even when no operation
		// references it yet (request/response body binding is a later slice).
		OutputOptions: codegen.OutputOptions{SkipPrune: true},
	}
}

// ModelsConfig is the models-only subset of ServerConfig — the generated data
// types with no server/client wiring (and so no web-framework dependency).
// Pruning is disabled so every component schema is emitted as a model even
// when no operation references it yet (request-body binding is a later slice).
func ModelsConfig(packageName string) codegen.Configuration {
	return codegen.Configuration{
		PackageName:   packageName,
		Generate:      codegen.GenerateOptions{Models: true},
		OutputOptions: codegen.OutputOptions{SkipPrune: true},
	}
}

// GenerateServer builds the OpenAPI 3.1 document from a resolved model and runs
// oapi-codegen in-process, returning the generated Go source (models +
// ServerInterface + gin wiring + client) under packageName.
func GenerateServer(m *engine.Model, packageName string) (string, error) {
	return generate(BuildDocument(m), ServerConfig(packageName))
}

// GenerateModels builds the document and generates only the data-type models —
// a dependency-free subset useful for tests and model-only consumers.
func GenerateModels(m *engine.Model, packageName string) (string, error) {
	return generate(BuildDocument(m), ModelsConfig(packageName))
}

// GenerateFromConfig runs oapi-codegen against a resolved model with a caller-
// supplied configuration, for callers that need a non-default output mix.
func GenerateFromConfig(m *engine.Model, cfg codegen.Configuration) (string, error) {
	return generate(BuildDocument(m), cfg)
}

// generate loads the document as an OpenAPI 3.1 spec, validates it, and invokes
// oapi-codegen with cfg.
func generate(doc *Document, cfg codegen.Configuration) (string, error) {
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromData([]byte(doc.YAML()))
	if err != nil {
		return "", fmt.Errorf("load openapi spec: %w", err)
	}
	if err := spec.Validate(loader.Context); err != nil {
		return "", fmt.Errorf("validate openapi spec: %w", err)
	}
	cfg = cfg.UpdateDefaults()
	if errs := cfg.Validate(); errs != nil {
		return "", fmt.Errorf("invalid codegen configuration: %w", errs)
	}
	out, err := codegen.Generate(spec, cfg)
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return out, nil
}
