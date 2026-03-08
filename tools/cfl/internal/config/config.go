// Package config provides configuration management for cfl.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-cli-collective/atlassian-go/auth"
	sharedconfig "github.com/open-cli-collective/atlassian-go/config"
	"gopkg.in/yaml.v3"
)

// Config holds the cfl configuration.
type Config struct {
	URL          string `yaml:"url"`
	Email        string `yaml:"email"`
	APIToken     string `yaml:"api_token"`
	DefaultSpace string `yaml:"default_space,omitempty"`
	OutputFormat string `yaml:"output_format,omitempty"`
	AuthMethod   string `yaml:"auth_method,omitempty"` // "basic" (default) or "bearer"
	CloudID      string `yaml:"cloud_id,omitempty"`    // Required for bearer auth (gateway URL)
}

// Validate checks that all required fields are present and valid.
// For bearer auth: URL + API token + Cloud ID are required (no email).
// For basic auth: URL + email + API token are required.
func (c *Config) Validate() error {
	if c.URL == "" {
		return errors.New("url is required")
	}
	if c.APIToken == "" {
		return errors.New("api_token is required")
	}

	// Validate auth method if set (empty defaults to basic)
	if c.AuthMethod != "" {
		if err := auth.ValidateAuthMethod(c.AuthMethod); err != nil {
			return fmt.Errorf("config: %w", err)
		}
	}

	if c.AuthMethod == auth.AuthMethodBearer {
		if c.CloudID == "" {
			return errors.New("cloud_id is required for bearer auth")
		}
	} else {
		if c.Email == "" {
			return errors.New("email is required")
		}
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
	if url := sharedconfig.GetEnvWithFallback("CFL_URL", "ATLASSIAN_URL"); url != "" {
		c.URL = url
	}
	if email := sharedconfig.GetEnvWithFallback("CFL_EMAIL", "ATLASSIAN_EMAIL"); email != "" {
		c.Email = email
	}
	if token := sharedconfig.GetEnvWithFallback("CFL_API_TOKEN", "ATLASSIAN_API_TOKEN"); token != "" {
		c.APIToken = token
	}
	if space := os.Getenv("CFL_DEFAULT_SPACE"); space != "" {
		c.DefaultSpace = space
	}
	if method := sharedconfig.GetEnvWithFallback("CFL_AUTH_METHOD", "ATLASSIAN_AUTH_METHOD"); method != "" {
		c.AuthMethod = method
	}
	if cloudID := sharedconfig.GetEnvWithFallback("CFL_CLOUD_ID", "ATLASSIAN_CLOUD_ID"); cloudID != "" {
		c.CloudID = cloudID
	}
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
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Write with restricted permissions (user read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// Load reads the configuration from the specified path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // reading config file by path
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
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
