package manifest

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

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
	if cfg.Version == "" {
		cfg.Version = "v1"
	}
	if cfg.Tasks == nil {
		cfg.Tasks = map[string]TaskDef{}
	}
	return &cfg, nil
}

func LoadSyncConfigFromPath(path string) (*SyncConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg SyncConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.Version == "" {
		cfg.Version = "v1"
	}
	if cfg.Sources == nil {
		cfg.Sources = map[string]Source{}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return &cfg, nil
}

func ResolveSyncConfig(taskCfg *TaskConfig, taskConfigPath string) (*SyncConfig, error) {
	if taskCfg.Sync.Inline != nil {
		cfg := taskCfg.Sync.Inline
		if cfg.Version == "" {
			cfg.Version = "v1"
		}
		if cfg.Sources == nil {
			cfg.Sources = map[string]Source{}
		}
		if cfg.Profiles == nil {
			cfg.Profiles = map[string]Profile{}
		}
		return cfg, nil
	}

	baseDir := filepath.Dir(taskConfigPath)
	if taskCfg.Sync.Ref != "" {
		if isHTTP(taskCfg.Sync.Ref) {
			return loadSyncConfigFromURL(taskCfg.Sync.Ref)
		}
		path := taskCfg.Sync.Ref
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		return LoadSyncConfigFromPath(path)
	}

	fallback := filepath.Join(baseDir, "sync.yaml")
	if _, err := os.Stat(fallback); err == nil {
		return LoadSyncConfigFromPath(fallback)
	}
	if err := validateTaskConfig(taskCfg); err != nil {
		return nil, err
	}
	return &SyncConfig{Version: "v1", Sources: map[string]Source{}, Profiles: map[string]Profile{}}, nil
}

func validateTaskConfig(cfg *TaskConfig) error {
	for name, task := range cfg.Tasks {
		if task.Run == "" && len(task.DependsOn) == 0 {
			return fmt.Errorf("task %q must have run or depends_on", name)
		}
	}
	return nil
}

func loadSyncConfigFromURL(rawURL string) (*SyncConfig, error) {
	resp, err := http.Get(rawURL) // #nosec G107
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sync ref download failed: status=%d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var cfg SyncConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.Version == "" {
		cfg.Version = "v1"
	}
	if cfg.Sources == nil {
		cfg.Sources = map[string]Source{}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return &cfg, nil
}

func isHTTP(v string) bool {
	u, err := url.Parse(v)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func ValidateSyncConfig(cfg *SyncConfig) error {
	for id, src := range cfg.Sources {
		if src.Type == "" {
			return fmt.Errorf("source %q type is required", id)
		}
		if src.Type != "http" {
			return fmt.Errorf("source %q type %q is not supported", id, src.Type)
		}
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
	for profileName, profile := range cfg.Profiles {
		for i, rule := range profile.Files {
			if rule.Source == "" {
				return fmt.Errorf("profiles.%s.files[%d].source is required", profileName, i)
			}
			if _, ok := cfg.Sources[rule.Source]; !ok {
				return fmt.Errorf("profiles.%s.files[%d].source %q not found in sources", profileName, i, rule.Source)
			}
			if rule.Path == "" {
				return fmt.Errorf("profiles.%s.files[%d].path is required", profileName, i)
			}
		}
	}
	return nil
}

func ResolveProfileFiles(cfg *SyncConfig, profile string) ([]FileRule, error) {
	if profile == "" {
		return cfg.Files, nil
	}
	p, ok := cfg.Profiles[profile]
	if !ok {
		return nil, errors.New("profile not found")
	}
	merged := make([]FileRule, 0, len(cfg.Files)+len(p.Files))
	merged = append(merged, cfg.Files...)
	merged = append(merged, p.Files...)
	return merged, nil
}
