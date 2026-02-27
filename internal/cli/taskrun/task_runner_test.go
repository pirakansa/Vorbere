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
