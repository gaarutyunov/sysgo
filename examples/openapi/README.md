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
| `main.go`            | hand-written | Server entrypoint: wires the handlers into a gin router (graceful shutdown). |
| `Dockerfile`         | hand-written | Builds the server with `go build -cover` for the integration test's real-container coverage. |
| `integration_test.go`| hand-written | testcontainers integration test (build tag `integration`) — see below. |

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
# POST /products            -> create a product (JSON body matching the generated schema); becomes the featured product
# GET  /products/featured   -> the featured (most recently created) product
```

## Integration test

`integration_test.go` (build tag `integration`) builds the `Dockerfile` into a
real image, runs the API in a container, issues real HTTP requests, and
validates every request and response against `openapi.yaml` with kin-openapi's
`openapi3filter`. The container binary is built with `go build -cover`, so real
coverage is collected from the running container (bind-mounted `GOCOVERDIR`,
flushed on graceful shutdown) and turned into a profile with `go tool covdata`.
It needs Docker and runs in the `examples-openapi` CI job:

```sh
mkdir -p covdata
INTEGRATION_COVERDIR=$PWD/covdata go test -tags integration ./...
go tool covdata textfmt -i=covdata -o=coverage.out
go tool cover -func=coverage.out | tail -1
```

## Scope note

Operations return a single `Product` each (`POST /products`,
`GET /products/featured`). The generator maps a directed parameter to that
type's object schema and does not model array multiplicity, so single-object
operations keep the generated schema and the handlers in agreement — which the
integration test enforces. Path-parameter operations (e.g. `GET /products/{id}`)
are also omitted: the generator does not yet emit OpenAPI `parameters` for path
templates, which oapi-codegen requires.
