// Package config manages the jtk configuration file.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/open-cli-collective/atlassian-go/auth"
	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/url"
)

// loadShared returns the shared credential store. Accessors can't
// propagate errors, so on corrupt shared store we warn once on stderr
// (so the user sees something is wrong) and fall through to legacy
// reads. Init has a separate code path that surfaces corruption as a
// hard error and refuses to clobber the file.
func loadShared() *credstore.Store {
	s, err := credstore.Load(credstore.DefaultPath())
	if err != nil {
		warnCorruptSharedOnce(err)
		return &credstore.Store{}
	}
	return s
}

var corruptSharedWarnOnce sync.Once

func warnCorruptSharedOnce(err error) {
	corruptSharedWarnOnce.Do(func() {
		fmt.Fprintf(os.Stderr, "warning: shared credential store is unreadable (%v); falling back to per-tool config. Run `jtk init` to fix.\n", err)
	})
}

// jtkSection returns the resolved Section for jtk merged from default
// and the jtk override.
func jtkSection() credstore.Section {
	return loadShared().Resolve(credstore.ToolJTK)
}

// jtkSectionWithSource returns the resolved value and source for one
// field of the jtk section.
func jtkSectionWithSource(field string) (string, credstore.Source) {
	return loadShared().ResolveWithSource(credstore.ToolJTK, field)
}

const (
	configDirName  = "jira-ticket-cli"
	configFileName = "config.json"
	configFileMode = 0600
	configDirMode  = 0700
)

// Config holds the CLI configuration
type Config struct {
	URL            string `json:"url,omitempty"`
	Domain         string `json:"domain,omitempty"` // Deprecated: use URL instead
	Email          string `json:"email"`
	APIToken       string `json:"api_token"`
	DefaultProject string `json:"default_project,omitempty"`
	AuthMethod     string `json:"auth_method,omitempty"` // "basic" (default) or "bearer"
	CloudID        string `json:"cloud_id,omitempty"`    // Required for bearer auth (gateway URL)
}

// configPath returns the path to the config file
func configPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting config directory: %w", err)
	}
	return filepath.Join(configDir, configDirName, configFileName), nil
}

// Load loads the configuration from file
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path) //nolint:gosec // CLI tool reading its own config file
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
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
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ") //nolint:gosec // config file intentionally stores API token
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, configFileMode); err != nil {
		return fmt.Errorf("writing config file: %w", err)
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
		return fmt.Errorf("removing config file: %w", err)
	}

	return nil
}

