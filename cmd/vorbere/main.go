package main

import (
	"os"

	"github.com/pirakansa/vorbere/internal/cli/commands"
)

var Version = "dev"

func main() {
	os.Exit(commands.Execute(Version))
}
