package manifest

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pirakansa/vorbere/internal/cli/shared"
	"github.com/ulikunitz/xz"
)

const (
	outcomeCreated   = "created"
	outcomeUpdated   = "updated"
	outcomeUnchanged = "unchanged"
)

// SyncOptions controls sync behavior.
type SyncOptions struct {
	RootDir   string
	Overwrite bool
	DryRun    bool
	Now       func() time.Time
	OnFile    func(SyncFileProgress)
}

// SyncFileProgress describes one processed file during sync.
type SyncFileProgress struct {
	Index   int
	Total   int
	Path    string
	Outcome string
}

// SyncResult describes sync outcome.
type SyncResult struct {
	Created   int
	Updated   int
	Unchanged int
}

func Sync(cfg *SyncConfig, opts SyncOptions) (*SyncResult, error) {
	if opts.RootDir == "" {
		return nil, errors.New("root dir is required")
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if err := ValidateSyncConfig(cfg); err != nil {
		return nil, err
	}

	rules := cfg.Files

	res := &SyncResult{}
	total := len(rules)
	for index, rule := range rules {
		src := cfg.Sources[rule.Source]
		artifact, err := download(src)
		if err != nil {
			return nil, err
		}
		if err := verifyChecksum(artifact, rule.ArtifactChecksum); err != nil {
			return nil, err
		}

		target := resolveTargetPath(opts.RootDir, rule.Path)
		outcome, err := applyProcessedRule(target, artifact, rule, opts)
		if err != nil {
			return nil, err
		}

		recordOutcome(res, outcome)
		if opts.OnFile != nil {
			opts.OnFile(SyncFileProgress{
				Index:   index + 1,
				Total:   total,
				Path:    target,
				Outcome: outcome,
			})
		}
	}
	return res, nil
}

func applyProcessedRule(targetPath string, artifact []byte, rule FileRule, opts SyncOptions) (string, error) {
	processed, err := processArtifact(artifact, rule)
	if err != nil {
		return "", err
	}
	if processed.single != nil {
		if err := verifyChecksum(processed.single.content, rule.Checksum); err != nil {
			return "", err
		}
		modeValue := rule.Mode
		if modeValue == "" && processed.single.mode != 0 {
			modeValue = fmt.Sprintf("%04o", uint32(processed.single.mode.Perm()))
		}
		return applyRule(targetPath, processed.single.content, modeValue, opts)
	}
	if rule.Checksum != "" {
		return "", errors.New("digest cannot be used when extract resolves to multiple files")
	}
	return applyArchiveRule(targetPath, processed.entries, opts)
}

type processedArtifact struct {
	single  *processedFile
	entries []archiveEntry
}

type processedFile struct {
	content []byte
	mode    os.FileMode
}

type archiveEntry struct {
	path string
	body []byte
	mode os.FileMode
}

func processArtifact(artifact []byte, rule FileRule) (*processedArtifact, error) {
	switch rule.Encoding {
	case "":
		return &processedArtifact{single: &processedFile{content: artifact}}, nil
	case EncodingZstd:
		decoded, err := decodeZstd(artifact)
		if err != nil {
			return nil, err
		}
		return &processedArtifact{single: &processedFile{content: decoded}}, nil
	case EncodingTarGzip, EncodingTarXz:
		selected, err := selectArchiveContent(artifact, rule.Encoding, rule.Extract, rule.ExpandArchive)
		if err != nil {
			return nil, err
		}
		return selected, nil
	default:
		return nil, fmt.Errorf("unsupported encoding %q", rule.Encoding)
	}
}

func decodeZstd(content []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()
	return decoder.DecodeAll(content, nil)
}

func selectArchiveContent(content []byte, encoding, extract string, expandArchive bool) (*processedArtifact, error) {
	entries, err := readArchiveEntries(content, encoding)
	if err != nil {
		return nil, err
	}
	if expandArchive {
		return &processedArtifact{entries: entries}, nil
	}
	for _, entry := range entries {
		if entry.path == extract {
			return &processedArtifact{
				single: &processedFile{
					content: entry.body,
					mode:    entry.mode,
				},
			}, nil
		}
	}
	prefix := extract + "/"
	var children []archiveEntry
	for _, entry := range entries {
		if strings.HasPrefix(entry.path, prefix) {
			rel := strings.TrimPrefix(entry.path, prefix)
			if rel == "" {
				continue
			}
			children = append(children, archiveEntry{
				path: rel,
				body: entry.body,
				mode: entry.mode,
			})
		}
	}
	if len(children) > 0 {
		return &processedArtifact{entries: children}, nil
	}
	return nil, fmt.Errorf("extract path %q not found in archive", extract)
}

func readArchiveEntries(content []byte, encoding string) ([]archiveEntry, error) {
	var (
		reader io.Reader = bytes.NewReader(content)
		closer io.Closer
	)
	switch encoding {
	case EncodingTarGzip:
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		reader = gzipReader
		closer = gzipReader
	case EncodingTarXz:
		xzReader, err := xz.NewReader(reader)
		if err != nil {
			return nil, err
		}
		reader = xzReader
	default:
		return nil, fmt.Errorf("unsupported archive encoding %q", encoding)
	}
	if closer != nil {
		defer closer.Close()
	}

	tarReader := tar.NewReader(reader)
	var entries []archiveEntry
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if !header.FileInfo().Mode().IsRegular() {
			continue
		}

		name, err := normalizeArchiveEntryName(header.Name)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, err
		}
		entries = append(entries, archiveEntry{
			path: name,
			body: body,
			mode: header.FileInfo().Mode().Perm(),
		})
	}
	return entries, nil
}

