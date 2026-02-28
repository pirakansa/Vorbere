package commands

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

const defaultVersionValue = "dev"

var readBuildInfo = debug.ReadBuildInfo

func newVersionCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show CLI version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), resolveVersion(version))
		},
	}
	return cmd
}

func resolveVersion(version string) string {
	if version != "" && version != defaultVersionValue {
		return version
	}

	info, ok := readBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return defaultVersionValue
}
