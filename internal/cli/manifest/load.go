package manifest

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadTaskConfig(path string) (*TaskConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg TaskConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	normalizeTaskConfig(&cfg)
	return &cfg, nil
}

func ResolveSyncConfig(taskCfg *TaskConfig, taskConfigPath string) (*SyncConfig, error) {
	if err := validateTaskConfig(taskCfg); err != nil {
		return nil, err
	}
	return buildSyncConfig(taskCfg)
}

func validateTaskConfig(cfg *TaskConfig) error {
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

func normalizeTaskConfig(cfg *TaskConfig) {
	if cfg.Version == 0 {
		cfg.Version = 3
	}
	if cfg.Tasks == nil {
		cfg.Tasks = map[string]TaskDef{}
	}
	if cfg.Repositories == nil {
		cfg.Repositories = []Repository{}
	}
}

func buildSyncConfig(taskCfg *TaskConfig) (*SyncConfig, error) {
	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{},
	}

	for repoIndex, repo := range taskCfg.Repositories {
		if strings.TrimSpace(repo.URL) == "" {
			return nil, fmt.Errorf("repositories[%d].url is required", repoIndex)
		}
		for fileIndex, file := range repo.Files {
			if err := validateRepositoryFile(file, repoIndex, fileIndex); err != nil {
				return nil, err
			}
			targetName := file.Rename
			if strings.TrimSpace(targetName) == "" {
				targetName = path.Base(file.FileName)
			}
			if targetName == "." || targetName == "/" || targetName == "" {
				return nil, fmt.Errorf("repositories[%d].files[%d] could not determine output filename", repoIndex, fileIndex)
			}
			expandedOutDir := os.ExpandEnv(file.OutDir)
			targetPath := filepath.Join(expandedOutDir, targetName)
			sourceID := fmt.Sprintf("r%df%d", repoIndex, fileIndex)
			cfg.Sources[sourceID] = Source{
				URL:     joinURL(repo.URL, file.FileName),
				Headers: repo.Headers,
			}
			rule := FileRule{
				Source: sourceID,
				Path:   targetPath,
				Mode:   file.Mode,
				Merge:  MergeOverwrite,
			}
			if strings.TrimSpace(file.Digest) != "" {
				rule.Checksum = strings.TrimSpace(strings.ToLower(file.Digest))
			}
			cfg.Files = append(cfg.Files, rule)
		}
	}

	return cfg, nil
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
