// Package config manages the timerd configuration file and job definitions.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// JobConfig holds the configuration for a single scheduled job.
type JobConfig struct {
	Command     string            `yaml:"command"`
	Args        []string          `yaml:"args,omitempty"`
	WorkDir     string            `yaml:"workdir,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Schedule    string            `yaml:"schedule"`
	Description string            `yaml:"description,omitempty"`
	User        string            `yaml:"user,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	RestartSec  string            `yaml:"restart_sec,omitempty"`
	Timeout     string            `yaml:"timeout,omitempty"`
	OneShot     bool              `yaml:"oneshot,omitempty"`
	Persistent  bool              `yaml:"persistent,omitempty"`
	After       []string          `yaml:"after,omitempty"`
	Wants       []string          `yaml:"wants,omitempty"`
	Requires    []string          `yaml:"requires,omitempty"`
	Enabled     bool              `yaml:"enabled,omitempty"`

	// Shell wraps the command in bash -lc '...' so that shell constructs
	// (pipes, &&, variable expansion) work inside the systemd unit.
	Shell bool `yaml:"shell,omitempty"`

	// InheritEnv controls whether the current process PATH is injected into
	// the unit's Environment= directive. Nil means true (opt-in by default).
	InheritEnv *bool `yaml:"inherit_env,omitempty"`
}

// Config is the top-level configuration structure.
type Config struct {
	Jobs map[string]*JobConfig `yaml:"jobs"`
}

// NewConfig creates a new empty Config.
func NewConfig() *Config {
	return &Config{
		Jobs: make(map[string]*JobConfig),
	}
}

// Load reads and parses a config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s: run 'timerd init' first", path)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := NewConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Jobs == nil {
		cfg.Jobs = make(map[string]*JobConfig)
	}

	return cfg, nil
}

// Save writes the config to disk, creating parent directories if needed.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// AddJob adds a new job to the config, returning an error if it already exists.
func (c *Config) AddJob(name string, job *JobConfig) error {
	if _, exists := c.Jobs[name]; exists {
		return fmt.Errorf("job %q already exists", name)
	}
	c.Jobs[name] = job
	return nil
}

// UpdateJob replaces an existing job config.
func (c *Config) UpdateJob(name string, job *JobConfig) error {
	if _, exists := c.Jobs[name]; !exists {
		return fmt.Errorf("job %q not found", name)
	}
	c.Jobs[name] = job
	return nil
}

// RemoveJob deletes a job from the config.
func (c *Config) RemoveJob(name string) error {
	if _, exists := c.Jobs[name]; !exists {
		return fmt.Errorf("job %q not found", name)
	}
	delete(c.Jobs, name)
	return nil
}

// GetJob retrieves a job by name.
func (c *Config) GetJob(name string) (*JobConfig, error) {
	job, ok := c.Jobs[name]
	if !ok {
		return nil, fmt.Errorf("job %q not found", name)
	}
	return job, nil
}

// Validate checks that all jobs have required fields.
func (c *Config) Validate() error {
	for name, job := range c.Jobs {
		if job.Command == "" {
			return fmt.Errorf("job %q: command is required", name)
		}
		if job.Schedule == "" {
			return fmt.Errorf("job %q: schedule is required", name)
		}
	}
	return nil
}

// DefaultConfigPath returns the default config path based on mode.
func DefaultConfigPath(userMode bool) string {
	if userMode {
		home, err := os.UserHomeDir()
		if err != nil {
			return "/etc/timerd/config.yml"
		}
		return filepath.Join(home, ".config", "timerd", "config.yml")
	}
	return "/etc/timerd/config.yml"
}

// DefaultConfigDir returns the config directory.
func DefaultConfigDir(userMode bool) string {
	return filepath.Dir(DefaultConfigPath(userMode))
}

// DefaultUnitDir returns the systemd unit directory based on mode.
func DefaultUnitDir(userMode bool) string {
	if userMode {
		home, err := os.UserHomeDir()
		if err != nil {
			return "/etc/systemd/system"
		}
		return filepath.Join(home, ".config", "systemd", "user")
	}
	return "/etc/systemd/system"
}
