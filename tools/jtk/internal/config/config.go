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
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/url"
)

// loadShared returns the shared credential store. Accessors can't
// propagate errors, so on corrupt shared store we warn once on stderr
// (so the user sees something is wrong) and fall through to legacy
// reads. Init has a separate code path that surfaces corruption as a
// hard error and refuses to clobber the file.
func loadShared() *credstore.Store {
	// §3.2 relocation-aware, mutation-free runtime resolver: canonical
	// store, transparent read-fallback to the prior hand-rolled location
	// when only it exists, and on an old↔new divergence the canonical
	// store is returned alongside the error so commands keep working
	// while the conflict is surfaced once. `jtk init` is the fail-loud
	// mutating gate.
	s, err := credstore.LoadSharedRuntime()
	if err != nil {
		warnCorruptSharedOnce(err)
		if s == nil {
			return &credstore.Store{}
		}
	}
	return s
}

var corruptSharedWarnOnce sync.Once

func warnCorruptSharedOnce(err error) {
	corruptSharedWarnOnce.Do(func() {
		if errors.Is(err, credstore.ErrRelocationConflict) {
			// Readable, not a fallback: the canonical config is in use.
			fmt.Fprintf(os.Stderr, "warning: prior and current shared config diverge (%v); using the current config. Run `jtk init` to reconcile.\n", err)
			return
		}
		fmt.Fprintf(os.Stderr, "warning: shared credential store is unreadable (%v); falling back to per-tool config. Run `jtk init` to fix.\n", err)
	})
}

// jtkSection returns the resolved connection Section from the shared
// `default` (§2.2: single-sourced — no per-tool override).
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

	// Save-projection: the API token lives in the OS keyring, never the
	// plaintext config file. Strip it before marshaling so no Save path
	// can persist the secret. Load still parses a legacy api_token so the
	// one-time keyring migration can find it (asymmetric codec).
	toWrite := *cfg
	toWrite.APIToken = ""
	data, err := json.MarshalIndent(&toWrite, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Atomic write (temp + rename) so a crash mid-write never leaves a
	// truncated config. Dir/file modes are already 0700/0600 (the §3
	// on-disk-state standard). On any error the temp file is removed
	// best-effort so a failed save leaves no stale .tmp.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, configFileMode); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("writing config file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("finalizing config file: %w", err)
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
// Precedence: JIRA_URL → ATLASSIAN_URL → shared default → legacy config url → JIRA_DOMAIN → legacy config domain.
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
// Precedence: JIRA_EMAIL → ATLASSIAN_EMAIL → shared default → legacy config email.
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

// ResolveAPIToken is the AUTHORITATIVE runtime token resolver: env
// (JIRA_API_TOKEN → ATLASSIAN_API_TOKEN) then the OS keyring, running the
// one-time §1.8 migration. A keyring error PROPAGATES — it must never be
// folded into an empty token (that would silently de-authenticate every
// command). This is the single migrating entry point; APIClient uses it.
func ResolveAPIToken() (string, error) {
	tok, _, err := keyring.ResolveToken(credstore.ToolJTK)
	return tok, err
}

// GetAPIToken returns the API token via the NON-migrating keyring path
// (env → keyring), swallowing keyring errors to an empty string. It is
// used only by diagnostics (`config show` source column) and the
// IsConfigured gate; the authoritative, error-propagating path is
// ResolveAPIToken. The token is no longer read from the plaintext config
// file or shared store.
func GetAPIToken() string {
	tok, _, err := keyring.ResolveTokenNoMigrate(credstore.ToolJTK)
	if err != nil {
		return ""
	}
	return tok
}

// IsConfigured returns true if the NON-SECRET config is complete and a
// token is resolvable (env or keyring, non-migrating). The token left
// the plaintext config store, so completeness is composed from both
// halves. For bearer auth: URL + Cloud ID + token; for basic: URL +
// email + token.
func IsConfigured() bool {
	if GetAuthMethod() == auth.AuthMethodBearer {
		return GetURL() != "" && GetCloudID() != "" && GetAPIToken() != ""
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
// Precedence: JIRA_AUTH_METHOD → ATLASSIAN_AUTH_METHOD → shared default → legacy config auth_method → "basic"
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
// Precedence: JIRA_CLOUD_ID → ATLASSIAN_CLOUD_ID → shared default → legacy config cloud_id.
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
