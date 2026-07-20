# examples/combined — one use case, REST + Temporal driving adapters

This example combines all three sysgo generators over a single bounded context
(`model.sysml`, the Order context), the way the framework intends:

> The business logic is a **DDD use case**. **REST** and **Temporal** are not
> separate programs — they are **driving adapters** (`adapter/in/…`) that both
> invoke the *same* use case. Whether an order arrives over HTTP or is processed
> by a Temporal workflow, the identical `PlaceOrder` code runs.

## The hexagon

```text
        HTTP request                     Temporal activity
             │                                   │
  adapter/in/http (REST driving adapter)  adapter/in/temporal (Temporal driving adapter)
             │        implements api.        │   implements orders.Activities
             │        ServerInterface        │
             └───────────────┬───────────────┘
                             ▼
                app/port/in: PlaceOrderUseCase        (driving port)
                app/usecase: PlaceOrderInteractor      (business logic)
                             │
                app/port/out: OrderRepository          (driven port)
                             ▼
                adapter/out/repository (in-memory)      (driven adapter)
```

| Path | What | Source |
|---|---|---|
| `internal/order/domain`, `internal/order/app` | domain, ports, and the `PlaceOrder` use case | `sysgo generate` (from `model.json`) |
| `internal/order/adapter/in/http` | **REST driving adapter** — implements `api.ServerInterface`, drives the use case | scaffolded by `sysgo generate`, filled in |
| `internal/order/adapter/in/temporal` | **Temporal driving adapter** — implements `orders.Activities`, drives the *same* use case | hand-written |
| `internal/order/adapter/out/repository` | in-memory `OrderRepository` driven adapter | scaffolded by `sysgo generate`, filled in |
| `api/` | generated gin `ServerInterface` + models | `sysgo gen openapi` + oapi-codegen |
| `orders/` | generated workflow / activities / worker | `sysgo gen temporal` |
| `cmd/api`, `cmd/worker` | thin composition roots that wire repo → use case → adapter and host the server / worker | hand-written |

`combined_test.go` drives the **same** `PlaceOrder` use case (over one
repository) from **both** the REST adapter and the Temporal adapter, asserting
both orders land in the same store — the whole point of the example.

## Model

`model.sysml` is the single source: the Order domain plus `@REST` (OpenAPI) and
`@Workflow`/`@Activity` (Temporal) annotations. The DDD core is generated from
`model.json` (the same Order domain as `examples/order`; the annotations are
additive metadata), which keeps the DDD path free of the SysML-to-JSON toolchain.

## Regenerate

```bash
go generate ./...   # DDD core (sysgo generate), OpenAPI doc + gin server
                    # (sysgo gen openapi + oapi-codegen), Temporal worker code
                    # (sysgo gen temporal). Scaffolded adapters are never clobbered.
```
