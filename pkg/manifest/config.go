package manifest

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	DefaultTaskConfigVersion = 3
	SyncConfigVersion        = "v3"
	EncodingZstd             = "zstd"
	EncodingTarGzip          = "tar+gzip"
	EncodingTarXz            = "tar+xz"
	DigestAlgorithmBLAKE3    = "blake3"
	DigestAlgorithmSHA256    = "sha256"
	DigestAlgorithmMD5       = "md5"
)

func NormalizeTaskConfig(cfg *TaskConfig) {
	if cfg.Version == 0 {
		cfg.Version = DefaultTaskConfigVersion
	}
	if cfg.Tasks == nil {
		cfg.Tasks = map[string]TaskDef{}
	}
	if cfg.Repositories == nil {
		cfg.Repositories = []Repository{}
	}
}

func IsRemoteConfigLocation(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func ValidateTaskConfig(cfg *TaskConfig) error {
	for name, task := range cfg.Tasks {
		if task.Run == "" && len(task.DependsOn) == 0 {
			return fmt.Errorf("task %q must have run or depends_on", name)
		}
	}
	return nil
}

func ValidateSyncConfig(cfg *SyncConfig) error {
	for id, src := range cfg.Sources {
		if src.URL == "" {
			return fmt.Errorf("source %q url is required", id)
		}
	}
	for i, rule := range cfg.Files {
		if rule.Source == "" {
			return fmt.Errorf("files[%d].source is required", i)
		}
		if _, ok := cfg.Sources[rule.Source]; !ok {
			return fmt.Errorf("files[%d].source %q not found in sources", i, rule.Source)
		}
		if rule.Path == "" {
			return fmt.Errorf("files[%d].path is required", i)
		}
	}
	return nil
}

func BuildSyncConfig(taskCfg *TaskConfig) (*SyncConfig, error) {
	cfg := &SyncConfig{
		Version: SyncConfigVersion,
		Sources: map[string]Source{},
	}

	for repoIndex, repo := range taskCfg.Repositories {
		if strings.TrimSpace(repo.URL) == "" {
			return nil, fmt.Errorf("repositories[%d].url is required", repoIndex)
		}
		for fileIndex, file := range repo.Files {
			sourceID, source, rule, err := buildSyncEntry(repo, file, repoIndex, fileIndex)
			if err != nil {
				return nil, err
			}
			cfg.Sources[sourceID] = source
			cfg.Files = append(cfg.Files, rule)
		}
	}

	return cfg, nil
}

func buildSyncEntry(repo Repository, file RepositoryFile, repoIndex, fileIndex int) (string, Source, FileRule, error) {
	encoding, extract, err := validateRepositoryFile(file, repoIndex, fileIndex)
	if err != nil {
		return "", Source{}, FileRule{}, err
	}

	expandArchive := isArchiveEncoding(encoding) && extract == ""
	targetName := strings.TrimSpace(file.Rename)
	if !expandArchive && targetName == "" {
		derivedName, err := deriveTargetName(file.FileName, encoding, extract)
		if err != nil {
			return "", Source{}, FileRule{}, fmt.Errorf(
				"repositories[%d].files[%d] %w",
				repoIndex, fileIndex, err,
			)
		}
		targetName = derivedName
	}
	if !expandArchive && (targetName == "." || targetName == "/" || targetName == "") {
		return "", Source{}, FileRule{}, fmt.Errorf(
			"repositories[%d].files[%d] could not determine output filename",
			repoIndex, fileIndex,
		)
	}

	targetPath := os.ExpandEnv(file.OutDir)
	if !expandArchive {
		targetPath = filepath.Join(targetPath, targetName)
	}
	sourceID := fmt.Sprintf("r%df%d", repoIndex, fileIndex)
	source := Source{
		URL:     joinURL(repo.URL, file.FileName),
		Headers: repo.Headers,
	}
	rule := FileRule{
		Source:           sourceID,
		Path:             targetPath,
		Mode:             file.Mode,
		Checksum:         "",
		ArtifactChecksum: "",
		Encoding:         encoding,
		Extract:          extract,
		ExpandArchive:    expandArchive,
	}
	checksum, err := normalizeDigest(file.Digest)
	if err != nil {
		return "", Source{}, FileRule{}, fmt.Errorf("repositories[%d].files[%d].digest %w", repoIndex, fileIndex, err)
	}
	artifactChecksum, err := normalizeDigest(file.ArtifactDigest)
	if err != nil {
		return "", Source{}, FileRule{}, fmt.Errorf("repositories[%d].files[%d].artifact_digest %w", repoIndex, fileIndex, err)
	}
	rule.Checksum = checksum
	rule.ArtifactChecksum = artifactChecksum
	if rule.ExpandArchive {
		rule.Mode = ""
	}
	if rule.ExpandArchive && rule.Checksum != "" {
		return "", Source{}, FileRule{}, fmt.Errorf(
			"repositories[%d].files[%d].digest cannot be used when extract is omitted for archive encodings",
			repoIndex, fileIndex,
		)
	}

	return sourceID, source, rule, nil
}

func normalizeDigest(value string) (string, error) {
	raw := strings.TrimSpace(strings.ToLower(value))
	if raw == "" {
		return "", nil
	}
	algorithm, digest, ok := strings.Cut(raw, ":")
	if !ok || strings.TrimSpace(algorithm) == "" || strings.TrimSpace(digest) == "" {
		return "", fmt.Errorf(
			"must be in format %q, %q, or %q",
			DigestAlgorithmBLAKE3+":<hex>",
			DigestAlgorithmSHA256+":<hex>",
			DigestAlgorithmMD5+":<hex>",
		)
	}
	if !isSupportedDigestAlgorithm(algorithm) {
		return "", fmt.Errorf("unsupported algorithm %q", algorithm)
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", errors.New("must contain lowercase hex digest")
	}
	return algorithm + ":" + digest, nil
}

func isSupportedDigestAlgorithm(value string) bool {
	switch value {
	case DigestAlgorithmBLAKE3, DigestAlgorithmSHA256, DigestAlgorithmMD5:
		return true
	default:
		return false
	}
}

func validateRepositoryFile(file RepositoryFile, repoIndex, fileIndex int) (string, string, error) {
	if strings.TrimSpace(file.FileName) == "" {
		return "", "", fmt.Errorf("repositories[%d].files[%d].file_name is required", repoIndex, fileIndex)
	}
	if strings.TrimSpace(file.OutDir) == "" {
		return "", "", fmt.Errorf("repositories[%d].files[%d].out_dir is required", repoIndex, fileIndex)
	}

	encoding := strings.TrimSpace(strings.ToLower(file.Encoding))
	switch encoding {
	case "", EncodingZstd, EncodingTarGzip, EncodingTarXz:
	default:
		return "", "", fmt.Errorf(
			"repositories[%d].files[%d].encoding must be one of %q, %q, %q",
			repoIndex, fileIndex, EncodingZstd, EncodingTarGzip, EncodingTarXz,
		)
	}

	extract := strings.TrimSpace(file.Extract)
	if extract == "." {
		extract = ""
	}
	if extract != "" && !isArchiveEncoding(encoding) {
		return "", "", fmt.Errorf(
			"repositories[%d].files[%d].extract requires archive encoding",
			repoIndex, fileIndex,
		)
	}
	if isArchiveEncoding(encoding) {
		cleaned, err := normalizeExtractPath(extract)
		if err != nil {
			return "", "", fmt.Errorf(
				"repositories[%d].files[%d].extract %w",
				repoIndex, fileIndex, err,
			)
		}
		extract = cleaned
	} else {
		extract = ""
	}
	if file.Symlink != nil {
		return "", "", fmt.Errorf("repositories[%d].files[%d].symlink is not supported", repoIndex, fileIndex)
	}
	return encoding, extract, nil
}

func joinURL(base, fileName string) string {
	base = strings.TrimSpace(base)
	fileName = strings.TrimSpace(fileName)
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(fileName, "/")
}

func isArchiveEncoding(encoding string) bool {
	return encoding == EncodingTarGzip || encoding == EncodingTarXz
}

func normalizeExtractPath(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	value = strings.TrimPrefix(value, "./")
	cleaned := path.Clean(value)
	if cleaned == "." {
		return "", nil
	}
	if path.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("must stay within archive root")
	}
	return cleaned, nil
}

func deriveTargetName(fileName, encoding, extract string) (string, error) {
	switch {
	case isArchiveEncoding(encoding):
		name := path.Base(extract)
		if name == "." || name == "/" || name == "" {
			return "", errors.New("could not determine output filename from extract")
		}
		return name, nil
	case encoding == EncodingZstd:
		base := path.Base(fileName)
		switch {
		case strings.HasSuffix(base, ".zst"):
			return strings.TrimSuffix(base, ".zst"), nil
		case strings.HasSuffix(base, ".zstd"):
			return strings.TrimSuffix(base, ".zstd"), nil
		default:
			return base, nil
		}
	default:
		return path.Base(fileName), nil
	}
}
