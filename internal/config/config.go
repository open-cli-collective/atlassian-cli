package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	configDirName  = "jira-ticket-cli"
	configFileName = "config.json"
	configFileMode = 0600
	configDirMode  = 0700
)

// Config holds the CLI configuration
type Config struct {
	URL      string `json:"url,omitempty"`
	Domain   string `json:"domain,omitempty"` // Deprecated: use URL instead
	Email    string `json:"email"`
	APIToken string `json:"api_token"`
}

// configPath returns the path to the config file
func configPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}
	return filepath.Join(configDir, configDirName, configFileName), nil
}

// Load loads the configuration from file
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save saves the configuration to file
func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, configDirMode); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, configFileMode); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Clear removes the configuration file
func Clear() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	return nil
}

// GetURL returns the Jira URL from config or environment.
// It checks JIRA_URL first, then falls back to constructing from JIRA_DOMAIN for backwards compatibility.
func GetURL() string {
	if v := os.Getenv("JIRA_URL"); v != "" {
		return NormalizeURL(v)
	}
	cfg, err := Load()
	if err != nil {
		return ""
	}
	if cfg.URL != "" {
		return NormalizeURL(cfg.URL)
	}
	// Backwards compatibility: construct URL from domain
	if v := os.Getenv("JIRA_DOMAIN"); v != "" {
		return "https://" + v + ".atlassian.net"
	}
	if cfg.Domain != "" {
		return "https://" + cfg.Domain + ".atlassian.net"
	}
	return ""
}

// GetDomain returns the domain from config or environment.
// Deprecated: Use GetURL instead. This is kept for backwards compatibility.
func GetDomain() string {
	if v := os.Getenv("JIRA_DOMAIN"); v != "" {
		return v
	}
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return cfg.Domain
}

// NormalizeURL ensures the URL has a scheme and no trailing slash.
func NormalizeURL(u string) string {
	if u == "" {
		return ""
	}
	// Add https:// if no scheme
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}
	// Remove trailing slash
	return strings.TrimSuffix(u, "/")
}

// GetEmail returns the email from config or environment
func GetEmail() string {
	if v := os.Getenv("JIRA_EMAIL"); v != "" {
		return v
	}
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return cfg.Email
}

// GetAPIToken returns the API token from config or environment
func GetAPIToken() string {
	if v := os.Getenv("JIRA_API_TOKEN"); v != "" {
		return v
	}
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return cfg.APIToken
}

// IsConfigured returns true if all required config values are set
func IsConfigured() bool {
	return GetURL() != "" && GetEmail() != "" && GetAPIToken() != ""
}

// Path returns the path to the config file
func Path() string {
	path, _ := configPath()
	return path
}
