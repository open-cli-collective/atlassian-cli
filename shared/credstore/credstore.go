// Package credstore reads and writes the shared Atlassian NON-SECRET
// config at ~/.config/atlassian-cli/config.yml. The store has a single
// "default" section that both cfl and jtk consume, plus optional
// "cfl" and "jtk" sections that hold per-tool overrides (full or partial).
//
// The API token is NOT persisted here — it lives in the OS keyring via
// the sibling shared/keyring package. The codec is intentionally
// asymmetric: Load still READS a legacy api_token (it is the one-time
// migration source) but Save NEVER writes one (see Store.MarshalYAML).
//
// Field resolution is per-field merge (tool override beats default, any
// unset field falls through), so a partial override — say, a different
// cloud_id for cfl — is fully supported.
package credstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

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

// ToolSection embeds Section and adds per-tool defaults that aren't
// shareable (e.g., default_space is meaningless to jtk).
type ToolSection struct {
	Section        `yaml:",inline"`
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

// DefaultPath returns the canonical shared store path, honoring
// $XDG_CONFIG_HOME if set.
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "atlassian-cli", "config.yml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".atlassian-cli", "config.yml")
	}
	return filepath.Join(home, ".config", "atlassian-cli", "config.yml")
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
	c.CFL.APIToken = ""
	c.JTK.APIToken = ""
	return alias(c), nil
}

func (s *Store) toolSection(tool string) ToolSection {
	switch tool {
	case ToolCFL:
		return s.CFL
	case ToolJTK:
		return s.JTK
	default:
		return ToolSection{}
	}
}

// Resolve returns the effective credentials for tool by merging the
// tool's override section over default per field. Unset override fields
// fall through to default; default fields fall through to zero.
func (s *Store) Resolve(tool string) Section {
	d := s.Default
	o := s.toolSection(tool).Section
	out := Section{
		URL:        firstNonEmpty(o.URL, d.URL),
		Email:      firstNonEmpty(o.Email, d.Email),
		APIToken:   firstNonEmpty(o.APIToken, d.APIToken),
		AuthMethod: firstNonEmpty(o.AuthMethod, d.AuthMethod),
		CloudID:    firstNonEmpty(o.CloudID, d.CloudID),
	}
	return out
}

// Source describes where a resolved field came from. Used by
// `config show` to render an audit-friendly source column.
type Source string

const (
	SourceUnset       Source = "unset"
	SourceDefault     Source = "shared default"
	SourceOverrideCFL Source = "shared cfl override"
	SourceOverrideJTK Source = "shared jtk override"
)

// ResolveWithSource returns the resolved value and where it came from.
// Field is the YAML field name (url, email, api_token, auth_method, cloud_id).
func (s *Store) ResolveWithSource(tool, field string) (string, Source) {
	o := s.toolSection(tool).Section
	d := s.Default
	get := func(sec Section) string {
		switch field {
		case "url":
			return sec.URL
		case "email":
			return sec.Email
		case "api_token":
			return sec.APIToken
		case "auth_method":
			return sec.AuthMethod
		case "cloud_id":
			return sec.CloudID
		}
		return ""
	}
	if v := get(o); v != "" {
		switch tool {
		case ToolCFL:
			return v, SourceOverrideCFL
		case ToolJTK:
			return v, SourceOverrideJTK
		default:
			return v, SourceUnset
		}
	}
	if v := get(d); v != "" {
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
