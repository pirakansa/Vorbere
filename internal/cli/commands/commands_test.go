package commands

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/pirakansa/vorbere/internal/cli/shared"
)

func TestMapExitCode(t *testing.T) {
	if got := mapExitCode(newExitCodeError(shared.ExitTaskFailed, errors.New("x"))); got != shared.ExitTaskFailed {
		t.Fatalf("expected %d got %d", shared.ExitTaskFailed, got)
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
	if !containsAll(string(b), []string{"version: 1", "repositories:", "file_name:"}) {
		t.Fatalf("unexpected template content:\n%s", string(b))
	}
}

func TestRunCommandReturnsDefinedExitCodes(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "vorbere.yaml")
	cfg := `version: 1
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
		_, _ = w.Write([]byte("version: 1\ntasks:\n  test:\n    run: \"echo ok\"\n"))
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

func TestSyncCommandSucceedsWithOverwriteFlag(t *testing.T) {
	temp := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	taskBody := `version: 1
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
	if err := runSyncWithOptions(ctx, syncCommandOptions{overwrite: true}); err != nil {
		t.Fatalf("expected sync success with overwrite flag, err=%v", err)
	}
}

func TestVersionCommandPrintsVersion(t *testing.T) {
	cmd := newVersionCmd("v0.1.0")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	if out.String() != "v0.1.0\n" {
		t.Fatalf("expected version output %q, got %q", "v0.1.0\n", out.String())
	}
}

func TestResolveVersionUsesBuildInfoWhenDefaultVersion(t *testing.T) {
	orig := readBuildInfo
	t.Cleanup(func() { readBuildInfo = orig })
	readBuildInfo = func() (info *debug.BuildInfo, ok bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Version: "v0.2.0",
			},
		}, true
	}

	if got := resolveVersion(defaultVersionValue); got != "v0.2.0" {
		t.Fatalf("expected build info version, got %q", got)
	}
}

func TestResolveVersionPrefersProvidedVersion(t *testing.T) {
	orig := readBuildInfo
	t.Cleanup(func() { readBuildInfo = orig })
	readBuildInfo = func() (info *debug.BuildInfo, ok bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Version: "v9.9.9",
			},
		}, true
	}

	if got := resolveVersion("v0.3.0"); got != "v0.3.0" {
		t.Fatalf("expected provided version, got %q", got)
	}
}

func TestResolveVersionFallsBackToDefaultWhenBuildInfoUnavailable(t *testing.T) {
	orig := readBuildInfo
	t.Cleanup(func() { readBuildInfo = orig })
	readBuildInfo = func() (info *debug.BuildInfo, ok bool) {
		return nil, false
	}

	if got := resolveVersion(defaultVersionValue); got != defaultVersionValue {
		t.Fatalf("expected default version %q, got %q", defaultVersionValue, got)
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
