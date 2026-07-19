# examples/temporal

A self-contained, runnable example of the sysgo **Temporal** generator. It is a
separate Go module so the Temporal SDK stays out of the core module's build.

## Layout

| Path                 | Origin       | What it is                                                        |
|----------------------|--------------|-------------------------------------------------------------------|
| `model.sysml`        | hand-written | The SysML v2 model (order-fulfilment workflow) — the source of truth. |
| `orders/*.go`        | generated    | `sysgo gen temporal` output — the Activities port, workflow, worker. **Do not edit.** |
| `orders/workflowcheck.sh` | generated | Determinism-check script emitted alongside the Go code.       |
| `activities.go`      | hand-written | `OrderActivities` — the real implementation behind the generated `orders.Activities` port. |
| `main.go`            | hand-written | Worker entrypoint: dials Temporal and runs the generated worker.  |

## Regenerating

The `orders/` package is regenerated from `model.sysml` by the in-repo sysgo
(wired as a `tool` dependency, pinned to this checkout via a `replace`):

```sh
go generate ./...
```

Regeneration is byte-for-byte deterministic, so a drift check is just
`go generate ./... && git diff --exit-code` (see issue #115).

## Building and running

```sh
go build ./...                       # builds against the real Temporal SDK

# With a Temporal dev server on 127.0.0.1:7233 (override with TEMPORAL_HOSTPORT):
go run .
```

The worker registers `ProcessOrderWorkflow` and the `ChargeCard` / `SendReceipt`
activities on the `orders` task queue and blocks until interrupted.
