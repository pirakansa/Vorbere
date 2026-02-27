package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create vorbere.yaml template",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := writeIfNotExists("vorbere.yaml", taskTemplate()); err != nil {
				return err
			}
			fmt.Println("initialized: vorbere.yaml")
			return nil
		},
	}
	return cmd
}

func writeIfNotExists(path, content string) error {
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func taskTemplate() string {
	return `version: 3
tasks:
  fmt:
    run: "echo define formatter command"
    desc: format code
  lint:
    run: "echo define linter command"
    desc: lint code
  test:
    run: "echo define test command"
    desc: run tests
  build:
    run: "echo define build command"
    desc: build artifacts
  ci:
    depends_on: [fmt, lint, test, build]
repositories:
  - _comment: example repository
    url: https://example.com/
    files:
      - file_name: path/to/file
        out_dir: .
        rename: file
        x_vorbere:
          merge: three_way
          backup: timestamp
`
}
