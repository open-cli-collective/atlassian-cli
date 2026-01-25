// Package config provides configuration management for cfl.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the cfl configuration.
type Config struct {
	URL          string `yaml:"url"`
	Email        string `yaml:"email"`
	APIToken     string `yaml:"api_token"`
	DefaultSpace string `yaml:"default_space,omitempty"`
	OutputFormat string `yaml:"output_format,omitempty"`
}

// Validate checks that all required fields are present and valid.
func (c *Config) Validate() error {
	if c.URL == "" {
		return errors.New("url is required")
	}
	if c.Email == "" {
		return errors.New("email is required")
	}
	if c.APIToken == "" {
		return errors.New("api_token is required")
	}

	// Validate URL scheme
	if !strings.HasPrefix(c.URL, "https://") {
		return errors.New("url must use https")
	}

	return nil
}

// NormalizeURL ensures the URL has the /wiki suffix for Confluence Cloud.
func (c *Config) NormalizeURL() {
	c.URL = strings.TrimSuffix(c.URL, "/")
	if !strings.HasSuffix(c.URL, "/wiki") {
		c.URL = c.URL + "/wiki"
	}
}

// LoadFromEnv loads configuration from environment variables.
// Environment variables override existing values only if set and non-empty.
// Precedence: CFL_* → ATLASSIAN_* → existing config value
func (c *Config) LoadFromEnv() {
	if url := getEnvWithFallback("CFL_URL", "ATLASSIAN_URL"); url != "" {
		c.URL = url
	}
	if email := getEnvWithFallback("CFL_EMAIL", "ATLASSIAN_EMAIL"); email != "" {
		c.Email = email
	}
	if token := getEnvWithFallback("CFL_API_TOKEN", "ATLASSIAN_API_TOKEN"); token != "" {
		c.APIToken = token
	}
	if space := os.Getenv("CFL_DEFAULT_SPACE"); space != "" {
		c.DefaultSpace = space
	}
}

// getEnvWithFallback returns the value of the primary env var, or the fallback if primary is empty.
func getEnvWithFallback(primary, fallback string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	return os.Getenv(fallback)
}

// DefaultConfigPath returns the default configuration file path.
func DefaultConfigPath() string {
	// Try XDG config directory first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "cfl", "config.yml")
	}

	// Fall back to ~/.config/cfl/config.yml
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".cfl", "config.yml")
	}

	return filepath.Join(home, ".config", "cfl", "config.yml")
}

// Save writes the configuration to the specified path.
func (c *Config) Save(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restricted permissions (user read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Load reads the configuration from the specified path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// LoadWithEnv loads configuration from file and overrides with environment variables.
func LoadWithEnv(path string) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		// If file doesn't exist, start with empty config
		cfg = &Config{}
	}

	cfg.LoadFromEnv()
	return cfg, nil
}
