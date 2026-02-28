package commands

import (
	"github.com/spf13/cobra"
)

func newPlanCmd(ctx *appContext) *cobra.Command {
	opts := syncCommandOptions{dryRun: true}

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Preview sync changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSyncWithOptions(ctx, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.overwrite, "overwrite", false, "overwrite existing files without timestamp backup")
	return cmd
}
