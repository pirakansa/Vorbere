package taskrun

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pirakansa/vorbere/internal/cli/manifest"
)

func TestRunTaskDependsOnOrder(t *testing.T) {
	temp := t.TempDir()
	marker := filepath.Join(temp, "result.txt")

	cfg := &manifest.TaskConfig{
		Version: "v1",
		Tasks: map[string]manifest.TaskDef{
			"fmt":  {Run: "echo fmt > result.txt"},
			"ci":   {DependsOn: []string{"fmt", "test"}},
			"test": {Run: "echo test >> result.txt"},
		},
	}

	if err := RunTask(cfg, "ci", temp, nil); err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}
	b, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if string(b) != "fmt\ntest\n" {
		t.Fatalf("unexpected run output: %q", string(b))
	}
}

func TestRunTaskAppliesEnvAndCWD(t *testing.T) {
	temp := t.TempDir()
	workDir := filepath.Join(temp, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir work dir: %v", err)
	}

	cfg := &manifest.TaskConfig{
		Version: "v1",
		Tasks: map[string]manifest.TaskDef{
			"envcwd": {
				Run: `echo "${MY_VALUE}" > output.txt`,
				Env: map[string]string{
					"MY_VALUE": "from-env",
				},
				CWD: "work",
			},
		},
	}

	if err := RunTask(cfg, "envcwd", temp, nil); err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(workDir, "output.txt"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(b) != "from-env\n" {
		t.Fatalf("unexpected output: %q", string(b))
	}
}

func TestRunTaskDetectsDependencyCycle(t *testing.T) {
	temp := t.TempDir()
	cfg := &manifest.TaskConfig{
		Version: "v1",
		Tasks: map[string]manifest.TaskDef{
			"a": {DependsOn: []string{"b"}},
			"b": {DependsOn: []string{"a"}},
		},
	}

	if err := RunTask(cfg, "a", temp, nil); err == nil {
		t.Fatalf("expected dependency cycle error")
	}
}

func TestRunTaskAppendsArgsToCommand(t *testing.T) {
	temp := t.TempDir()
	marker := filepath.Join(temp, "args.txt")
	cfg := &manifest.TaskConfig{
		Version: "v1",
		Tasks: map[string]manifest.TaskDef{
			"args": {Run: `echo > args.txt`},
		},
	}

	if err := RunTask(cfg, "args", temp, []string{"hello", "world"}); err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}
	b, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := string(b); got != "hello world\n" {
		t.Fatalf("unexpected args output: %q", got)
	}
}
