package keyring

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/open-cli-collective/atlassian-go/credstore"
)

// ClearPlan is the non-secret preview of what `config clear` would do.
// It is computed from KEYRING STATE ONLY (which bundle keys actually
// exist) — never the env-first runtime resolver, because environment
// variables cannot be cleared and must not drive deletion.
type ClearPlan struct {
	Ref string

	// ToolKey is the single key a default (non --all) clear deletes for
	// this tool: its override (<tool>_api_token) if that key exists,
	// otherwise the shared api_token if THAT exists, otherwise "".
	ToolKey string

	// SharedDefault is true when ToolKey == api_token — deleting it also
	// de-authenticates the sibling tool.
	SharedDefault bool

	// ExistingKeys are all bundle keys currently holding a value (the
	// --all blast radius).
	ExistingKeys []string

	// EnvActive lists the token env vars currently set for this tool;
	// they still override at runtime and clear cannot remove them.
	EnvActive []string

	// SharedConfigPath / LegacyPaths are the extra plaintext files --all
	// removes/scrubs (only those that exist are listed).
	SharedConfigPath string
	LegacyPaths      []string
}

// PlanClear computes the ClearPlan for tool via the non-migrating path.
func PlanClear(tool string) (ClearPlan, error) {
	p := ClearPlan{Ref: Ref}

	for _, name := range envVarsFor(tool) {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			p.EnvActive = append(p.EnvActive, name)
		}
	}

	s, err := OpenNoMigrate()
	if err != nil {
		return p, err
	}
	defer func() { _ = s.Close() }()

	existing, err := s.ExistingKeys()
	if err != nil {
		return p, err
	}
	p.ExistingKeys = existing

	has := func(k string) bool {
		for _, e := range existing {
			if e == k {
				return true
			}
		}
		return false
	}
	if ok := KeyFor(tool); ok != "" && has(ok) {
		p.ToolKey = ok
	} else if has(KeyAPIToken) {
		p.ToolKey = KeyAPIToken
		p.SharedDefault = true
	}

	if sp := credstore.DefaultPath(); fileExists(sp) {
		p.SharedConfigPath = sp
	}
	for _, lp := range []string{credstore.LegacyCFLPath(), credstore.LegacyJTKPath()} {
		if lp != "" && fileExists(lp) {
			p.LegacyPaths = append(p.LegacyPaths, lp)
		}
	}
	return p, nil
}

// ClearKey deletes one already-resolved bundle key (idempotent; an empty
// key is a no-op). The caller passes the key from a ClearPlan it already
// computed, so this performs exactly ONE keyring open — avoiding both the
// PlanClear→delete TOCTOU window and a second file-backend passphrase
// prompt.
func ClearKey(key string) error {
	if key == "" {
		return nil
	}
	s, err := OpenNoMigrate()
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	return s.DeleteToken(key)
}

// ClearAll is the explicit destructive path: the whole keyring bundle
// plus the shared non-secret config file, plus a scrub of any surviving
// pre-migration legacy plaintext files.
func ClearAll() error {
	s, err := OpenNoMigrate()
	if err != nil {
		return err
	}
	if err := s.ClearBundle(); err != nil {
		_ = s.Close()
		return err
	}
	_ = s.Close()

	if sp := credstore.DefaultPath(); fileExists(sp) {
		if err := os.Remove(sp); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := scrubLegacyFile(credstore.LegacyCFLPath()); err != nil {
		return err
	}
	return scrubLegacyFile(credstore.LegacyJTKPath())
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// scrubLegacyFile removes api_token from a surviving legacy plaintext
// file in place, preserving non-secret fields. Absent file is a no-op.
// The codec is derived from the file EXTENSION (.json → JSON, otherwise
// YAML — cfl uses .yml, jtk .json) rather than a positional bool, so a
// reordered or new call site cannot silently apply the wrong parser.
func scrubLegacyFile(path string) error {
	if path == "" || !fileExists(path) {
		return nil
	}
	data, err := os.ReadFile(path) //nolint:gosec // scrubbing the tool's own legacy config
	if err != nil {
		return err
	}
	m := map[string]any{}
	isJSON := strings.HasSuffix(path, ".json")
	unmarshal := yaml.Unmarshal
	if isJSON {
		unmarshal = json.Unmarshal
	}
	if err := unmarshal(data, &m); err != nil {
		// Destructive --all path: refuse to claim success while a
		// possibly-plaintext token may still sit in an unparseable file.
		// Name the exact path so the user can remove it themselves.
		return fmt.Errorf(
			"legacy file %s is unparseable and was NOT scrubbed; it may still contain a plaintext api_token — remove it manually: %w",
			path, err)
	}
	if _, ok := m["api_token"]; !ok {
		return nil
	}
	delete(m, "api_token")
	var out []byte
	if isJSON {
		out, err = json.MarshalIndent(m, "", "  ")
	} else {
		out, err = yaml.Marshal(m)
	}
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil { //nolint:gosec // G306: 0600 is correct for a config file
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
