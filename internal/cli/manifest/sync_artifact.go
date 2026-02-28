package manifest

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

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

func applyProcessedRule(targetPath string, artifact []byte, rule FileRule, opts SyncOptions) (string, error) {
	processed, err := processArtifact(artifact, rule)
	if err != nil {
		return "", err
	}
	if processed.single != nil {
		return applySingleOutput(targetPath, processed.single, rule, opts)
	}
	return applyMultiOutput(targetPath, processed.entries, opts)
}

func applySingleOutput(targetPath string, file *processedFile, rule FileRule, opts SyncOptions) (string, error) {
	modeValue := rule.Mode
	if modeValue == "" && file.mode != 0 {
		modeValue = fmt.Sprintf("%04o", uint32(file.mode.Perm()))
	}
	return applyRule(targetPath, file.content, modeValue, opts)
}

func applyMultiOutput(targetRoot string, entries []archiveEntry, opts SyncOptions) (string, error) {
	return applyArchiveEntries(targetRoot, entries, opts)
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
		return selectArchiveContent(artifact, rule.Encoding, rule.Extract, rule.ExpandArchive)
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
			relativePath := strings.TrimPrefix(entry.path, prefix)
			if relativePath == "" {
				continue
			}
			children = append(children, archiveEntry{
				path: relativePath,
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
	reader, closer, err := openArchiveReader(content, encoding)
	if err != nil {
		return nil, err
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

		entryPath, err := normalizeArchiveEntryName(header.Name)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, err
		}
		entries = append(entries, archiveEntry{
			path: entryPath,
			body: body,
			mode: header.FileInfo().Mode().Perm(),
		})
	}
	return entries, nil
}

func openArchiveReader(content []byte, encoding string) (io.Reader, io.Closer, error) {
	var baseReader io.Reader = bytes.NewReader(content)
	switch encoding {
	case EncodingTarGzip:
		gzipReader, err := gzip.NewReader(baseReader)
		if err != nil {
			return nil, nil, err
		}
		return gzipReader, gzipReader, nil
	case EncodingTarXz:
		xzReader, err := xz.NewReader(baseReader)
		if err != nil {
			return nil, nil, err
		}
		return xzReader, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported archive encoding %q", encoding)
	}
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

func applyArchiveEntries(targetRoot string, entries []archiveEntry, opts SyncOptions) (string, error) {
	anyCreated := false
	anyUpdated := false

	for _, entry := range entries {
		targetPath, err := resolveArchiveTargetPath(targetRoot, entry.path)
		if err != nil {
			return "", err
		}

		modeValue := "0644"
		if entry.mode != 0 {
			modeValue = fmt.Sprintf("%04o", uint32(entry.mode))
		}
		outcome, err := applyRule(targetPath, entry.body, modeValue, opts)
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

	switch {
	case anyUpdated:
		return outcomeUpdated, nil
	case anyCreated:
		return outcomeCreated, nil
	default:
		return outcomeUnchanged, nil
	}
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
