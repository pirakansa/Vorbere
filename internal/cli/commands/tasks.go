package commands

import (
	"fmt"

	"github.com/pirakansa/vorbere/internal/cli/taskrun"
	"github.com/spf13/cobra"
)

func newTasksCmd(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Task helpers",
	}
	cmd.AddCommand(newTasksListCmd(ctx))
	return cmd
}

func newTasksListCmd(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			taskCfg, _, err := loadTaskAndRoot(ctx.configPath)
			if err != nil {
				return err
			}
			for _, name := range taskrun.ListTaskNames(taskCfg) {
				desc := taskCfg.Tasks[name].Desc
				if desc == "" {
					fmt.Println(name)
					continue
				}
				fmt.Printf("%s\t%s\n", name, desc)
			}
			return nil
		},
	}
}
