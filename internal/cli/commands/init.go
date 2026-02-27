package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var syncRef string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create task.yaml and sync.yaml templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := writeIfNotExists("task.yaml", taskTemplate(syncRef)); err != nil {
				return err
			}
			if syncRef == "" {
				if err := writeIfNotExists("sync.yaml", syncTemplate()); err != nil {
					return err
				}
			}
			fmt.Println("initialized: task.yaml", ternary(syncRef == "", "and sync.yaml", ""))
			return nil
		},
	}
	cmd.Flags().StringVar(&syncRef, "with-sync-ref", "", "set sync.ref to local path or URL")
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

func taskTemplate(syncRef string) string {
	if syncRef == "" {
		return `version: v1
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
sync:
  ref: sync.yaml
`
	}
	return fmt.Sprintf(`version: v1
tasks:
  fmt:
    run: "echo define formatter command"
  lint:
    run: "echo define linter command"
  test:
    run: "echo define test command"
  build:
    run: "echo define build command"
  ci:
    depends_on: [fmt, lint, test, build]
sync:
  ref: %q
`, syncRef)
}

func syncTemplate() string {
	return `version: v1
sources:
  bootkit:
    type: http
    url: https://example.com/path/to/file
files:
  - source: bootkit
    path: .devcontainer/devcontainer.json
    merge: three_way
    backup: timestamp
`
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
