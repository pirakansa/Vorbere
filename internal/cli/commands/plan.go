package commands

import (
	"github.com/spf13/cobra"
)

func newPlanCmd(ctx *appContext) *cobra.Command {
	var mode string
	var backup string
	var profile string

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Preview sync changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			argsForSync := []string{"--dry-run"}
			if mode != "" {
				argsForSync = append(argsForSync, "--mode", mode)
			}
			if backup != "" {
				argsForSync = append(argsForSync, "--backup", backup)
			}
			if profile != "" {
				argsForSync = append(argsForSync, "--profile", profile)
			}
			syncCmd := newSyncCmd(ctx)
			syncCmd.SetArgs(argsForSync)
			return syncCmd.Execute()
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "", "merge mode override: three_way|overwrite|keep_local")
	cmd.Flags().StringVar(&backup, "backup", "", "backup strategy override: none|timestamp")
	cmd.Flags().StringVar(&profile, "profile", "", "profile to append file rules")
	return cmd
}
