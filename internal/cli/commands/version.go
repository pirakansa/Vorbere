package commands

import (
	"fmt"
	"runtime/debug"
	"strings"

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
		return normalizeVersion(version)
	}

	info, ok := readBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return normalizeVersion(info.Main.Version)
	}
	return defaultVersionValue
}

func normalizeVersion(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}
