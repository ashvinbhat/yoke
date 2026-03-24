// Package config handles yoke configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Config represents the yoke configuration.
type Config struct {
	// Notion integration settings
	Notion NotionConfig `yaml:"notion"`

	// Sync settings
	Sync SyncConfig `yaml:"sync"`

	// Default values
	Defaults DefaultsConfig `yaml:"defaults"`
}

// NotionConfig holds Notion integration settings.
type NotionConfig struct {
	Token        string `yaml:"token"`         // Notion integration token
	DatabaseID   string `yaml:"database_id"`   // Tasks database ID
	AssigneeName string `yaml:"assignee_name"` // Required assignee name for safety
}

// SyncConfig holds sync settings.
type SyncConfig struct {
	Auto     bool   `yaml:"auto"`     // Auto-sync on commands
	Interval string `yaml:"interval"` // Sync interval if auto
}

// DefaultsConfig holds default values for new tasks.
type DefaultsConfig struct {
	Priority int    `yaml:"priority"` // Default priority (1-5)
	Status   string `yaml:"status"`   // Default status
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Notion: NotionConfig{
			Token:        "",
			DatabaseID:   "",
			AssigneeName: "Ashvin Bhat", // Safety: only interact with your tasks
		},
		Sync: SyncConfig{
			Auto:     false,
			Interval: "5m",
		},
		Defaults: DefaultsConfig{
			Priority: 3,
			Status:   "pending",
		},
	}
}

// YokeDir returns the yoke data directory path.
func YokeDir() string {
	if dir := os.Getenv("YOKE_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".yoke")
}

// DBPath returns the database file path.
func DBPath() string {
	return filepath.Join(YokeDir(), "yoke.db")
}

// ConfigPath returns the config file path.
func ConfigPath() string {
	return filepath.Join(YokeDir(), "config.yaml")
}

// EnsureDir creates the yoke directory if it doesn't exist.
func EnsureDir() error {
	dir := YokeDir()
	return os.MkdirAll(dir, 0755)
}

// Exists checks if yoke has been initialized.
func Exists() bool {
	_, err := os.Stat(DBPath())
	return err == nil
}

// Load reads the configuration from disk.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(ConfigPath())
	if os.IsNotExist(err) {
		return cfg, nil // Return defaults if no config file
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand environment variables in config values
	cfg.expandEnvVars()

	return cfg, nil
}

// expandEnvVars expands ${VAR} and $VAR patterns in config values.
func (c *Config) expandEnvVars() {
	c.Notion.Token = expandEnv(c.Notion.Token)
	c.Notion.DatabaseID = expandEnv(c.Notion.DatabaseID)
	c.Notion.AssigneeName = expandEnv(c.Notion.AssigneeName)
}

// expandEnv expands environment variables in a string.
// Supports ${VAR} and $VAR syntax.
func expandEnv(s string) string {
	// First try ${VAR} syntax
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	s = re.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[2 : len(match)-1] // Extract VAR from ${VAR}
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match // Keep original if env var not set
	})

	// Then try $VAR syntax (word boundary)
	re2 := regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	s = re2.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[1:] // Extract VAR from $VAR
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match // Keep original if env var not set
	})

	return s
}

// Save writes the configuration to disk.
func (c *Config) Save() error {
	if err := EnsureDir(); err != nil {
		return fmt.Errorf("failed to create yoke directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(ConfigPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
