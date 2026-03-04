package manifest

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	DefaultTaskConfigVersion = 1
	SyncConfigVersion        = "v1"
	EncodingZstd             = "zstd"
	EncodingTarGzip          = "tar+gzip"
	EncodingTarXz            = "tar+xz"
	DigestAlgorithmBLAKE3    = "blake3"
	DigestAlgorithmSHA256    = "sha256"
	DigestAlgorithmMD5       = "md5"
)

var headerEnvPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
var varsTemplatePattern = regexp.MustCompile(`\$\{\{\s*\.vars\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
var varsReferencePattern = regexp.MustCompile(`\$\{\{\s*\.vars\.([^}\s]+)\s*\}\}`)
var varsKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type BuildSyncConfigOptions struct {
	ExpandRepositoryHeaderEnv bool
}

func NormalizeTaskConfig(cfg *TaskConfig) {
	if cfg.Version == 0 {
		cfg.Version = DefaultTaskConfigVersion
	}
	if cfg.Vars == nil {
		cfg.Vars = map[string]string{}
	}
	if cfg.Tasks == nil {
		cfg.Tasks = map[string]TaskDef{}
	}
	if cfg.Repositories == nil {
		cfg.Repositories = []Repository{}
	}
}

func ExpandTaskConfigTemplates(cfg *TaskConfig) error {
	for name, task := range cfg.Tasks {
		expandedTask, err := expandTaskDefTemplates(name, task, cfg.Vars)
		if err != nil {
			return err
		}
		cfg.Tasks[name] = expandedTask
	}

	for repoIndex, repo := range cfg.Repositories {
		expandedRepo, err := expandRepositoryTemplates(repoIndex, repo, cfg.Vars)
		if err != nil {
			return err
		}
		cfg.Repositories[repoIndex] = expandedRepo
	}

	return nil
}

func expandTaskDefTemplates(taskName string, task TaskDef, vars map[string]string) (TaskDef, error) {
	run, err := expandVarsTemplate(task.Run, vars, "tasks."+taskName+".run")
	if err != nil {
		return TaskDef{}, err
	}
	task.Run = run

	cwd, err := expandVarsTemplate(task.CWD, vars, "tasks."+taskName+".cwd")
	if err != nil {
		return TaskDef{}, err
	}
	task.CWD = cwd

	for key, value := range task.Env {
		expanded, expandErr := expandVarsTemplate(value, vars, "tasks."+taskName+".env."+key)
		if expandErr != nil {
			return TaskDef{}, expandErr
		}
		task.Env[key] = expanded
	}
	return task, nil
}

func expandRepositoryTemplates(repoIndex int, repo Repository, vars map[string]string) (Repository, error) {
	urlValue, err := expandVarsTemplate(repo.URL, vars, fmt.Sprintf("repositories[%d].url", repoIndex))
	if err != nil {
		return Repository{}, err
	}
	repo.URL = urlValue

	for fileIndex, file := range repo.Files {
		expandedFile, expandErr := expandRepositoryFileTemplates(repoIndex, fileIndex, file, vars)
		if expandErr != nil {
			return Repository{}, expandErr
		}
		repo.Files[fileIndex] = expandedFile
	}
	return repo, nil
}

func expandRepositoryFileTemplates(repoIndex, fileIndex int, file RepositoryFile, vars map[string]string) (RepositoryFile, error) {
	fileName, fileNameErr := expandVarsTemplate(
		file.FileName,
		vars,
		fmt.Sprintf("repositories[%d].files[%d].file_name", repoIndex, fileIndex),
	)
	if fileNameErr != nil {
		return RepositoryFile{}, fileNameErr
	}
	file.FileName = fileName

	outDir, outDirErr := expandVarsTemplate(
		file.OutDir,
		vars,
		fmt.Sprintf("repositories[%d].files[%d].out_dir", repoIndex, fileIndex),
	)
	if outDirErr != nil {
		return RepositoryFile{}, outDirErr
	}
	file.OutDir = outDir

	rename, renameErr := expandVarsTemplate(
		file.Rename,
		vars,
		fmt.Sprintf("repositories[%d].files[%d].rename", repoIndex, fileIndex),
	)
	if renameErr != nil {
		return RepositoryFile{}, renameErr
	}
	file.Rename = rename
	return file, nil
}

func IsRemoteConfigLocation(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func ValidateTaskConfig(cfg *TaskConfig) error {
	if cfg.Version != DefaultTaskConfigVersion {
		return fmt.Errorf("unsupported config version %d (supported: %d)", cfg.Version, DefaultTaskConfigVersion)
	}
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
	return BuildSyncConfigWithOptions(taskCfg, BuildSyncConfigOptions{
		ExpandRepositoryHeaderEnv: true,
	})
}

func BuildSyncConfigWithOptions(taskCfg *TaskConfig, opts BuildSyncConfigOptions) (*SyncConfig, error) {
	NormalizeTaskConfig(taskCfg)
	if err := ExpandTaskConfigTemplates(taskCfg); err != nil {
		return nil, err
	}
	if err := ValidateTaskConfig(taskCfg); err != nil {
		return nil, err
	}

	cfg := &SyncConfig{
		Version: SyncConfigVersion,
		Sources: map[string]Source{},
	}

	for repoIndex, repo := range taskCfg.Repositories {
		if strings.TrimSpace(repo.URL) == "" {
			return nil, fmt.Errorf("repositories[%d].url is required", repoIndex)
		}
		if opts.ExpandRepositoryHeaderEnv {
			resolvedHeaders, err := expandRepositoryHeaders(repo.Headers, repoIndex)
			if err != nil {
				return nil, err
			}
			repo.Headers = resolvedHeaders
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

func expandVarsTemplate(value string, vars map[string]string, fieldPath string) (string, error) {
	if value == "" {
		return value, nil
	}

	missing := map[string]struct{}{}
	expanded := varsTemplatePattern.ReplaceAllStringFunc(value, func(match string) string {
		matches := varsTemplatePattern.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		name := matches[1]
		resolved, ok := vars[name]
		if !ok {
			missing[name] = struct{}{}
			return ""
		}
		return resolved
	})

	if len(missing) > 0 {
		return "", fmt.Errorf("%s references undefined var(s): %s", fieldPath, strings.Join(sortedSetKeys(missing), ", "))
	}

	invalidKeys := map[string]struct{}{}
	refs := varsReferencePattern.FindAllStringSubmatch(expanded, -1)
	for _, ref := range refs {
		if len(ref) < 2 {
			continue
		}
		key := ref[1]
		if varsKeyPattern.MatchString(key) {
			continue
		}
		invalidKeys[key] = struct{}{}
	}
	if len(invalidKeys) > 0 {
		return "", fmt.Errorf(
			"%s references invalid var key(s): %s (allowed pattern: [A-Za-z_][A-Za-z0-9_]*)",
			fieldPath,
			strings.Join(sortedSetKeys(invalidKeys), ", "),
		)
	}
	return expanded, nil
}

func sortedSetKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func expandRepositoryHeaders(headers map[string]string, repoIndex int) (map[string]string, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	expanded := make(map[string]string, len(headers))
	for key, rawValue := range headers {
		value, err := expandHeaderValue(rawValue)
		if err != nil {
			return nil, fmt.Errorf("repositories[%d].headers[%q] %w", repoIndex, key, err)
		}
		expanded[key] = value
	}
	return expanded, nil
}

func expandHeaderValue(rawValue string) (string, error) {
	missing := map[string]struct{}{}
	expanded := headerEnvPattern.ReplaceAllStringFunc(rawValue, func(match string) string {
		name := match[2 : len(match)-1]
		value, ok := os.LookupEnv(name)
		if !ok {
			missing[name] = struct{}{}
			return ""
		}
		return value
	})
	if len(missing) == 0 {
		return expanded, nil
	}
	return "", fmt.Errorf("references undefined environment variable(s): %s", strings.Join(sortedSetKeys(missing), ", "))
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
		DownloadChecksum: "",
		OutputChecksum:   "",
		Encoding:         encoding,
		Extract:          extract,
		ExpandArchive:    expandArchive,
	}
	downloadChecksum, err := normalizeDigest(file.DownloadDigest)
	if err != nil {
		return "", Source{}, FileRule{}, fmt.Errorf("repositories[%d].files[%d].download_digest %w", repoIndex, fileIndex, err)
	}
	outputChecksum, err := normalizeDigest(file.OutputDigest)
	if err != nil {
		return "", Source{}, FileRule{}, fmt.Errorf("repositories[%d].files[%d].output_digest %w", repoIndex, fileIndex, err)
	}
	rule.DownloadChecksum = downloadChecksum
	rule.OutputChecksum = outputChecksum
	if rule.ExpandArchive {
		rule.Mode = ""
	}
	if rule.ExpandArchive && rule.OutputChecksum != "" {
		return "", Source{}, FileRule{}, fmt.Errorf(
			"repositories[%d].files[%d].output_digest cannot be used when extract is omitted for archive encodings",
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
