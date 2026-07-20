// Command openapi is a self-contained, runnable example of the sysgo OpenAPI
// generator. `sysgo gen openapi --server` turns the SysML model in model.sysml
// directly into the gin server + types under ./api (oapi-codegen runs in-process
// against an in-memory openapi3.T — there is no openapi.yaml to keep in sync).
// The committed openapi.yaml is emitted only as a reference document and is used
// by the integration test to validate requests/responses. This package supplies
// the hand-written handlers behind the generated ServerInterface and a server
// main.
//
// Regenerate everything after editing model.sysml with:
//
//	go generate ./...
//
//go:generate go tool sysgo gen openapi model.sysml --server --out api/server.gen.go --package api
//go:generate go tool sysgo gen openapi model.sysml --out openapi.yaml
package main
