package main

import (
	"os"

	"github.com/pirakansa/vorbere/internal/cli/commands"
)

func main() {
	os.Exit(commands.Execute())
}
