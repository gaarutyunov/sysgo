package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gaarutyunov/sysgo/engine"
	"github.com/gaarutyunov/sysgo/gen/contracts"
	"github.com/gaarutyunov/sysgo/gen/temporal"
)

// newGenCmd is the parent for the engine-based generators (gen/temporal,
// gen/contracts), which consume the engine's resolved model directly — distinct
// from the legacy `generate` scaffold pipeline.
func newGenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate code from a SysML model using the engine-based generators",
	}
	cmd.AddCommand(newGenTemporalCmd())
	cmd.AddCommand(newGenOpenAPICmd())
	return cmd
}

func newGenOpenAPICmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "openapi <model.sysml>",
		Short: "Generate an OpenAPI 3.1 document from a SysML model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenOpenAPI(cmd, args[0], out)
		},
	}
	cmd.Flags().StringVar(&out, "out", "openapi.yaml", "output OpenAPI document path")
	return cmd
}

// runGenOpenAPI loads the model and writes the OpenAPI document deterministically,
// so the same command drives both the example scaffolding and the drift check.
func runGenOpenAPI(cmd *cobra.Command, modelPath, out string) error {
	src, err := os.ReadFile(modelPath)
	if err != nil {
		return err
	}
	m := engine.New().AddFile(filepath.Base(modelPath), string(src)).Build()
	if diags := m.Diagnostics(); len(diags) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, "model %s has %d diagnostic(s):", modelPath, len(diags))
		for _, d := range diags {
			fmt.Fprintf(&b, "\n  %s", d.Message)
		}
		return fmt.Errorf("%s", b.String())
	}

	doc := contracts.BuildDocument(m)
	if dir := filepath.Dir(out); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(out, []byte(doc.YAML()), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "sysgo: wrote OpenAPI document to %s (%d schema(s))\n", out, len(doc.SchemaNames()))
	return nil
}

func newGenTemporalCmd() *cobra.Command {
	var out, pkg string
	cmd := &cobra.Command{
		Use:   "temporal <model.sysml>",
		Short: "Generate Temporal Go (activities, workflows, worker, schedules, codec) from a SysML model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenTemporal(cmd, args[0], out, pkg)
		},
	}
	cmd.Flags().StringVar(&out, "out", ".", "output directory")
	cmd.Flags().StringVar(&pkg, "package", "temporal", "generated Go package name")
	return cmd
}

// runGenTemporal loads the model and writes the Temporal generators' output
// deterministically, so the same command drives both project scaffolding and the
// drift check.
func runGenTemporal(cmd *cobra.Command, modelPath, out, pkg string) error {
	src, err := os.ReadFile(modelPath)
	if err != nil {
		return err
	}
	m := engine.New().AddFile(filepath.Base(modelPath), string(src)).Build()
	if diags := m.Diagnostics(); len(diags) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, "model %s has %d diagnostic(s):", modelPath, len(diags))
		for _, d := range diags {
			fmt.Fprintf(&b, "\n  %s", d.Message)
		}
		return fmt.Errorf("%s", b.String())
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}

	// Go artifacts. schedule/externalstorage/handlers are emitted only when the
	// model has the relevant construct (otherwise the generator returns a
	// header-only file, which we skip so the output stays clean and stable).
	gens := []struct {
		file string
		fn   func(*engine.Model, string) (string, error)
		req  string // only write when the code contains this token; "" = always
	}{
		{"activities.go", temporal.GenerateActivities, ""},
		{"workflows.go", temporal.GenerateWorkflows, ""},
		{"worker.go", temporal.GenerateWorker, ""},
		{"handlers.go", temporal.GenerateHandlers, "type "},
		{"schedule.go", temporal.GenerateSchedules, "func "},
		{"externalstorage.go", temporal.GenerateExternalStorage, "type "},
	}
	var written []string
	for _, g := range gens {
		code, err := g.fn(m, pkg)
		if err != nil {
			return fmt.Errorf("generate %s: %w", g.file, err)
		}
		if g.req != "" && !strings.Contains(code, g.req) {
			continue
		}
		if err := os.WriteFile(filepath.Join(out, g.file), []byte(code), 0o644); err != nil {
			return err
		}
		written = append(written, g.file)
	}

	// The determinism-check (workflowcheck) CI script.
	if err := os.WriteFile(filepath.Join(out, "workflowcheck.sh"), []byte(temporal.GenerateWorkflowcheck()), 0o755); err != nil {
		return err
	}
	written = append(written, "workflowcheck.sh")

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "sysgo: generated %d files into %s\n", len(written), out)
	for _, f := range written {
		fmt.Fprintln(w, "  + "+f)
	}
	return nil
}