// GetURL returns the Jira URL from config or environment.
// Precedence: JIRA_URL → ATLASSIAN_URL → shared jtk override → shared default → legacy config url → JIRA_DOMAIN → legacy config domain.
func GetURL() string {
	if v := os.Getenv("JIRA_URL"); v != "" {
		return url.NormalizeURL(v)
	}
	if v := os.Getenv("ATLASSIAN_URL"); v != "" {
		return url.NormalizeURL(v)
	}
	if v := jtkSection().URL; v != "" {
		return url.NormalizeURL(v)
	}
	cfg, err := Load()
	if err != nil {
		return ""
	}
	if cfg.URL != "" {
		return url.NormalizeURL(cfg.URL)
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

// GetEmail returns the email from config or environment.
// Precedence: JIRA_EMAIL → ATLASSIAN_EMAIL → shared jtk override → shared default → legacy config email.
func GetEmail() string {
	if v := os.Getenv("JIRA_EMAIL"); v != "" {
		return v
	}
	if v := os.Getenv("ATLASSIAN_EMAIL"); v != "" {
		return v
	}
	if v := jtkSection().Email; v != "" {
		return v
	}
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return cfg.Email
}

// GetAPIToken returns the API token from config or environment.
// Precedence: JIRA_API_TOKEN → ATLASSIAN_API_TOKEN → shared jtk override → shared default → legacy config api_token.
func GetAPIToken() string {
	if v := os.Getenv("JIRA_API_TOKEN"); v != "" {
		return v
	}
	if v := os.Getenv("ATLASSIAN_API_TOKEN"); v != "" {
		return v
	}
	if v := jtkSection().APIToken; v != "" {
		return v
	}
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return cfg.APIToken
}

// IsConfigured returns true if all required config values are set.
// For bearer auth: URL + API token + Cloud ID are required (no email).
// For basic auth: URL + email + API token are required.
func IsConfigured() bool {
	if GetAuthMethod() == auth.AuthMethodBearer {
		return GetURL() != "" && GetAPIToken() != "" && GetCloudID() != ""
	}
	return GetURL() != "" && GetEmail() != "" && GetAPIToken() != ""
}

// GetAuthMethod returns the auth method from config or environment.
// Precedence: JIRA_AUTH_METHOD → ATLASSIAN_AUTH_METHOD → config auth_method → "basic"
// Invalid values are ignored and fall through to the next source.
func GetAuthMethod() string {
	v, _ := GetAuthMethodWithSource()
	return v
}

// GetAuthMethodWithSource returns the auth method and its source.
// Precedence: JIRA_AUTH_METHOD → ATLASSIAN_AUTH_METHOD → shared jtk override → shared default → legacy config auth_method → "basic"
// Invalid values are skipped and fall through to the next source.
// Validation happens at entry points (api.New, init --auth-method) not here.
func GetAuthMethodWithSource() (value, source string) {
	if v := os.Getenv("JIRA_AUTH_METHOD"); v != "" {
		if auth.ValidateAuthMethod(v) == nil {
			return v, "env (JIRA_AUTH_METHOD)"
		}
	}
	if v := os.Getenv("ATLASSIAN_AUTH_METHOD"); v != "" {
		if auth.ValidateAuthMethod(v) == nil {
			return v, "env (ATLASSIAN_AUTH_METHOD)"
		}
	}
	if v, src := jtkSectionWithSource("auth_method"); v != "" && auth.ValidateAuthMethod(v) == nil {
		return v, string(src)
	}
	cfg, err := Load()
	if err != nil {
		return auth.AuthMethodBasic, "default"
	}
	if cfg.AuthMethod != "" {
		if auth.ValidateAuthMethod(cfg.AuthMethod) == nil {
			return cfg.AuthMethod, "config"
		}
	}
	return auth.AuthMethodBasic, "default"
}

// GetCloudID returns the Atlassian Cloud ID from config or environment.
// Precedence: JIRA_CLOUD_ID → ATLASSIAN_CLOUD_ID → config cloud_id
func GetCloudID() string {
	v, _ := GetCloudIDWithSource()
	return v
}

// GetCloudIDWithSource returns the Cloud ID and its source.
// Precedence: JIRA_CLOUD_ID → ATLASSIAN_CLOUD_ID → shared jtk override → shared default → legacy config cloud_id.
func GetCloudIDWithSource() (value, source string) {
	if v := os.Getenv("JIRA_CLOUD_ID"); v != "" {
		return v, "env (JIRA_CLOUD_ID)"
	}
	if v := os.Getenv("ATLASSIAN_CLOUD_ID"); v != "" {
		return v, "env (ATLASSIAN_CLOUD_ID)"
	}
	if v, src := jtkSectionWithSource("cloud_id"); v != "" {
		return v, string(src)
	}
	cfg, err := Load()
	if err != nil {
		return "", "-"
	}
	if cfg.CloudID != "" {
		return cfg.CloudID, "config"
	}
	return "", "-"
}

// GetDefaultProject returns the default project from config or environment.
// Precedence: JIRA_DEFAULT_PROJECT → shared jtk.default_project → legacy config default_project.
func GetDefaultProject() string {
	if v := os.Getenv("JIRA_DEFAULT_PROJECT"); v != "" {
		return v
	}
	if v := loadShared().JTK.DefaultProject; v != "" {
		return v
	}
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return cfg.DefaultProject
}

// Path returns the path to the config file
func Path() string {
	path, _ := configPath()
	return path
}
