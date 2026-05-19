// Package credstore reads and writes the shared Atlassian NON-SECRET
// config at ~/.config/atlassian-cli/config.yml. The store has a single
// "default" section that both cfl and jtk consume for connection
// config, plus optional "cfl" and "jtk" sections that hold ONLY
// non-secret per-tool defaults (default_space/default_project/
// output_format). Per §2.2 (MON-5328) a per-tool section may NOT
// override connection credentials — connection is single-sourced from
// "default" (env still overrides at the caller's runtime layer).
//
// The API token is NOT persisted here — it lives in the OS keyring via
// the sibling shared/keyring package. The codec is intentionally
// asymmetric: Load still READS a legacy default api_token (it is the
// one-time migration source) but Save NEVER writes one (see
// Store.MarshalYAML). Pre-MON-5328 files that still carry per-tool
// connection/token fields are decoded once by the migration projection
// (LoadSharedLegacyProjection), never by the canonical Store.
package credstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/open-cli-collective/cli-common/statedir"

	"github.com/open-cli-collective/atlassian-go/auth"
)

// ToolCFL is the section key for cfl-scoped overrides and defaults.
const ToolCFL = "cfl"

// ToolJTK is the section key for jtk-scoped overrides and defaults.
const ToolJTK = "jtk"

// ErrCorruptStore wraps any failure to parse a shared store. Init
// surfaces this as a hard error and refuses to overwrite the file;
// runtime config-resolution paths warn-and-fall-back instead so a
// corrupt shared file doesn't crash every command.
var ErrCorruptStore = errors.New("credstore: corrupt or unparseable")

// Section holds the credential fields shared across both tools.
type Section struct {
	URL        string `yaml:"url,omitempty"`
	Email      string `yaml:"email,omitempty"`
	APIToken   string `yaml:"api_token,omitempty"`
	AuthMethod string `yaml:"auth_method,omitempty"`
	CloudID    string `yaml:"cloud_id,omitempty"`
}

// ToolSection holds ONLY the non-secret per-tool defaults. Per §2.2
// (MON-5328) a per-tool section may NOT override connection credentials
// (url/email/auth_method/cloud_id) or carry api_token — connection is
// single-sourced from the shared `default` section (env still overrides
// at runtime). Legacy files that still carry per-tool connection fields
// are handled once by the migration projection (see legacy.go), never by
// this canonical struct.
type ToolSection struct {
	DefaultSpace   string `yaml:"default_space,omitempty"`   // cfl
	DefaultProject string `yaml:"default_project,omitempty"` // jtk
	OutputFormat   string `yaml:"output_format,omitempty"`   // cfl
}

// Store is the on-disk representation of the shared credential file.
type Store struct {
	Default Section     `yaml:"default"`
	CFL     ToolSection `yaml:"cfl,omitempty"`
	JTK     ToolSection `yaml:"jtk,omitempty"`
}

// DefaultPath returns the canonical shared store path. It resolves via
// the shared statedir resolver (os.UserConfigDir()/atlassian-cli),
// which honors $XDG_CONFIG_HOME on Linux and returns the OS-native
// config dir on macOS/Windows. A relative or unresolvable
// $XDG_CONFIG_HOME now returns an error (the §1.1 intentional
// tightening) instead of the prior silent cwd-relative
// `./.atlassian-cli` fallback. Existence-check callers treat the error
// as "no shared file".
func DefaultPath() (string, error) {
	dir, err := statedir.Scope{Name: "atlassian-cli"}.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yml"), nil
}

// Load reads the store at path. An absent file returns an empty Store
// with nil error so first-run callers don't have to special-case it.
// A present-but-unreadable or unparseable file returns ErrCorruptStore.
//
// init code paths use this directly so they can refuse to overwrite a
// file we couldn't read. Runtime config-resolution paths (cfl
// LoadWithEnv, jtk's accessors) instead warn-and-fall-back so a
// corrupt shared file doesn't break every command for the user.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path) //nolint:gosec // CLI tool reading its own config
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Store{}, nil
		}
		return nil, fmt.Errorf("%w: reading %s: %s", ErrCorruptStore, path, err.Error())
	}
	var s Store
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("%w: parsing %s: %s", ErrCorruptStore, path, err.Error())
	}
	return &s, nil
}