func normalizeArchiveEntryName(value string) (string, error) {
	cleaned := filepath.Clean(value)
	cleaned = strings.TrimPrefix(cleaned, "./")
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("invalid archive entry path %q", value)
	}
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry path escapes root: %q", value)
	}
	return filepath.ToSlash(cleaned), nil
}

func applyArchiveRule(targetRoot string, entries []archiveEntry, opts SyncOptions) (string, error) {
	anyCreated := false
	anyUpdated := false
	for _, entry := range entries {
		targetPath, err := resolveArchiveTargetPath(targetRoot, entry.path)
		if err != nil {
			return "", err
		}
		mode := fmt.Sprintf("%04o", uint32(entry.mode))
		if entry.mode == 0 {
			mode = "0644"
		}
		outcome, err := applyRule(targetPath, entry.body, mode, opts)
		if err != nil {
			return "", err
		}
		if outcome == outcomeUpdated {
			anyUpdated = true
		}
		if outcome == outcomeCreated {
			anyCreated = true
		}
	}
	if anyUpdated {
		return outcomeUpdated, nil
	}
	if anyCreated {
		return outcomeCreated, nil
	}
	return outcomeUnchanged, nil
}

func resolveArchiveTargetPath(root, rel string) (string, error) {
	target := filepath.Join(root, filepath.FromSlash(rel))
	cleanRoot := filepath.Clean(root)
	cleanTarget := filepath.Clean(target)
	if cleanTarget != cleanRoot && !strings.HasPrefix(cleanTarget, cleanRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry path escapes target root: %q", rel)
	}
	return target, nil
}

func applyRule(targetPath string, incoming []byte, fileMode string, opts SyncOptions) (string, error) {
	current, err := os.ReadFile(targetPath)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	incomingHash := shared.SHA256Hex(incoming)
	if exists && shared.SHA256Hex(current) == incomingHash {
		return outcomeUnchanged, nil
	}

	backup := shared.BackupTimestamp
	if opts.Overwrite {
		backup = shared.BackupNone
	}

	return writeTarget(targetPath, incoming, current, exists, fileMode, backup, opts)
}

func writeTarget(path string, incoming, current []byte, existed bool, fileMode, backup string, opts SyncOptions) (string, error) {
	if opts.DryRun {
		if existed {
			return outcomeUpdated, nil
		}
		return outcomeCreated, nil
	}
	if existed {
		if err := shared.BackupFile(path, current, backup, opts.Now()); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	perm, err := resolveOutputMode(fileMode)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, incoming, perm); err != nil {
		return "", err
	}
	if existed {
		return outcomeUpdated, nil
	}
	return outcomeCreated, nil
}

func resolveOutputMode(v string) (os.FileMode, error) {
	if strings.TrimSpace(v) == "" {
		return 0o644, nil
	}
	parsed, err := strconv.ParseUint(v, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid file mode %q", v)
	}
	return os.FileMode(parsed), nil
}

func download(src Source) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, src.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range src.Headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download failed: %s status=%d", src.URL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func verifyChecksum(content []byte, checksum string) error {
	if checksum == "" {
		return nil
	}
	value := strings.TrimSpace(strings.ToLower(checksum))
	if value == "" {
		return nil
	}
	if shared.BLAKE3Hex(content) != value {
		return errors.New("checksum mismatch")
	}
	return nil
}

func resolveTargetPath(rootDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(rootDir, path)
}

func recordOutcome(res *SyncResult, outcome string) {
	switch outcome {
	case outcomeCreated:
		res.Created++
	case outcomeUpdated:
		res.Updated++
	case outcomeUnchanged:
		res.Unchanged++
	}
}
