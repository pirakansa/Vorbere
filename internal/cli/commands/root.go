package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pirakansa/vorbere/internal/cli/manifest"
	"github.com/pirakansa/vorbere/internal/cli/shared"
	"github.com/spf13/cobra"
)

type appContext struct {
	configPath string
}

func NewRootCmd() *cobra.Command {
	ctx := &appContext{}
	cmd := &cobra.Command{
		Use:   "vorbere",
		Short: "Manifest-driven sync and common task runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringVar(&ctx.configPath, "config", "vorbere.yaml", "path to task config")

	cmd.AddCommand(newRunCmd(ctx))
	cmd.AddCommand(newSyncCmd(ctx))
	cmd.AddCommand(newTasksCmd(ctx))
	cmd.AddCommand(newPlanCmd(ctx))
	cmd.AddCommand(newInitCmd())

	return cmd
}

func Execute() int {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return mapExitCode(err)
	}
	return shared.ExitOK
}

func mapExitCode(err error) int {
	var codeErr *exitCodeError
	if errors.As(err, &codeErr) {
		return codeErr.code
	}
	return 1
}

func loadTaskAndRoot(configPath string) (*manifest.TaskConfig, string, error) {
	if manifest.IsRemoteConfigLocation(configPath) {
		taskCfg, err := manifest.LoadTaskConfig(configPath)
		if err != nil {
			return nil, "", newExitCodeError(shared.ExitConfigError, err)
		}
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", err
		}
		return taskCfg, cwd, nil
	}

	abs, err := filepath.Abs(configPath)
	if err != nil {
		return nil, "", err
	}
	taskCfg, err := manifest.LoadTaskConfig(abs)
	if err != nil {
		return nil, "", newExitCodeError(shared.ExitConfigError, err)
	}
	return taskCfg, filepath.Dir(abs), nil
}

type exitCodeError struct {
	code int
	err  error
}

func newExitCodeError(code int, err error) *exitCodeError {
	return &exitCodeError{code: code, err: err}
}

func (e *exitCodeError) Error() string {
	return e.err.Error()
}

func (e *exitCodeError) Unwrap() error {
	return e.err
}