// Save writes the store atomically (temp + rename). On any error before
// or during rename, the temp file is removed best-effort so a failed
// save never leaves stale .tmp behind. Mode is 0600; parent dir 0700.
func (s *Store) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming %s -> %s: %w", tmp, path, err)
	}
	return nil
}

// MarshalYAML is the write half of the asymmetric codec: it strips every
// api_token before serialization so Save can NEVER persist a secret, even
// if a Store still carries a legacy token freshly read by Load (the
// migration source). Load is unchanged — it must keep reading api_token so
// the one-time keyring migration can find it.
//
// The local alias type has Store's fields but not this method, so the
// returned value marshals through the default path (no recursion).
func (s Store) MarshalYAML() (any, error) {
	type alias Store
	c := s
	c.Default.APIToken = ""
	// Per-tool sections no longer carry api_token (or any connection
	// field) — the struct can't hold them, so nothing to strip there.
	return alias(c), nil
}

// Resolve returns the effective credentials for tool. Per §2.2
// (MON-5328) connection config is single-sourced from `default`; per-tool
// sections no longer override it, so the tool argument no longer affects
// the connection result (kept for signature stability — ~every command
// calls this). Env overrides still apply at the caller's runtime layer.
func (s *Store) Resolve(_ string) Section {
	return s.Default
}

// Source describes where a resolved field came from. Used by
// `config show` to render an audit-friendly source column.
type Source string

const (
	SourceUnset   Source = "unset"
	SourceDefault Source = "shared default"
)

// ResolveWithSource returns the resolved value and where it came from.
// Field is the YAML field name (url, email, api_token, auth_method,
// cloud_id). Per §2.2 (MON-5328) connection config is single-sourced
// from `default`, so the only non-unset source is the shared default
// (the tool argument no longer selects a per-tool override).
func (s *Store) ResolveWithSource(_, field string) (string, Source) {
	d := s.Default
	var v string
	switch field {
	case "url":
		v = d.URL
	case "email":
		v = d.Email
	case "api_token":
		v = d.APIToken
	case "auth_method":
		v = d.AuthMethod
	case "cloud_id":
		v = d.CloudID
	}
	if v != "" {
		return v, SourceDefault
	}
	return "", SourceUnset
}

// HasUsableConfig reports whether the NON-SECRET config for tool is
// complete enough to authenticate once a token is supplied. The api_token
// is no longer part of this store (it lives in the keyring), so callers
// must compose this with keyring.HasToken for full readiness. Basic
// requires url + email; bearer requires url + cloud_id. Empty auth_method
// defaults to basic, matching the rest of the codebase.
func (s *Store) HasUsableConfig(tool string) bool {
	r := s.Resolve(tool)
	method := r.AuthMethod
	if method == "" {
		method = auth.AuthMethodBasic
	}
	switch method {
	case auth.AuthMethodBearer:
		return r.URL != "" && r.CloudID != ""
	case auth.AuthMethodBasic:
		return r.URL != "" && r.Email != ""
	default:
		return false
	}
}

// NormalizeBaseURL strips the "/wiki" suffix and any trailing "/" so
// the shared store always carries the bare instance URL. Idempotent.
func NormalizeBaseURL(raw string) string {
	if raw == "" {
		return ""
	}
	u := strings.TrimRight(raw, "/")
	for strings.HasSuffix(u, "/wiki") {
		u = strings.TrimSuffix(u, "/wiki")
		u = strings.TrimRight(u, "/")
	}
	return u
}

// URLForCFL returns base + "/wiki", refusing to double-append. Always
// produces a /wiki-suffixed URL even when given a base that already
// has trailing slashes or a stray /wiki.
func URLForCFL(base string) string {
	b := NormalizeBaseURL(base)
	if b == "" {
		return ""
	}
	return b + "/wiki"
}
