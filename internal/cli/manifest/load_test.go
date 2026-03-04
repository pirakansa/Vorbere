package manifest

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSyncConfigBuildsRulesFromRepositories(t *testing.T) {
	temp := t.TempDir()
	const downloadDigest = "blake3:abcdef"
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{
			{
				URL: "https://example.com/base/",
				Files: []RepositoryFile{
					{
						FileName:       "a.txt",
						OutDir:         "dest",
						Rename:         "renamed.txt",
						DownloadDigest: downloadDigest,
					},
				},
			},
		},
	}

	resolved, err := ResolveSyncConfig(cfg, filepath.Join(temp, "vorbere.yaml"))
	if err != nil {
		t.Fatalf("ResolveSyncConfig returned error: %v", err)
	}
	if len(resolved.Files) != 1 {
		t.Fatalf("expected one file rule, got %d", len(resolved.Files))
	}
	rule := resolved.Files[0]
	if rule.Path != filepath.Join("dest", "renamed.txt") {
		t.Fatalf("unexpected rule path: %s", rule.Path)
	}
	if rule.DownloadChecksum != downloadDigest {
		t.Fatalf("unexpected checksum: %s", rule.DownloadChecksum)
	}
	src := resolved.Sources[rule.Source]
	if src.URL != "https://example.com/base/a.txt" {
		t.Fatalf("unexpected source url: %s", src.URL)
	}
}

func TestResolveSyncConfigCollectsAllRepositoryFiles(t *testing.T) {
	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{
			{
				URL: "https://example.com",
				Files: []RepositoryFile{
					{FileName: "base.txt", OutDir: "."},
					{FileName: "profile.txt", OutDir: "."},
				},
			},
		},
	}
	resolved, err := ResolveSyncConfig(cfg, filepath.Join(temp, "vorbere.yaml"))
	if err != nil {
		t.Fatalf("ResolveSyncConfig returned error: %v", err)
	}
	if len(resolved.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(resolved.Files))
	}
}

func TestResolveSyncConfigRejectsUnknownEncoding(t *testing.T) {
	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{
			{
				URL: "https://example.com",
				Files: []RepositoryFile{
					{FileName: "a.txt", OutDir: ".", Encoding: "zip"},
				},
			},
		},
	}

	if _, err := ResolveSyncConfig(cfg, filepath.Join(temp, "vorbere.yaml")); err == nil {
		t.Fatalf("expected invalid encoding error")
	}
}

func TestValidateSyncConfigRejectsMissingSourceURL(t *testing.T) {
	cfg := &SyncConfig{
		Version: "v1",
		Sources: map[string]Source{
			"s1": {},
		},
		Files: []FileRule{{Source: "s1", Path: "a.txt"}},
	}
	if err := ValidateSyncConfig(cfg); err == nil {
		t.Fatalf("expected missing source url error")
	}
}

func TestResolveSyncConfigRejectsTaskWithoutRunOrDependsOn(t *testing.T) {
	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: 1,
		Tasks: map[string]TaskDef{
			"broken": {},
		},
	}
	if _, err := ResolveSyncConfig(cfg, filepath.Join(temp, "vorbere.yaml")); err == nil {
		t.Fatalf("expected validation error for task without run and depends_on")
	}
}

