// Package combined is a self-contained example that combines all three sysgo
// generators over one bounded context (model.sysml):
//
//   - the DDD core (domain, ports, PlaceOrder use case) under ./internal, from
//     model.json via `sysgo generate`;
//   - an OpenAPI entrypoint and a Temporal entrypoint (added in follow-ups) that
//     natively import and drive that DDD business logic.
//
// The DDD core is the business logic; OpenAPI and Temporal are just transports
// over it. Regenerate the DDD core after editing the model with:
//
//	go generate ./...
//
//go:generate go tool sysgo generate -c sysgo.yaml --out .
//go:generate go tool sysgo gen openapi model.sysml --out openapi.yaml
//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
//go:generate go tool sysgo gen temporal model.sysml --out orders --package orders
package combined
