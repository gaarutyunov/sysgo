// Package combined is a self-contained example that combines all three sysgo
// generators over one bounded context (model.sysml), the way the framework
// intends: the business logic lives in a single DDD use case, and the REST and
// Temporal transports are *driving adapters* that both invoke it.
//
//   - the DDD core — domain, ports, the PlaceOrder use case, and the scaffolded
//     adapter layer — lives under ./internal, generated from model.json by
//     `sysgo generate` (generate.adapters: scaffold);
//   - the REST driving adapter (internal/order/adapter/in/http) implements the
//     generated api.ServerInterface and drives the PlaceOrder use case;
//   - the Temporal driving adapter (internal/order/adapter/in/temporal)
//     implements the generated orders.Activities port and drives the *same* use
//     case;
//   - cmd/api and cmd/worker are thin composition roots that wire the in-memory
//     repository into the use case and host the gin server / Temporal worker.
//
// The same PlaceOrder use case therefore runs whether an order arrives over HTTP
// or Temporal (see combined_test.go). Regenerate after editing the model with:
//
//	go generate ./...
//
//go:generate go tool sysgo generate -c sysgo.yaml --out .
//go:generate go tool sysgo gen openapi model.sysml --out openapi.yaml
//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
//go:generate go tool sysgo gen temporal model.sysml --out orders --package orders
package combined
