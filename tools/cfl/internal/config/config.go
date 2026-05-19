// Package config provides configuration management for cfl.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/open-cli-collective/atlassian-go/auth"
	sharedconfig "github.com/open-cli-collective/atlassian-go/config"
	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"
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

// MarshalYAML strips the API token before serialization so Save can
// never persist the secret to the plaintext legacy file (the token lives
// in the OS keyring). Load/UnmarshalYAML is unchanged — it still parses a
// legacy api_token so the one-time keyring migration can find it. The
// local alias has Config's fields but not this method (no recursion).
func (c Config) MarshalYAML() (any, error) {
	type alias Config
	c.APIToken = ""
	return alias(c), nil
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
	// The API token is intentionally NOT read here: it is resolved
	// exclusively via keyring.ResolveToken (which itself checks
	// CFL_API_TOKEN → ATLASSIAN_API_TOKEN before the keyring). Reading it
	// here too would reintroduce a plaintext path.
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

// Save writes the configuration atomically (temp + rename) so a crash
// mid-write never leaves a truncated config behind. Dir mode is 0700
// and file mode 0600 (the §3 on-disk-state standard). On any error the
// temp file is removed best-effort so a failed save leaves no stale
// .tmp.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("writing config file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("finalizing config file: %w", err)
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

// LoadFromShared layers connection credentials from the shared store's
// `default` section (§2.2: single-sourced — no per-tool override) on
// top of the receiver. URLs from the shared store are stored as base;
// this method appends "/wiki" so the receiver matches cfl's legacy URL
// convention.
func (c *Config) LoadFromShared(s *credstore.Store) {
	if s == nil {
		return
	}
	r := s.Resolve(credstore.ToolCFL)
	if r.URL != "" {
		c.URL = credstore.URLForCFL(r.URL)
	}
	if r.Email != "" {
		c.Email = r.Email
	}
	// r.APIToken is intentionally ignored — the token lives in the
	// keyring, not the shared config store (resolved in LoadWithEnv).
	if r.AuthMethod != "" {
		c.AuthMethod = r.AuthMethod
	}
	if r.CloudID != "" {
		c.CloudID = r.CloudID
	}
	if s.CFL.DefaultSpace != "" {
		c.DefaultSpace = s.CFL.DefaultSpace
	}
	if s.CFL.OutputFormat != "" {
		c.OutputFormat = s.CFL.OutputFormat
	}
}

var corruptSharedWarnOnce sync.Once

func warnCorruptSharedOnce(err error) {
	corruptSharedWarnOnce.Do(func() {
		if errors.Is(err, credstore.ErrRelocationConflict) {
			// Readable, not a fallback: the canonical config is in use.
			fmt.Fprintf(os.Stderr, "warning: prior and current shared config diverge (%v); using the current config. Run `cfl init` to reconcile.\n", err)
			return
		}
		fmt.Fprintf(os.Stderr, "warning: shared credential store is unreadable (%v); falling back to per-tool config. Run `cfl init` to fix.\n", err)
	})
}

// LoadWithEnv loads configuration with full precedence.
//
// Non-secret fields (url, email, auth_method, cloud_id, default_space,
// output_format):
//  1. legacy file (lowest)
//  2. shared store default (§2.2: single-sourced — no per-tool override)
//  3. ATLASSIAN_* env
//  4. CFL_* env (highest)
//
// The API token is resolved separately and authoritatively via
// keyring.ResolveToken (env → OS keyring, running the one-time §1.8
// migration). It is never read from the plaintext legacy file or shared
// store. A keyring error propagates (it must not be folded into an empty
// token, which would silently de-authenticate every command).
//
// A corrupt shared store warns once on stderr and falls back to legacy
// + env so a broken shared file doesn't crash every cfl command. Init
// uses credstore.Load directly so it can surface the error and refuse
// to overwrite.
func LoadWithEnv(path string) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		// Legacy file missing or corrupt: start empty. cfl init has a
		// separate detect-and-reconcile path that distinguishes those
		// cases and refuses to clobber a corrupt legacy file.
		cfg = &Config{}
	}

	// Runtime shared resolver, §3.2 relocation-aware and mutation-free:
	// reads the canonical store, transparently falls back to the prior
	// hand-rolled location when only it exists, and on an old↔new
	// divergence returns the canonical store alongside the error so the
	// command keeps working while the conflict is surfaced once. A nil
	// store (unresolvable/corrupt) → fall back to legacy + env. `cfl
	// init` uses the fail-loud detect/gated-copy path instead.
	store, sErr := credstore.LoadSharedRuntime()
	if sErr != nil {
		warnCorruptSharedOnce(sErr)
	}
	if store != nil {
		cfg.LoadFromShared(store)
	}

	cfg.LoadFromEnv()

	// Authoritative token resolution: overwrites any token a legacy-file
	// parse may have populated, so plaintext can never reach the client.
	tok, _, kErr := keyring.ResolveToken(credstore.ToolCFL)
	if kErr != nil {
		return nil, kErr
	}
	cfg.APIToken = tok
	return cfg, nil
}
