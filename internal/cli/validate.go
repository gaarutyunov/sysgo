package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/gaarutyunov/sysgo/internal/core/ir"
)

func newValidateCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Load the model and report mapping diagnostics; emit nothing",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(gf)
			if err != nil {
				return err
			}
			pl, mapper, err := buildPipeline(cfg)
			if err != nil {
				return err
			}
			proj, _, err := pl.LoadIR(cmd.Context())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Resolved IR for module %s\n", proj.Module)
			for _, c := range proj.Contexts {
				fmt.Fprintf(out, "\nContext %s (package %s)\n", c.Name, c.Package)
				printNamed(out, "Aggregates/Entities", entityNames(c.Entities))
				printNamed(out, "Value Objects", voNames(c.ValueObjects))
				printNamed(out, "Domain Events", eventNames(c.Events))
				printNamed(out, "Domain Services", serviceNames(c.DomainServices))
				printNamed(out, "Use Cases", useCaseNames(c.UseCases))
				printNamed(out, "Driving Ports", portNames(c.DrivingPorts))
				printNamed(out, "Driven Ports", portNames(c.DrivenPorts))
			}

			if len(mapper.Diagnostics) > 0 {
				fmt.Fprintln(out, "\nDiagnostics:")
				sort.Slice(mapper.Diagnostics, func(i, j int) bool {
					return mapper.Diagnostics[i].DeclaredName < mapper.Diagnostics[j].DeclaredName
				})
				for _, d := range mapper.Diagnostics {
					fmt.Fprintf(out, "  [%s] %s (%s): %s\n", d.Severity, d.DeclaredName, d.Rule, d.Message)
				}
			}
			return nil
		},
	}
}

func printNamed(out interface{ Write([]byte) (int, error) }, label string, names []string) {
	if len(names) == 0 {
		return
	}
	sort.Strings(names)
	fmt.Fprintf(out, "  %s: %v\n", label, names)
}

func entityNames(es []*ir.Entity) []string {
	out := make([]string, len(es))
	for i, e := range es {
		if e.Aggregate {
			out[i] = e.Name + " (aggregate)"
		} else {
			out[i] = e.Name
		}
	}
	return out
}

func voNames(vs []*ir.ValueObject) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.Name
	}
	return out
}

func eventNames(es []*ir.DomainEvent) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}

func serviceNames(ss []*ir.DomainService) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.Name
	}
	return out
}

func useCaseNames(us []*ir.UseCase) []string {
	out := make([]string, len(us))
	for i, u := range us {
		out[i] = u.Name
	}
	return out
}

func portNames(ps []*ir.Port) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Name
	}
	return out
}
