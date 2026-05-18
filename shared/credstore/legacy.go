package credstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LegacyCreds is a minimal projection of either tool's legacy config
// file. Used by init reconciliation to compare against the shared store
// without depending on each tool's full Config struct.
type LegacyCreds struct {
	Path           string // "" if file was absent
	URL            string
	Email          string
	APIToken       string
	AuthMethod     string
	CloudID        string
	DefaultSpace   string // cfl-only
	DefaultProject string // jtk-only
	OutputFormat   string // cfl-only
}

// Section returns the credential fields, with URL normalized to base.
func (l *LegacyCreds) Section() Section {
	return Section{
		URL:        NormalizeBaseURL(l.URL),
		Email:      l.Email,
		APIToken:   l.APIToken,
		AuthMethod: l.AuthMethod,
		CloudID:    l.CloudID,
	}
}

// SharedLegacyConn is one shared-store section's pre-MON-5328
// connection + token fields. Decoded ONLY by the one-time migration so
// it can still see legacy per-tool credentials before the stripped
// schema is written. The canonical Store no longer exposes these — do
// NOT use for runtime resolution.
type SharedLegacyConn struct {
	URL        string `yaml:"url"`
	Email      string `yaml:"email"`
	APIToken   string `yaml:"api_token"`
	AuthMethod string `yaml:"auth_method"`
	CloudID    string `yaml:"cloud_id"`
}

// SharedLegacyProjection is the migration-only decode of the shared
// config.yml retaining the per-tool connection/token fields the
// canonical Store dropped (§2.2 / MON-5328). Migration-only.
type SharedLegacyProjection struct {
	Path    string
	Default SharedLegacyConn
	CFL     SharedLegacyConn
	JTK     SharedLegacyConn
}

// LoadSharedLegacyProjection decodes path retaining legacy per-tool
// connection/token fields. Absent file → (nil, nil). Parse failure →
// ErrCorruptStore (same contract as Load) so callers refuse to clobber
// an unreadable file.
func LoadSharedLegacyProjection(path string) (*SharedLegacyProjection, error) {
	data, err := os.ReadFile(path) //nolint:gosec // CLI tool reading its own config
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: reading %s: %s", ErrCorruptStore, path, err.Error())
	}
	var raw struct {
		Default SharedLegacyConn `yaml:"default"`
		CFL     SharedLegacyConn `yaml:"cfl"`
		JTK     SharedLegacyConn `yaml:"jtk"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: parsing %s: %s", ErrCorruptStore, path, err.Error())
	}
	return &SharedLegacyProjection{Path: path, Default: raw.Default, CFL: raw.CFL, JTK: raw.JTK}, nil
}

// LegacyCFLPath returns the canonical cfl legacy config path.
func LegacyCFLPath() string {
	return tooledPath("cfl", "config.yml")
}

// LegacyJTKPath returns the canonical jtk legacy config path. jtk's
// loader uses os.UserConfigDir(), which on macOS is
// ~/Library/Application Support — matching it here is critical so
// macOS users with an existing jtk config are detected by sibling
// init reconciliation.
func LegacyJTKPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".", "jira-ticket-cli", "config.json")
	}
	return filepath.Join(dir, "jira-ticket-cli", "config.json")
}

func tooledPath(toolDir, file string) string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, toolDir, file)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "."+toolDir, file)
	}
	return filepath.Join(home, ".config", toolDir, file)
}

// LoadLegacyCFL reads a cfl YAML legacy config file. An absent file
// returns (nil, nil). Parse failures return ErrCorruptStore so the
// caller can refuse to clobber it.
func LoadLegacyCFL(path string) (*LegacyCreds, error) {
	data, err := os.ReadFile(path) //nolint:gosec // CLI tool reading its own config
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: reading %s: %s", ErrCorruptStore, path, err.Error())
	}
	var raw struct {
		URL          string `yaml:"url"`
		Email        string `yaml:"email"`
		APIToken     string `yaml:"api_token"`
		DefaultSpace string `yaml:"default_space"`
		OutputFormat string `yaml:"output_format"`
		AuthMethod   string `yaml:"auth_method"`
		CloudID      string `yaml:"cloud_id"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: parsing %s: %s", ErrCorruptStore, path, err.Error())
	}
	return &LegacyCreds{
		Path:         path,
		URL:          raw.URL,
		Email:        raw.Email,
		APIToken:     raw.APIToken,
		AuthMethod:   raw.AuthMethod,
		CloudID:      raw.CloudID,
		DefaultSpace: raw.DefaultSpace,
		OutputFormat: raw.OutputFormat,
	}, nil
}

// LoadLegacyJTK reads a jtk JSON legacy config file. An absent file
// returns (nil, nil). Parse failures return ErrCorruptStore.
func LoadLegacyJTK(path string) (*LegacyCreds, error) {
	data, err := os.ReadFile(path) //nolint:gosec // CLI tool reading its own config
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: reading %s: %s", ErrCorruptStore, path, err.Error())
	}
	var raw struct {
		URL            string `json:"url"`
		Domain         string `json:"domain"`
		Email          string `json:"email"`
		APIToken       string `json:"api_token"`
		DefaultProject string `json:"default_project"`
		AuthMethod     string `json:"auth_method"`
		CloudID        string `json:"cloud_id"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: parsing %s: %s", ErrCorruptStore, path, err.Error())
	}
	url := raw.URL
	if url == "" && raw.Domain != "" {
		url = "https://" + raw.Domain + ".atlassian.net"
	}
	return &LegacyCreds{
		Path:           path,
		URL:            url,
		Email:          raw.Email,
		APIToken:       raw.APIToken,
		AuthMethod:     raw.AuthMethod,
		CloudID:        raw.CloudID,
		DefaultProject: raw.DefaultProject,
	}, nil
}
