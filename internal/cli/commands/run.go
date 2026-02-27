package commands

import (
	"errors"

	"github.com/pirakansa/vorbere/internal/cli/shared"
	"github.com/pirakansa/vorbere/internal/cli/taskrun"
	"github.com/spf13/cobra"
)

func newRunCmd(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <task> [-- args...]",
		Short: "Run a common task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskCfg, rootDir, err := loadTaskAndRoot(ctx.configPath)
			if err != nil {
				return err
			}
			taskName := args[0]
			taskArgs := []string{}
			if len(args) > 1 {
				taskArgs = args[1:]
			}
			if _, ok := taskCfg.Tasks[taskName]; !ok {
				return newExitCodeError(shared.ExitTaskUndefined, errors.New("task is not defined"))
			}
			if err := taskrun.RunTask(taskCfg, taskName, rootDir, taskArgs); err != nil {
				return newExitCodeError(shared.ExitTaskFailed, err)
			}
			return nil
		},
	}
	return cmd
}
