// Command openapi is a self-contained, runnable example of the sysgo OpenAPI
// generator. The SysML model in model.sysml is turned into an OpenAPI 3.1
// document (openapi.yaml) by `sysgo gen openapi`, and oapi-codegen turns that
// document into the gin server + types under ./api. This package supplies the
// hand-written handlers behind the generated ServerInterface and a server main.
//
// Regenerate everything after editing model.sysml with:
//
//	go generate ./...
//
//go:generate go tool sysgo gen openapi model.sysml --out openapi.yaml
//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
package main
