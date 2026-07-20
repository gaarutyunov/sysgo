// Package cli implements the sysgo command-line interface (cobra), the
// composition root that wires concrete adapters into the application pipeline.
package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gaarutyunov/sysgo/internal/adapter/gotmpl"
	"github.com/gaarutyunov/sysgo/internal/adapter/openapi"
	"github.com/gaarutyunov/sysgo/internal/adapter/osfs"
	"github.com/gaarutyunov/sysgo/internal/adapter/overlay"
	"github.com/gaarutyunov/sysgo/internal/adapter/sysmlapi"
	"github.com/gaarutyunov/sysgo/internal/adapter/sysmlfile"
	"github.com/gaarutyunov/sysgo/internal/app"
	"github.com/gaarutyunov/sysgo/internal/app/port"
	"github.com/gaarutyunov/sysgo/internal/config"
	"github.com/gaarutyunov/sysgo/internal/core/mapping"
)

// Version is set via -ldflags at release time.
var Version = "dev"

type globalFlags struct {
	configPath string
	module     string
	overlay    string
	templates  string
	out        string
}

// NewRootCmd builds the root command tree.
func NewRootCmd() *cobra.Command {
	gf := &globalFlags{}
	root := &cobra.Command{
		Use:           "sysgo",
		Short:         "Generate a DDD/hexagonal Go scaffold from a SysML v2 model",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	pf := root.PersistentFlags()
	pf.StringVarP(&gf.configPath, "config", "c", "sysgo.yaml", "path to sysgo.yaml")
	pf.StringVar(&gf.module, "module", "", "override module path")
	pf.StringVar(&gf.overlay, "overlay", "", "override overlay path")
	pf.StringVar(&gf.templates, "templates", "", "directory of user template overrides")
	pf.StringVar(&gf.out, "out", ".", "output root directory")

	root.AddCommand(newInitCmd())
	root.AddCommand(newGenerateCmd(gf))
	root.AddCommand(newGenCmd())
	root.AddCommand(newValidateCmd(gf))
	root.AddCommand(newVersionCmd())
	return root
}

// Execute runs the CLI.
func Execute() int {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "sysgo: "+err.Error())
		return 1
	}
	return 0
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the sysgo version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "sysgo "+Version)
			return nil
		},
	}
}

// loadConfig loads and overlays flag values onto the configuration.
func loadConfig(gf *globalFlags) (*config.Config, error) {
	cfg, err := config.Load(gf.configPath)
	if err != nil {
		return nil, err
	}
	// Resolve model/overlay paths relative to the config file's directory.
	base := filepath.Dir(gf.configPath)
	cfg.Source.File = resolveRel(base, cfg.Source.File)
	cfg.Source.SysML = resolveRel(base, cfg.Source.SysML)
	cfg.Overlay.Path = resolveRel(base, cfg.Overlay.Path)

	if gf.module != "" {
		cfg.Module = gf.module
	}
	if gf.overlay != "" {
		cfg.Overlay.Path = gf.overlay
	}
	if gf.templates != "" {
		if err := loadTemplateDir(cfg, gf.templates); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// resolveRel resolves a possibly-relative path against base (the config dir).
func resolveRel(base, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}

// loadTemplateDir registers every *.tmpl under dir as a same-name override.
func loadTemplateDir(cfg *config.Config, dir string) error {
	if cfg.OutputOptions.UserTemplates == nil {
		cfg.OutputOptions.UserTemplates = map[string]string{}
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		cfg.OutputOptions.UserTemplates[filepath.ToSlash(rel)] = path
		return nil
	})
}

// buildPipeline wires adapters into the pipeline from configuration.
func buildPipeline(cfg *config.Config) (*app.Pipeline, *mapping.Mapper, error) {
	loader, err := newLoader(cfg)
	if err != nil {
		return nil, nil, err
	}
	var ov port.OverlayEngine
	if cfg.Overlay.Path != "" {
		eng, err := overlay.Load(cfg.Overlay.Path)
		if err != nil {
			return nil, nil, err
		}
		ov = eng
	}
	mapper := mapping.New(cfg)
	renderer, err := gotmpl.New(cfg)
	if err != nil {
		return nil, nil, err
	}
	writer := osfs.New(cfg.OutputOptions.SkipFmt, cfg.OutputOptions.SkipPrune, cfg.OutputOptions.GeneratedMarker)

	var contractsEmitter port.ContractEmitter
	if cfg.Generate.OpenAPI {
		if cfg.Source.SysML == "" {
			return nil, nil, fmt.Errorf("generate.openapi requires source.sysml (path to the .sysml textual source)")
		}
		contractsEmitter = openapi.New(cfg.Source.SysML)
	}

	return &app.Pipeline{
		Loader:    loader,
		Overlay:   ov,
		Builder:   mapper,
		Renderer:  renderer,
		Writer:    writer,
		Contracts: contractsEmitter,
	}, mapper, nil
}

func newLoader(cfg *config.Config) (port.ModelLoader, error) {
	switch {
	case cfg.Source.File != "":
		return sysmlfile.New(cfg.Source.File), nil
	case cfg.Source.API != nil:
		return sysmlapi.New(cfg.Source.API.BaseURL, cfg.Source.API.Project, cfg.Source.API.Commit), nil
	default:
		return nil, fmt.Errorf("no source configured (set source.file or source.api)")
	}
}
