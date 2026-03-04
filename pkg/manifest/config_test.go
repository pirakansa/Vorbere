package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSyncConfigBuildsRulesFromRepositories(t *testing.T) {
	const downloadDigest = "blake3:abcdef"
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{{
			URL: "https://example.com/base/",
			Files: []RepositoryFile{{
				FileName:       "a.txt",
				OutDir:         "dest",
				Rename:         "renamed.txt",
				DownloadDigest: downloadDigest,
			}},
		}},
	}

	resolved, err := BuildSyncConfig(cfg)
	if err != nil {
		t.Fatalf("BuildSyncConfig returned error: %v", err)
	}
	if len(resolved.Files) != 1 {
		t.Fatalf("expected one file rule, got %d", len(resolved.Files))
	}
	rule := resolved.Files[0]
	if rule.Path != filepath.Join("dest", "renamed.txt") {
		t.Fatalf("unexpected rule path: %s", rule.Path)
	}
	if rule.DownloadChecksum != downloadDigest {
		t.Fatalf("unexpected download checksum: %s", rule.DownloadChecksum)
	}
	src := resolved.Sources[rule.Source]
	if src.URL != "https://example.com/base/a.txt" {
		t.Fatalf("unexpected source url: %s", src.URL)
	}
}

func TestBuildSyncConfigArchiveExtractRule(t *testing.T) {
	const downloadDigest = "blake3:deadbeef"
	const outputDigest = "blake3:cafebabe"
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{{
			URL: "https://example.com/base/",
			Files: []RepositoryFile{{
				FileName:       "tool.tar.gz",
				DownloadDigest: downloadDigest,
				OutputDigest:   outputDigest,
				Encoding:       "tar+gzip",
				Extract:        "bin/tool",
				OutDir:         "/tmp/bin",
				Rename:         "tool",
				Mode:           "0755",
			}},
		}},
	}

	resolved, err := BuildSyncConfig(cfg)
	if err != nil {
		t.Fatalf("BuildSyncConfig returned error: %v", err)
	}
	rule := resolved.Files[0]
	if rule.DownloadChecksum != downloadDigest {
		t.Fatalf("unexpected download checksum: %s", rule.DownloadChecksum)
	}
	if rule.OutputChecksum != outputDigest {
		t.Fatalf("unexpected output checksum: %s", rule.OutputChecksum)
	}
	if rule.Encoding != EncodingTarGzip {
		t.Fatalf("unexpected encoding: %s", rule.Encoding)
	}
	if rule.Extract != "bin/tool" {
		t.Fatalf("unexpected extract: %s", rule.Extract)
	}
	if rule.Path != filepath.Join("/tmp/bin", "tool") {
		t.Fatalf("unexpected path: %s", rule.Path)
	}
}

func TestBuildSyncConfigAllowsDownloadDigestOnArchiveFullExtraction(t *testing.T) {
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{{
			URL: "https://example.com/base/",
			Files: []RepositoryFile{{
				FileName:       "tool.tar.xz",
				Encoding:       "tar+xz",
				OutDir:         "/tmp/lib",
				DownloadDigest: "blake3:deadbeef",
			}},
		}},
	}

	resolved, err := BuildSyncConfig(cfg)
	if err != nil {
		t.Fatalf("BuildSyncConfig returned error: %v", err)
	}
	if got := resolved.Files[0].DownloadChecksum; got != "blake3:deadbeef" {
		t.Fatalf("unexpected download checksum: %s", got)
	}
}

func TestBuildSyncConfigRejectsOutputDigestOnArchiveFullExtraction(t *testing.T) {
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{{
			URL: "https://example.com/base/",
			Files: []RepositoryFile{{
				FileName:     "tool.tar.xz",
				Encoding:     "tar+xz",
				OutDir:       "/tmp/lib",
				OutputDigest: "blake3:deadbeef",
			}},
		}},
	}

	if _, err := BuildSyncConfig(cfg); err == nil {
		t.Fatalf("expected output_digest validation error")
	}
}

func TestBuildSyncConfigRejectsDigestWithoutPrefix(t *testing.T) {
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{{
			URL: "https://example.com/base/",
			Files: []RepositoryFile{{
				FileName:       "a.txt",
				OutDir:         ".",
				DownloadDigest: "abcdef",
			}},
		}},
	}

	if _, err := BuildSyncConfig(cfg); err == nil {
		t.Fatalf("expected digest format validation error")
	}
}

func TestBuildSyncConfigAcceptsSHA256AndMD5Digests(t *testing.T) {
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{{
			URL: "https://example.com/base/",
			Files: []RepositoryFile{
				{
					FileName:       "sha256.txt",
					OutDir:         ".",
					DownloadDigest: "sha256:abcdef",
				},
				{
					FileName:     "md5.txt",
					OutDir:       ".",
					OutputDigest: "md5:abcdef",
				},
			},
		}},
	}

	resolved, err := BuildSyncConfig(cfg)
	if err != nil {
		t.Fatalf("BuildSyncConfig returned error: %v", err)
	}
	if got := resolved.Files[0].DownloadChecksum; got != "sha256:abcdef" {
		t.Fatalf("unexpected sha256 checksum: %s", got)
	}
	if got := resolved.Files[1].OutputChecksum; got != "md5:abcdef" {
		t.Fatalf("unexpected md5 checksum: %s", got)
	}
}

func TestBuildSyncConfigRejectsUnsupportedVersion(t *testing.T) {
	cfg := &TaskConfig{
		Version: 3,
	}
	if _, err := BuildSyncConfig(cfg); err == nil {
		t.Fatalf("expected unsupported version error")
	}
}

