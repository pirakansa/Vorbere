package commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pirakansa/vorbere/internal/cli/manifest"
	"github.com/pirakansa/vorbere/internal/cli/shared"
)

func TestMapExitCode(t *testing.T) {
	if got := mapExitCode(newExitCodeError(shared.ExitTaskFailed, errors.New("x"))); got != shared.ExitTaskFailed {
		t.Fatalf("expected %d got %d", shared.ExitTaskFailed, got)
	}
	if got := mapExitCode(manifest.ErrSyncConflict); got != shared.ExitSyncConflict {
		t.Fatalf("expected %d got %d", shared.ExitSyncConflict, got)
	}
	if got := mapExitCode(errors.New("other")); got != 1 {
		t.Fatalf("expected 1 got %d", got)
	}
}

func TestInitCommandCreatesFilesAndFailsOnSecondRun(t *testing.T) {
	temp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(temp); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}

	cmd := newInitCmd()
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(temp, "task.yaml")); err != nil {
		t.Fatalf("task.yaml missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(temp, "sync.yaml")); err != nil {
		t.Fatalf("sync.yaml missing: %v", err)
	}

	cmd = newInitCmd()
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected second init to fail when files already exist")
	}
}

func TestInitWithSyncRefCreatesOnlyTaskYAML(t *testing.T) {
	temp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(temp); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}

	cmd := newInitCmd()
	cmd.SetArgs([]string{"--with-sync-ref", "https://example.com/sync.yaml"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init with sync ref failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(temp, "task.yaml")); err != nil {
		t.Fatalf("task.yaml missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(temp, "sync.yaml")); !os.IsNotExist(err) {
		t.Fatalf("sync.yaml should not be created when --with-sync-ref is set, err=%v", err)
	}
}

func TestRunCommandReturnsDefinedExitCodes(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "task.yaml")
	cfg := `version: v1
tasks:
  ok:
    run: "echo ok"
  fail:
    run: "false"
`
	if err := os.WriteFile(configPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write task.yaml failed: %v", err)
	}

	ctx := &appContext{configPath: configPath}

	cmd := newRunCmd(ctx)
	cmd.SetArgs([]string{"missing"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected undefined task error")
	}
	var exitErr *exitCodeError
	if !errors.As(err, &exitErr) || exitErr.code != shared.ExitTaskUndefined {
		t.Fatalf("expected ExitTaskUndefined, err=%v", err)
	}

	cmd = newRunCmd(ctx)
	cmd.SetArgs([]string{"fail"})
	err = cmd.Execute()
	if err == nil {
		t.Fatalf("expected failed task error")
	}
	if !errors.As(err, &exitErr) || exitErr.code != shared.ExitTaskFailed {
		t.Fatalf("expected ExitTaskFailed, err=%v", err)
	}
}
