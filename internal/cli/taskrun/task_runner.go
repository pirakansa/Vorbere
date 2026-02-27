package taskrun

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pirakansa/vorbere/internal/cli/manifest"
)

func ListTaskNames(cfg *manifest.TaskConfig) []string {
	names := make([]string, 0, len(cfg.Tasks))
	for name := range cfg.Tasks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func RunTask(cfg *manifest.TaskConfig, name, rootDir string, args []string) error {
	running := map[string]bool{}
	completed := map[string]bool{}
	return runTask(cfg, name, rootDir, args, running, completed)
}

func runTask(cfg *manifest.TaskConfig, name, rootDir string, args []string, running, completed map[string]bool) error {
	if completed[name] {
		return nil
	}
	task, ok := cfg.Tasks[name]
	if !ok {
		return fmt.Errorf("task %q is not defined", name)
	}
	if running[name] {
		return fmt.Errorf("task dependency cycle detected at %q", name)
	}
	running[name] = true
	for _, dep := range task.DependsOn {
		if err := runTask(cfg, dep, rootDir, nil, running, completed); err != nil {
			return err
		}
	}
	running[name] = false

	if strings.TrimSpace(task.Run) != "" {
		cmdLine := task.Run
		if len(args) > 0 {
			cmdLine += " " + strings.Join(args, " ")
		}
		cmd := exec.Command("bash", "-lc", cmdLine)
		cwd := rootDir
		if task.CWD != "" {
			if filepath.IsAbs(task.CWD) {
				cwd = task.CWD
			} else {
				cwd = filepath.Join(rootDir, task.CWD)
			}
		}
		cmd.Dir = cwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = os.Environ()
		for k, v := range task.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("task %q failed: %w", name, err)
		}
	}
	completed[name] = true
	return nil
}
