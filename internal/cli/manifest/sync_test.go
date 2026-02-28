package manifest

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pirakansa/vorbere/internal/cli/shared"
)

func TestSyncDefaultMakesTimestampBackupOnUpdate(t *testing.T) {
	content := "v1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(content))
	}))
	defer server.Close()

	temp := t.TempDir()
	target := "managed/file.txt"

	cfg := &SyncConfig{
		Version: "v1",
		Sources: map[string]Source{
			"src": {URL: server.URL},
		},
		Files: []FileRule{{Source: "src", Path: target}},
	}

	opts := SyncOptions{RootDir: temp, Now: func() time.Time { return time.Unix(0, 0) }}
	res, err := Sync(cfg, opts)
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("expected created=1 got %+v", res)
	}

	content = "v2"
	absTarget := filepath.Join(temp, target)
	if err := os.WriteFile(absTarget, []byte("local-change"), 0o644); err != nil {
		t.Fatalf("write local change: %v", err)
	}

	updateOpts := opts
	updateOpts.Now = func() time.Time { return time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC) }
	res, err = Sync(cfg, updateOpts)
	if err != nil {
		t.Fatalf("expected default sync success, got err=%v", err)
	}
	if res.Updated != 1 {
		t.Fatalf("expected updated=1 got %+v", res)
	}

	b, err := os.ReadFile(absTarget)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(b) != "v2" {
		t.Fatalf("expected target=v2 got %q", string(b))
	}

	backupPath := fmt.Sprintf("%s.%s.bak", absTarget, "20260227120000")
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected backup file %s: %v", backupPath, err)
	}
}

func TestSyncOverwriteFlagSkipsTimestampBackup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("remote-v2"))
	}))
	defer server.Close()

	temp := t.TempDir()
	target := filepath.Join(temp, "overwrite.txt")
	if err := os.WriteFile(target, []byte("local"), 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	cfg := &SyncConfig{
		Version: "v1",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files:   []FileRule{{Source: "src", Path: "overwrite.txt"}},
	}
	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	res, err := Sync(cfg, SyncOptions{RootDir: temp, Overwrite: true, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if res.Updated != 1 {
		t.Fatalf("expected updated=1 got %+v", res)
	}
	backupPath := fmt.Sprintf("%s.%s.bak", target, "20260228100000")
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file with overwrite flag, err=%v", err)
	}
}

func TestSyncDryRunDoesNotWriteFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("remote"))
	}))
	defer server.Close()

	temp := t.TempDir()
	cfg := &SyncConfig{
		Version: "v1",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files:   []FileRule{{Source: "src", Path: "dryrun.txt"}},
	}

	res, err := Sync(cfg, SyncOptions{RootDir: temp, DryRun: true})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("expected created=1 got %+v", res)
	}
	if _, err := os.Stat(filepath.Join(temp, "dryrun.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected no file write during dry-run, err=%v", err)
	}
}

func TestSyncChecksumValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	temp := t.TempDir()
	cases := []struct {
		name      string
		algorithm string
		digest    string
	}{
		{name: "blake3", algorithm: DigestAlgorithmBLAKE3, digest: shared.BLAKE3Hex([]byte("content"))},
		{name: "sha256", algorithm: DigestAlgorithmSHA256, digest: shared.SHA256Hex([]byte("content"))},
		{name: "md5", algorithm: DigestAlgorithmMD5, digest: shared.MD5Hex([]byte("content"))},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := &SyncConfig{
				Version: "v1",
				Sources: map[string]Source{"src": {URL: server.URL}},
				Files: []FileRule{
					{
						Source:   "src",
						Path:     "checksum-" + tc.name + ".txt",
						Checksum: checksumSpec(tc.algorithm, tc.digest),
					},
				},
			}

			if _, err := Sync(cfg, SyncOptions{RootDir: temp}); err != nil {
				t.Fatalf("sync with valid checksum failed: %v", err)
			}

			cfg.Files[0].Path = "checksum-" + tc.name + "-mismatch.txt"
			cfg.Files[0].Checksum = checksumSpec(tc.algorithm, tc.digest+"00")
			if _, err := Sync(cfg, SyncOptions{RootDir: temp}); err == nil {
				t.Fatalf("expected checksum mismatch error")
			}
		})
	}
}

func TestSyncReturnsErrorForHTTPStatusFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	temp := t.TempDir()
	cfg := &SyncConfig{
		Version: "v1",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files:   []FileRule{{Source: "src", Path: "fail.txt"}},
	}
	if _, err := Sync(cfg, SyncOptions{RootDir: temp}); err == nil {
		t.Fatalf("expected HTTP status failure")
	}
}

func TestSyncReportsPerFileProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	temp := t.TempDir()
	cfg := &SyncConfig{
		Version: "v1",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files: []FileRule{
			{Source: "src", Path: "a.txt"},
			{Source: "src", Path: "b.txt"},
		},
	}

	var progress []SyncFileProgress
	_, err := Sync(cfg, SyncOptions{
		RootDir: temp,
		OnFile: func(item SyncFileProgress) {
			progress = append(progress, item)
		},
	})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if len(progress) != 2 {
		t.Fatalf("expected 2 progress entries, got %d", len(progress))
	}
	if progress[0].Index != 1 || progress[0].Total != 2 || progress[0].Outcome != outcomeCreated {
		t.Fatalf("unexpected first progress entry: %#v", progress[0])
	}
	if progress[1].Index != 2 || progress[1].Total != 2 || progress[1].Outcome != outcomeCreated {
		t.Fatalf("unexpected second progress entry: %#v", progress[1])
	}
}

func TestSyncDigestValidationForDownloadedArtifactWithZstd(t *testing.T) {
	payload := []byte("decoded-content")
	artifact := mustEncodeZstd(t, payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(artifact)
	}))
	defer server.Close()

	temp := t.TempDir()
	cfg := &SyncConfig{
		Version: "v1",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files: []FileRule{
			{
				Source:   "src",
				Path:     "bin/tool",
				Encoding: EncodingZstd,
				Checksum: checksumSpec(DigestAlgorithmBLAKE3, shared.BLAKE3Hex(artifact)),
				Mode:     "0755",
			},
		},
	}

	if _, err := Sync(cfg, SyncOptions{RootDir: temp}); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(temp, "bin/tool"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("unexpected output: %q", string(got))
	}

	cfg.Files[0].Path = "bin/tool-2"
	cfg.Files[0].Checksum = checksumSpec(DigestAlgorithmBLAKE3, shared.BLAKE3Hex([]byte("bad")))
	if _, err := Sync(cfg, SyncOptions{RootDir: temp}); err == nil {
		t.Fatalf("expected artifact checksum mismatch")
	}

	cfg.Files[0].Path = "bin/tool-3"
	cfg.Files[0].Checksum = checksumSpec(DigestAlgorithmBLAKE3, shared.BLAKE3Hex(payload))
	if _, err := Sync(cfg, SyncOptions{RootDir: temp}); err == nil {
		t.Fatalf("expected decoded checksum mismatch when digest checks downloaded artifact")
	}
}

func TestSyncTarGzipExtractFile(t *testing.T) {
	artifact := mustBuildTarGzip(t, map[string]string{
		"pkg/bin/tool":  "tool-binary",
		"pkg/README.md": "readme",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(artifact)
	}))
	defer server.Close()

	temp := t.TempDir()
	cfg := &SyncConfig{
		Version: "v1",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files: []FileRule{
			{
				Source:   "src",
				Path:     "bin/tool",
				Encoding: EncodingTarGzip,
				Extract:  "pkg/bin/tool",
				Checksum: checksumSpec(DigestAlgorithmBLAKE3, shared.BLAKE3Hex(artifact)),
				Mode:     "0755",
			},
		},
	}

	if _, err := Sync(cfg, SyncOptions{RootDir: temp}); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(temp, "bin/tool"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(got) != "tool-binary" {
		t.Fatalf("unexpected output: %q", string(got))
	}
}

func mustEncodeZstd(t *testing.T, content []byte) []byte {
	t.Helper()
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatalf("zstd.NewWriter: %v", err)
	}
	defer encoder.Close()
	return encoder.EncodeAll(content, nil)
}

func mustBuildTarGzip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	gzipWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader(%s): %v", name, err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatalf("Write(%s): %v", name, err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tarWriter.Close: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzipWriter.Close: %v", err)
	}
	return buf.Bytes()
}

func checksumSpec(algorithm, hexDigest string) string {
	return algorithm + ":" + hexDigest
}
