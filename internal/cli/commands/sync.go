package commands

import (
	"fmt"

	"github.com/pirakansa/vorbere/internal/cli/manifest"
	"github.com/pirakansa/vorbere/internal/cli/shared"
	"github.com/spf13/cobra"
)

type syncCommandOptions struct {
	overwrite bool
	dryRun    bool
}

func newSyncCmd(ctx *appContext) *cobra.Command {
	opts := &syncCommandOptions{}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync files from manifest sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSyncWithOptions(ctx, *opts)
		},
	}
	cmd.Flags().BoolVar(&opts.overwrite, "overwrite", false, "overwrite existing files without timestamp backup")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "show actions without writing files")
	return cmd
}

func runSyncWithOptions(ctx *appContext, opts syncCommandOptions) error {
	taskCfg, rootDir, err := loadTaskAndRoot(ctx.configPath)
	if err != nil {
		return err
	}
	syncCfg, err := manifest.ResolveSyncConfig(taskCfg, ctx.configPath)
	if err != nil {
		return newExitCodeError(shared.ExitConfigError, err)
	}

	res, err := manifest.Sync(syncCfg, manifest.SyncOptions{
		RootDir:   rootDir,
		Overwrite: opts.overwrite,
		DryRun:    opts.dryRun,
	})
	printSyncResult(res)
	if err != nil {
		return newExitCodeError(shared.ExitSyncFailed, err)
	}
	return nil
}

func printSyncResult(res *manifest.SyncResult) {
	if res == nil {
		return
	}
	fmt.Printf("created=%d updated=%d unchanged=%d skipped=%d\n", res.Created, res.Updated, res.Unchanged, res.Skipped)
}
