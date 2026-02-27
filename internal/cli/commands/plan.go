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
	cmd.Flags().StringVar(&opts.mode, "mode", "", "merge mode override: three_way|overwrite|keep_local")
	cmd.Flags().StringVar(&opts.backup, "backup", "", "backup strategy override: none|timestamp")
	cmd.Flags().StringVar(&opts.profile, "profile", "", "profile to append file rules")
	return cmd
}
