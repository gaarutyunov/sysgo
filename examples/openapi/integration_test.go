//go:build integration

// Package main's integration test builds the example API into a real Docker
// image, runs it in a container (no mocks, no shortcuts), issues real HTTP
// requests, and validates every request and response against the
// sysgo-generated OpenAPI document with kin-openapi's openapi3filter.
//
// The container binary is built with `go build -cover` (see Dockerfile) and
// writes coverage to GOCOVERDIR=/coverage, which is bind-mounted to a host
// directory. After the requests the container is stopped gracefully so the
// coverage runtime flushes, and the CI job turns the host directory into a real
// coverage profile with `go tool covdata`.
//
// Gated behind the `integration` build tag so the default `go test` (and the
// core CI) stays Docker-free.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/gaarutyunov/sysgo/examples/openapi/api"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

// coverageDir returns the host directory the container's coverage is written to:
// INTEGRATION_COVERDIR when set (the CI job reads it afterwards), otherwise a
// temp dir. It is created before the container starts so the bind mount works.
func coverageDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("INTEGRATION_COVERDIR")
	if dir == "" {
		dir = t.TempDir()
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("resolve coverage dir: %v", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		t.Fatalf("make coverage dir: %v", err)
	}
	return abs
}

// startAPI builds the example image from the repo root and runs it with
// /coverage bind-mounted to coverDir, returning the container and its base URL.
func startAPI(ctx context.Context, t *testing.T, coverDir string) (testcontainers.Container, string) {
	t.Helper()
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       repoRoot(t),
			Dockerfile:    "examples/openapi/Dockerfile",
			PrintBuildLog: true,
			KeepImage:     false,
		},
		ExposedPorts: []string{"8080/tcp"},
		HostConfigModifier: func(hc *dcontainer.HostConfig) {
			hc.Binds = append(hc.Binds, coverDir+":/coverage")
		},
		WaitingFor: wait.ForHTTP("/products/featured").
			WithPort("8080/tcp").
			WithStatusCodeMatcher(func(status int) bool { return status == http.StatusOK }).
			WithStartupTimeout(5 * time.Minute),
	}
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start api container: %v", err)
	}
	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	port, err := ctr.MappedPort(ctx, "8080/tcp")
	if err != nil {
		t.Fatalf("mapped port: %v", err)
	}
	return ctr, "http://" + host + ":" + port.Port()
}

// schemaRouter loads and validates the sysgo-generated document and returns a
// router for matching requests to operations.
func schemaRouter(ctx context.Context, t *testing.T) routers.Router {
	t.Helper()
	loader := openapi3.NewLoader()
	loader.Context = ctx
	doc, err := loader.LoadFromFile("openapi.yaml")
	if err != nil {
		t.Fatalf("load openapi.yaml: %v", err)
	}
	if err := doc.Validate(ctx); err != nil {
		t.Fatalf("generated document is not a valid OpenAPI spec: %v", err)
	}
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("build router: %v", err)
	}
	return router
}

// callValidated issues method+url (with optional JSON body), validating both the
// request and the response against the OpenAPI schema, and returns the response
// status and body bytes.
func callValidated(ctx context.Context, t *testing.T, router routers.Router, method, url string, body []byte) (int, []byte) {
	t.Helper()
	newReq := func() *http.Request {
		var r io.Reader
		if body != nil {
			r = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, r)
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return req
	}

	vreq := newReq()
	route, pathParams, err := router.FindRoute(vreq)
	if err != nil {
		t.Fatalf("no matching operation for %s %s: %v", method, url, err)
	}
	reqInput := &openapi3filter.RequestValidationInput{
		Request:    vreq,
		PathParams: pathParams,
		Route:      route,
	}
	if err := openapi3filter.ValidateRequest(ctx, reqInput); err != nil {
		t.Fatalf("request does not conform to schema: %v", err)
	}

	resp, err := http.DefaultClient.Do(newReq())
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	respInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: reqInput,
		Status:                 resp.StatusCode,
		Header:                 resp.Header,
	}
	respInput.SetBodyBytes(respBody)
	if err := openapi3filter.ValidateResponse(ctx, respInput); err != nil {
		t.Fatalf("response does not conform to schema (status %d): %v\nbody: %s", resp.StatusCode, err, respBody)
	}
	return resp.StatusCode, respBody
}

func TestOpenAPIAgainstRealContainer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	coverDir := coverageDir(t)
	ctr, baseURL := startAPI(ctx, t, coverDir)
	t.Cleanup(func() {
		if err := ctr.Terminate(context.Background()); err != nil {
			t.Logf("terminate container: %v", err)
		}
	})
	router := schemaRouter(ctx, t)

	// GET /products/featured — seeded product, schema-validated both ways.
	status, body := callValidated(ctx, t, router, http.MethodGet, baseURL+"/products/featured", nil)
	if status != http.StatusOK {
		t.Fatalf("GET featured: status = %d, want 200", status)
	}
	var got api.CatalogAPIProduct
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode featured: %v", err)
	}
	if got.Id != "p-1" || got.Name != "Widget" {
		t.Fatalf("featured product = %+v, want seeded p-1/Widget", got)
	}

	// POST /products — create, then confirm it becomes the featured product.
	created := api.CatalogAPIProduct{Id: "p-2", Name: "Gadget", Price: 19.5}
	payload, _ := json.Marshal(created)
	status, body = callValidated(ctx, t, router, http.MethodPost, baseURL+"/products", payload)
	if status != http.StatusCreated {
		t.Fatalf("POST products: status = %d, want 201", status)
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if got != created {
		t.Fatalf("created product = %+v, want %+v", got, created)
	}

	status, body = callValidated(ctx, t, router, http.MethodGet, baseURL+"/products/featured", nil)
	if status != http.StatusOK {
		t.Fatalf("GET featured after create: status = %d, want 200", status)
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode featured: %v", err)
	}
	if got != created {
		t.Fatalf("featured after create = %+v, want %+v", got, created)
	}

	// Stop the container gracefully so the -cover binary flushes coverage to the
	// bind-mounted host directory, then assert it actually produced data.
	stopTimeout := 30 * time.Second
	if err := ctr.Stop(ctx, &stopTimeout); err != nil {
		t.Fatalf("stop container: %v", err)
	}
	assertCoverageWritten(t, coverDir)
}

// assertCoverageWritten checks the container's -cover runtime flushed data to
// the bind-mounted host directory (covmeta + covcounters files).
func assertCoverageWritten(t *testing.T, coverDir string) {
	t.Helper()
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		t.Fatalf("read coverage dir: %v", err)
	}
	var covFiles int
	for _, e := range entries {
		if !e.IsDir() {
			covFiles++
		}
	}
	if covFiles == 0 {
		t.Fatal("no coverage files were produced by the container run")
	}
	t.Logf("container produced %d coverage files in %s", covFiles, coverDir)
}
