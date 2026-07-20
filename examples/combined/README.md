# examples/combined — DDD + OpenAPI + Temporal over one model

This example combines all three sysgo generators over a single bounded context
(`model.sysml`, the Order context), demonstrating the framework's core idea:

> The business logic is written in **DDD** form (use cases, ports, adapters).
> **OpenAPI** and **Temporal** are just ways to generate *entrypoints* that
> natively import and drive that business logic.

## Layout

| Path | What | Generator |
|---|---|---|
| `internal/order/domain`, `internal/order/app` | the **DDD core** — domain, ports, and the `PlaceOrder` use case | `sysgo generate` (from `model.json`) |
| `cmd/api` | REST entrypoint whose handlers call the `PlaceOrder` use case | `sysgo gen openapi` + oapi-codegen |
| `cmd/worker` *(follow-up)* | Temporal worker whose activities call the use case | `sysgo gen temporal` |

## Model

`model.sysml` is the single source of truth for the entrypoint generators: the
Order domain plus `@REST` (OpenAPI) and `@Workflow`/`@Activity` (Temporal)
annotations. The DDD core is generated from `model.json` — the same Order domain
as `examples/order` (the annotations are additive metadata and don't change the
DDD structure), which keeps the DDD path free of the SysML-to-JSON toolchain.

## Regenerate

```bash
go generate ./...   # regenerates the DDD core (sysgo generate), the OpenAPI doc
                    # (sysgo gen openapi) and the gin server (oapi-codegen)
```
