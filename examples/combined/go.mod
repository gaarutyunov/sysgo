module github.com/gaarutyunov/sysgo/examples/combined

go 1.26.5

// The example regenerates its DDD core, OpenAPI and Temporal code from the model
// with the in-repo sysgo (see doc.go's go:generate directives), so it always
// tracks the generators in this checkout rather than a published release.
replace github.com/gaarutyunov/sysgo => ../..

tool github.com/gaarutyunov/sysgo/cmd/sysgo

require (
	github.com/dave/jennifer v1.7.1 // indirect
	github.com/dprotaso/go-yit v0.0.0-20220510233725-9ba8df137936 // indirect
	github.com/gaarutyunov/sysgo v0.2.0 // indirect
	github.com/getkin/kin-openapi v0.142.0 // indirect
	github.com/go-openapi/jsonpointer v0.23.1 // indirect
	github.com/go-openapi/swag/jsonname v0.26.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/oapi-codegen/oapi-codegen/v2 v2.8.0 // indirect
	github.com/oasdiff/yaml v0.1.1 // indirect
	github.com/oasdiff/yaml3 v0.0.14 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/speakeasy-api/jsonpath v0.6.3 // indirect
	github.com/speakeasy-api/openapi v1.24.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
