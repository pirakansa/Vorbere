package manifest

// TaskConfig is the repository-level task configuration in vorbere.yaml.
type TaskConfig struct {
	Version string             `yaml:"version"`
	Tasks   map[string]TaskDef `yaml:"tasks"`
	Sync    SyncRef            `yaml:"sync"`
}

// TaskDef defines one runnable task.
type TaskDef struct {
	Run       string            `yaml:"run"`
	Desc      string            `yaml:"desc"`
	Env       map[string]string `yaml:"env"`
	CWD       string            `yaml:"cwd"`
	DependsOn []string          `yaml:"depends_on"`
}

// SyncRef points to sync configuration inline or by reference.
type SyncRef struct {
	Ref    string      `yaml:"ref"`
	Inline *SyncConfig `yaml:"inline"`
}

// SyncConfig is the sync manifest.
type SyncConfig struct {
	Version  string             `yaml:"version"`
	Sources  map[string]Source  `yaml:"sources"`
	Files    []FileRule         `yaml:"files"`
	Profiles map[string]Profile `yaml:"profiles"`
}

// Profile appends profile-specific file rules.
type Profile struct {
	Files []FileRule `yaml:"files"`
}

// Source defines downloadable resource metadata.
type Source struct {
	Type    string            `yaml:"type"`
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
}

// FileRule defines one target placement operation.
type FileRule struct {
	Source   string `yaml:"source"`
	Path     string `yaml:"path"`
	Mode     string `yaml:"mode"`
	Merge    string `yaml:"merge"`
	Backup   string `yaml:"backup"`
	Checksum string `yaml:"checksum"`
}
