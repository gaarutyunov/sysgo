package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newGenerateCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Run the pipeline and emit the Go scaffold",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(gf)
			if err != nil {
				return err
			}
			pl, _, err := buildPipeline(cfg)
			if err != nil {
				return err
			}
			res, err := pl.Generate(cmd.Context(), gf.out)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "sysgo: wrote %d files, skipped %d scaffold files, pruned %d stale files\n",
				len(res.Written), len(res.Skipped), len(res.Pruned))
			for _, f := range res.Written {
				fmt.Fprintln(out, "  + "+f)
			}
			for _, f := range res.Skipped {
				fmt.Fprintln(out, "  = "+f+" (scaffold, kept)")
			}
			for _, f := range res.Pruned {
				fmt.Fprintln(out, "  - "+f+" (pruned)")
			}
			return nil
		},
	}
}
