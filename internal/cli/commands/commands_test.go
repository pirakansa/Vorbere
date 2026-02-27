package commands

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	if _, err := os.Stat(filepath.Join(temp, "vorbere.yaml")); err != nil {
		t.Fatalf("vorbere.yaml missing: %v", err)
	}

	cmd = newInitCmd()
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected second init to fail when files already exist")
	}
}

func TestInitTemplateContainsRepositories(t *testing.T) {
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
		t.Fatalf("init failed: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(temp, "vorbere.yaml"))
	if err != nil {
		t.Fatalf("vorbere.yaml missing: %v", err)
	}
	if !containsAll(string(b), []string{"version: 3", "repositories:", "file_name:"}) {
		t.Fatalf("unexpected template content:\n%s", string(b))
	}
}

func TestRunCommandReturnsDefinedExitCodes(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "vorbere.yaml")
	cfg := `version: 3
tasks:
  ok:
    run: "echo ok"
  fail:
    run: "false"
`
	if err := os.WriteFile(configPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write vorbere.yaml failed: %v", err)
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

func TestRunCommandReturnsConfigErrorWhenTaskConfigMissing(t *testing.T) {
	ctx := &appContext{configPath: filepath.Join(t.TempDir(), "missing-vorbere.yaml")}

	cmd := newRunCmd(ctx)
	cmd.SetArgs([]string{"test"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected config error")
	}
	var exitErr *exitCodeError
	if !errors.As(err, &exitErr) || exitErr.code != shared.ExitConfigError {
		t.Fatalf("expected ExitConfigError, err=%v", err)
	}
}

func TestLoadTaskAndRootUsesCWDForRemoteConfig(t *testing.T) {
	temp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(temp); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("version: 3\ntasks:\n  test:\n    run: \"echo ok\"\n"))
	}))
	defer server.Close()

	taskCfg, rootDir, err := loadTaskAndRoot(server.URL)
	if err != nil {
		t.Fatalf("loadTaskAndRoot returned error: %v", err)
	}
	if taskCfg.Tasks["test"].Run == "" {
		t.Fatalf("expected task to be loaded from remote config")
	}
	if rootDir != temp {
		t.Fatalf("expected rootDir=%s, got=%s", temp, rootDir)
	}
}

func TestSyncCommandReturnsSyncFailedForInvalidMode(t *testing.T) {
	temp := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	taskBody := `version: 3
repositories:
  - url: ` + server.URL + `
    files:
      - file_name: "/"
        out_dir: .
        rename: a.txt
`
	taskPath := filepath.Join(temp, "vorbere.yaml")
	if err := os.WriteFile(taskPath, []byte(taskBody), 0o644); err != nil {
		t.Fatalf("write task config: %v", err)
	}

	ctx := &appContext{configPath: taskPath}
	err := runSyncWithOptions(ctx, syncCommandOptions{mode: "invalid-mode"})
	if err == nil {
		t.Fatalf("expected sync failure")
	}
	var exitErr *exitCodeError
	if !errors.As(err, &exitErr) || exitErr.code != shared.ExitSyncFailed {
		t.Fatalf("expected ExitSyncFailed, err=%v", err)
	}
}

func containsAll(v string, items []string) bool {
	for _, item := range items {
		if !strings.Contains(v, item) {
			return false
		}
	}
	return true
}
