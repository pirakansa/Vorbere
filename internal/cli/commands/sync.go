package commands

import (
	"fmt"
	"path/filepath"

	"github.com/pirakansa/vorbere/internal/cli/manifest"
	"github.com/pirakansa/vorbere/internal/cli/shared"
	"github.com/spf13/cobra"
)

func newSyncCmd(ctx *appContext) *cobra.Command {
	var mode string
	var backup string
	var dryRun bool
	var profile string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync files from manifest sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			taskCfg, rootDir, err := loadTaskAndRoot(ctx.configPath)
			if err != nil {
				return err
			}
			syncCfg, err := manifest.ResolveSyncConfig(taskCfg, ctx.configPath)
			if err != nil {
				return newExitCodeError(shared.ExitConfigError, err)
			}
			res, err := manifest.Sync(syncCfg, manifest.SyncOptions{
				RootDir:      rootDir,
				LockPath:     filepath.Join(rootDir, manifest.LockFileName),
				ModeOverride: mode,
				Backup:       backup,
				DryRun:       dryRun,
				Profile:      profile,
			})
			if res != nil {
				fmt.Printf("created=%d updated=%d unchanged=%d skipped=%d\n", res.Created, res.Updated, res.Unchanged, res.Skipped)
				for _, c := range res.Conflicts {
					fmt.Printf("conflict: %s\n", c)
				}
			}
			if err != nil {
				if err == manifest.ErrSyncConflict {
					return newExitCodeError(shared.ExitSyncConflict, err)
				}
				return newExitCodeError(shared.ExitSyncFailed, err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "", "merge mode override: three_way|overwrite|keep_local")
	cmd.Flags().StringVar(&backup, "backup", "", "backup strategy override: none|timestamp")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show actions without writing files")
	cmd.Flags().StringVar(&profile, "profile", "", "profile to append file rules")
	return cmd
}
