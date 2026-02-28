package manifest

import (
	"path/filepath"
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