func TestLoadTaskConfigFromRemoteURL(t *testing.T) {
	oldClient := http.DefaultClient
	t.Cleanup(func() {
		http.DefaultClient = oldClient
	})
	http.DefaultClient = &http.Client{
		Transport: loadTestRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("version: 1\n")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	cfg, err := LoadTaskConfig("https://example.com/vorbere.yaml")
	if err != nil {
		t.Fatalf("LoadTaskConfig returned error: %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("expected version=1, got=%d", cfg.Version)
	}
}

func TestLoadTaskConfigFromRemoteURLReturnsErrorOnNon2xx(t *testing.T) {
	oldClient := http.DefaultClient
	t.Cleanup(func() {
		http.DefaultClient = oldClient
	})
	http.DefaultClient = &http.Client{
		Transport: loadTestRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	if _, err := LoadTaskConfig("https://example.com/vorbere.yaml"); err == nil {
		t.Fatalf("expected error for non-2xx response")
	}
}

func TestLoadTaskConfigRejectsUnknownFields(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "vorbere.yaml")
	content := `version: 1
repositories:
  - url: https://example.com
    files:
      - file_name: a.txt
        out_dir: .
        artifact_digest: blake3:deadbeef
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadTaskConfig(configPath); err == nil {
		t.Fatalf("expected unknown field error")
	}
}

func TestLoadTaskConfigRejectsLegacyDigestField(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "vorbere.yaml")
	content := `version: 1
repositories:
  - url: https://example.com
    files:
      - file_name: a.txt
        out_dir: .
        digest: blake3:deadbeef
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadTaskConfig(configPath); err == nil {
		t.Fatalf("expected unknown field error for legacy digest")
	}
}

func TestLoadTaskConfigExpandsVarsInTaskFields(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "vorbere.yaml")
	content := `version: 1
vars:
  TOOL_VERSION: "1.2.3"
tasks:
  print:
    run: "echo {{ .vars.TOOL_VERSION }}"
    cwd: "bin/{{ .vars.TOOL_VERSION }}"
    env:
      TOOL_VERSION: "{{ .vars.TOOL_VERSION }}"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadTaskConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTaskConfig returned error: %v", err)
	}
	task := cfg.Tasks["print"]
	if got, want := task.Run, "echo 1.2.3"; got != want {
		t.Fatalf("unexpected run: got=%q want=%q", got, want)
	}
	if got, want := task.CWD, "bin/1.2.3"; got != want {
		t.Fatalf("unexpected cwd: got=%q want=%q", got, want)
	}
	if got, want := task.Env["TOOL_VERSION"], "1.2.3"; got != want {
		t.Fatalf("unexpected env TOOL_VERSION: got=%q want=%q", got, want)
	}
}

func TestLoadTaskConfigRejectsUndefinedVars(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "vorbere.yaml")
	content := `version: 1
tasks:
  print:
    run: "echo {{ .vars.MISSING }}"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadTaskConfig(configPath)
	if err == nil {
		t.Fatalf("expected undefined var error")
	}
	message := err.Error()
	if !strings.Contains(message, "tasks.print.run") {
		t.Fatalf("expected field path in error, got: %v", err)
	}
	if !strings.Contains(message, "MISSING") {
		t.Fatalf("expected unresolved key in error, got: %v", err)
	}
}

func TestLoadTaskConfigAllowsLiteralBracesInTaskFields(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "vorbere.yaml")
	content := `version: 1
tasks:
  print:
    run: "echo '{{keep}}'"
    env:
      TOKEN: "{{not-a-vars-template}}"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadTaskConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTaskConfig returned error: %v", err)
	}
	task := cfg.Tasks["print"]
	if got, want := task.Run, "echo '{{keep}}'"; got != want {
		t.Fatalf("unexpected run: got=%q want=%q", got, want)
	}
	if got, want := task.Env["TOKEN"], "{{not-a-vars-template}}"; got != want {
		t.Fatalf("unexpected env TOKEN: got=%q want=%q", got, want)
	}
}

func TestIsRemoteConfigLocation(t *testing.T) {
	if !IsRemoteConfigLocation("https://example.com/vorbere.yaml") {
		t.Fatalf("expected https URL to be remote config")
	}
	if !IsRemoteConfigLocation("http://example.com/vorbere.yaml") {
		t.Fatalf("expected http URL to be remote config")
	}
	if IsRemoteConfigLocation("vorbere.yaml") {
		t.Fatalf("expected local path to not be remote config")
	}
}

func TestResolveSyncConfigDoesNotExpandRepositoryHeaderEnvForRemoteConfig(t *testing.T) {
	t.Setenv("REMOTE_ONLY_TOKEN", "secret-token")
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{
			{
				URL: "https://example.com/base/",
				Headers: map[string]string{
					"Authorization": "Bearer ${REMOTE_ONLY_TOKEN}",
				},
				Files: []RepositoryFile{
					{
						FileName: "a.txt",
						OutDir:   "dest",
					},
				},
			},
		},
	}

	resolved, err := ResolveSyncConfig(cfg, "https://example.com/vorbere.yaml")
	if err != nil {
		t.Fatalf("ResolveSyncConfig returned error: %v", err)
	}
	rule := resolved.Files[0]
	src := resolved.Sources[rule.Source]
	if got := src.Headers["Authorization"]; got != "Bearer ${REMOTE_ONLY_TOKEN}" {
		t.Fatalf("expected literal header for remote config, got %q", got)
	}
}

type loadTestRoundTripFunc func(*http.Request) (*http.Response, error)

func (f loadTestRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