func TestBuildSyncConfigExpandsVarsBeforeEnvironmentExpansion(t *testing.T) {
	t.Setenv("HOME", "/tmp/vor-home")
	cfg := &TaskConfig{
		Version: 1,
		Vars: map[string]string{
			"TOOL_VERSION": "1.2.3",
		},
		Repositories: []Repository{{
			URL: "https://example.com/releases/${{ .vars.TOOL_VERSION }}/",
			Files: []RepositoryFile{{
				FileName: "tool-${{ .vars.TOOL_VERSION }}.txt",
				OutDir:   "$HOME/bin/${{ .vars.TOOL_VERSION }}",
			}},
		}},
	}

	resolved, err := BuildSyncConfig(cfg)
	if err != nil {
		t.Fatalf("BuildSyncConfig returned error: %v", err)
	}
	rule := resolved.Files[0]
	if got, want := rule.Path, filepath.Join("/tmp/vor-home", "bin", "1.2.3", "tool-1.2.3.txt"); got != want {
		t.Fatalf("unexpected rule path: got=%q want=%q", got, want)
	}
	src := resolved.Sources[rule.Source]
	if got, want := src.URL, "https://example.com/releases/1.2.3/tool-1.2.3.txt"; got != want {
		t.Fatalf("unexpected source url: got=%q want=%q", got, want)
	}
}

func TestBuildSyncConfigRejectsUndefinedVars(t *testing.T) {
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{{
			URL: "https://example.com/base/",
			Files: []RepositoryFile{{
				FileName: "${{ .vars.MISSING }}.txt",
				OutDir:   ".",
			}},
		}},
	}

	_, err := BuildSyncConfig(cfg)
	if err == nil {
		t.Fatalf("expected undefined var error")
	}
	message := err.Error()
	if !strings.Contains(message, "repositories[0].files[0].file_name") {
		t.Fatalf("expected field path in error, got: %v", err)
	}
	if !strings.Contains(message, "MISSING") {
		t.Fatalf("expected unresolved key in error, got: %v", err)
	}
}

func TestBuildSyncConfigAllowsLiteralBracesInTaskFields(t *testing.T) {
	cfg := &TaskConfig{
		Version: 1,
		Tasks: map[string]TaskDef{
			"echo": {
				Run: "echo '{{not-a-vars-template}}'",
				Env: map[string]string{
					"LITERAL": "{{keep-as-is}}",
				},
			},
		},
	}

	if _, err := BuildSyncConfig(cfg); err != nil {
		t.Fatalf("expected literal braces to be accepted, got error: %v", err)
	}
}

func TestBuildSyncConfigRejectsInvalidVarKeyReference(t *testing.T) {
	cfg := &TaskConfig{
		Version: 1,
		Tasks: map[string]TaskDef{
			"echo": {
				Run: "echo ${{ .vars.TOOL-VERSION }}",
			},
		},
	}

	_, err := BuildSyncConfig(cfg)
	if err == nil {
		t.Fatalf("expected invalid var key error")
	}
	message := err.Error()
	if !strings.Contains(message, "tasks.echo.run") {
		t.Fatalf("expected field path in error, got: %v", err)
	}
	if !strings.Contains(message, "TOOL-VERSION") {
		t.Fatalf("expected invalid key in error, got: %v", err)
	}
	if !strings.Contains(message, "allowed pattern") {
		t.Fatalf("expected key pattern hint in error, got: %v", err)
	}
}

func TestBuildSyncConfigExpandsRepositoryHeaderEnvironmentVariables(t *testing.T) {
	t.Setenv("VOR_TOKEN", "secret-token")
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{{
			URL: "https://example.com/base/",
			Headers: map[string]string{
				"Authorization": "Bearer ${VOR_TOKEN}",
			},
			Files: []RepositoryFile{{
				FileName: "a.txt",
				OutDir:   ".",
			}},
		}},
	}

	resolved, err := BuildSyncConfig(cfg)
	if err != nil {
		t.Fatalf("BuildSyncConfig returned error: %v", err)
	}
	rule := resolved.Files[0]
	src := resolved.Sources[rule.Source]
	if got := src.Headers["Authorization"]; got != "Bearer secret-token" {
		t.Fatalf("unexpected header value: %q", got)
	}
}

func TestBuildSyncConfigRejectsUndefinedRepositoryHeaderEnvironmentVariable(t *testing.T) {
	const envName = "VOR_UNDEFINED_SECRET"
	oldValue, hadOldValue := os.LookupEnv(envName)
	_ = os.Unsetenv(envName)
	t.Cleanup(func() {
		if hadOldValue {
			_ = os.Setenv(envName, oldValue)
			return
		}
		_ = os.Unsetenv(envName)
	})
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{{
			URL: "https://example.com/base/",
			Headers: map[string]string{
				"Authorization": "Bearer ${" + envName + "}",
			},
			Files: []RepositoryFile{{
				FileName: "a.txt",
				OutDir:   ".",
			}},
		}},
	}

	_, err := BuildSyncConfig(cfg)
	if err == nil {
		t.Fatalf("expected error for undefined header environment variable")
	}
	message := err.Error()
	if !strings.Contains(message, "headers[\"Authorization\"]") {
		t.Fatalf("expected header key context, got: %v", err)
	}
	if !strings.Contains(message, envName) {
		t.Fatalf("expected undefined env var name, got: %v", err)
	}
}
