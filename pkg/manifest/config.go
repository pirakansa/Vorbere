package manifest

import (
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
	if err := validateRepositoryFile(file, repoIndex, fileIndex); err != nil {
		return "", Source{}, FileRule{}, err
	}

	targetName := file.Rename
	if strings.TrimSpace(targetName) == "" {
		targetName = path.Base(file.FileName)
	}
	if targetName == "." || targetName == "/" || targetName == "" {
		return "", Source{}, FileRule{}, fmt.Errorf(
			"repositories[%d].files[%d] could not determine output filename",
			repoIndex, fileIndex,
		)
	}

	targetPath := filepath.Join(os.ExpandEnv(file.OutDir), targetName)
	sourceID := fmt.Sprintf("r%df%d", repoIndex, fileIndex)
	source := Source{
		URL:     joinURL(repo.URL, file.FileName),
		Headers: repo.Headers,
	}
	rule := FileRule{
		Source:   sourceID,
		Path:     targetPath,
		Mode:     file.Mode,
		Checksum: normalizeDigest(file.Digest),
	}

	return sourceID, source, rule, nil
}

func normalizeDigest(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func validateRepositoryFile(file RepositoryFile, repoIndex, fileIndex int) error {
	if strings.TrimSpace(file.FileName) == "" {
		return fmt.Errorf("repositories[%d].files[%d].file_name is required", repoIndex, fileIndex)
	}
	if strings.TrimSpace(file.OutDir) == "" {
		return fmt.Errorf("repositories[%d].files[%d].out_dir is required", repoIndex, fileIndex)
	}
	if strings.TrimSpace(file.ArtifactDigest) != "" {
		return fmt.Errorf("repositories[%d].files[%d].artifact_digest is not supported", repoIndex, fileIndex)
	}
	if strings.TrimSpace(file.Encoding) != "" {
		return fmt.Errorf("repositories[%d].files[%d].encoding is not supported", repoIndex, fileIndex)
	}
	if strings.TrimSpace(file.Extract) != "" {
		return fmt.Errorf("repositories[%d].files[%d].extract is not supported", repoIndex, fileIndex)
	}
	if file.Symlink != nil {
		return fmt.Errorf("repositories[%d].files[%d].symlink is not supported", repoIndex, fileIndex)
	}
	return nil
}

func joinURL(base, fileName string) string {
	base = strings.TrimSpace(base)
	fileName = strings.TrimSpace(fileName)
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(fileName, "/")
}
