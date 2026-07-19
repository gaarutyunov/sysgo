//go:build integration

// Package main's integration test spins up a real Temporal server in a
// container (no mocks, no shortcuts), runs the sysgo-generated worker with the
// hand-written activity implementations, executes the generated workflow via
// the real Temporal client, and asserts the server reports completion.
//
// It is gated behind the `integration` build tag so the default `go test` (and
// the core CI) stays Docker-free; the examples-temporal CI job runs it with
// `-tags integration` and collects coverage over the example packages.
package main

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"

	"github.com/gaarutyunov/sysgo/examples/temporal/orders"
)

const (
	// The Temporal CLI image; `server start-dev` runs a full single-binary dev
	// server (in-memory persistence, frontend gRPC on 7233) — a real Temporal
	// deployment, not a mock.
	temporalImage = "temporalio/temporal:1.8.0"
	taskQueue     = "orders"
)

// startTemporal launches a real Temporal dev server in a container and returns
// its frontend host:port. The container is terminated at test cleanup.
func startTemporal(ctx context.Context, t *testing.T) string {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        temporalImage,
		ExposedPorts: []string{"7233/tcp"},
		Cmd:          []string{"server", "start-dev", "--ip", "0.0.0.0", "--log-level", "error"},
		WaitingFor:   wait.ForListeningPort("7233/tcp").WithStartupTimeout(3 * time.Minute),
	}
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start temporal container: %v", err)
	}
	t.Cleanup(func() {
		if err := ctr.Terminate(context.Background()); err != nil {
			t.Logf("terminate container: %v", err)
		}
	})

	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	port, err := ctr.MappedPort(ctx, "7233/tcp")
	if err != nil {
		t.Fatalf("mapped port: %v", err)
	}
	return net.JoinHostPort(host, port.Port())
}

// dialWithRetry dials the frontend, retrying until the server is serving.
func dialWithRetry(t *testing.T, hostPort string) client.Client {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for {
		c, err := client.Dial(client.Options{HostPort: hostPort})
		if err == nil {
			return c
		}
		if time.Now().After(deadline) {
			t.Fatalf("dial temporal at %s: %v", hostPort, err)
		}
		time.Sleep(time.Second)
	}
}

// executeWithRetry starts the generated workflow, retrying until the default
// namespace is registered and the frontend accepts the request.
func executeWithRetry(ctx context.Context, t *testing.T, c client.Client) client.WorkflowRun {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for {
		run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
			TaskQueue: taskQueue,
		}, "ProcessOrderWorkflow", orders.Order{Id: "order-123"})
		if err == nil {
			return run
		}
		if time.Now().After(deadline) {
			t.Fatalf("execute workflow: %v", err)
		}
		time.Sleep(time.Second)
	}
}

// TestProcessOrderWorkflowAgainstRealTemporal is the end-to-end integration
// test: real server, generated worker, generated workflow, hand-written
// activities, real client.
func TestProcessOrderWorkflowAgainstRealTemporal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	hostPort := startTemporal(ctx, t)

	c := dialWithRetry(t, hostPort)
	defer c.Close()

	// Run the generated worker (RunWorker) with the hand-written activities.
	// RunWorker blocks on the interrupt channel, so drive it in a goroutine and
	// stop it after the workflow completes.
	workerErr := make(chan error, 1)
	go func() { workerErr <- orders.RunWorker(c, OrderActivities{}) }()

	// Execute the generated workflow through the real client and wait for it.
	run := executeWithRetry(ctx, t, c)
	if err := run.Get(ctx, nil); err != nil {
		t.Fatalf("workflow did not complete: %v", err)
	}

	// Assert the server itself reports a completed execution — a real,
	// server-side assertion, not just a local return value.
	desc, err := c.DescribeWorkflowExecution(ctx, run.GetID(), run.GetRunID())
	if err != nil {
		t.Fatalf("describe workflow execution: %v", err)
	}
	if got := desc.GetWorkflowExecutionInfo().GetStatus(); got != enums.WORKFLOW_EXECUTION_STATUS_COMPLETED {
		t.Fatalf("workflow status = %v, want COMPLETED", got)
	}

	// Stop RunWorker: deliver an interrupt to this process. worker.InterruptCh
	// registers for os.Interrupt via signal.Notify, so this is caught (the
	// process is not terminated) and RunWorker returns after stopping workers.
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("find process: %v", err)
	}
	if err := p.Signal(os.Interrupt); err != nil {
		t.Fatalf("signal interrupt: %v", err)
	}
	select {
	case err := <-workerErr:
		if err != nil {
			t.Fatalf("RunWorker returned error: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("RunWorker did not stop after interrupt")
	}
}
