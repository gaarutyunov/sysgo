# examples/openapi

A self-contained, runnable example of the sysgo **OpenAPI** generator. It is a
separate Go module so the gin / oapi-codegen deps stay out of the core module's
build.

## Layout

| Path                 | Origin       | What it is                                                        |
|----------------------|--------------|-------------------------------------------------------------------|
| `model.sysml`        | hand-written | The SysML v2 model (a product-catalog REST API) annotated with the `RESTProfile` — the source of truth. |
| `openapi.yaml`       | generated    | `sysgo gen openapi` output — the OpenAPI 3.1 document. **Do not edit.** |
| `api/server.gen.go`  | generated    | oapi-codegen output from `openapi.yaml` — types + the gin `ServerInterface`. **Do not edit.** |
| `oapi-codegen.yaml`  | hand-written | oapi-codegen configuration.                                       |
| `handlers.go`        | hand-written | `Catalog` — the real implementation behind the generated `api.ServerInterface`. |
| `main.go`            | hand-written | Server entrypoint: wires the handlers into a gin router.          |

## Regenerating

`openapi.yaml` and `api/server.gen.go` are regenerated from `model.sysml` by the
in-repo sysgo and oapi-codegen (both wired as `tool` dependencies, sysgo pinned
to this checkout via a `replace`):

```sh
go generate ./...
```

The pipeline is `sysgo gen openapi` → `oapi-codegen`, and its output is
byte-for-byte deterministic, so a drift check is just
`go generate ./... && git diff --exit-code` (see issue #123).

## Building and running

```sh
go build ./...          # builds against gin + the oapi-codegen runtime

go run .                # serves the catalog API on :8080 (override with ADDR)
# GET  /products        -> list the catalog
# POST /products        -> create a product (JSON body matching the generated schema)
```

## Scope note

The model uses body-carrying operations (`GET /products`, `POST /products`).
Path-parameter operations (e.g. `GET /products/{id}`) are omitted because the
current generator does not yet emit OpenAPI `parameters` for path templates,
which oapi-codegen requires.
