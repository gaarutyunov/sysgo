# Builds the example API server WITH coverage instrumentation (`go build -cover`)
# so the integration test can extract real coverage from the running container
# via GOCOVERDIR (see integration_test.go).
#
# The build context is the repository root: the example module's
# `replace github.com/gaarutyunov/sysgo => ../..` needs the core module present.
FROM golang:1.26.5-bookworm AS build
WORKDIR /src
COPY . .
WORKDIR /src/examples/openapi
RUN CGO_ENABLED=0 go build -cover -o /out/server .

FROM debian:bookworm-slim
RUN mkdir -p /coverage
COPY --from=build /out/server /usr/local/bin/server
ENV GIN_MODE=release
ENV GOCOVERDIR=/coverage
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/server"]
