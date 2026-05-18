package keyring

import (
	"encoding/json"
	"errors"
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

	// ToolKey is the single key a default (non --all) clear deletes: the
	// shared api_token if it exists, otherwise "". There is one key per
	// logical credential (§1.11.10) — deleting it de-authenticates BOTH
	// jtk and cfl.
	ToolKey string

	// ExistingKeys are all bundle keys currently holding a value (the
	// --all blast radius). Empty when the keyring could not be opened.
	ExistingKeys []string

	// EnvActive lists the token env vars currently set for this tool;
	// they still override at runtime and clear cannot remove them.
	EnvActive []string

	// SharedConfigPath / LegacyPaths are the extra plaintext files --all
	// removes/scrubs (only those that exist are listed). Computed without
	// the keyring so `--all` can still preview/clean them when the keyring
	// itself is unopenable (the recovery path).
	SharedConfigPath string
	LegacyPaths      []string
}

// PlanClear computes the ClearPlan for tool and, on success, returns the
// SAME open keyring handle the caller then uses to delete — exactly ONE
// keyring open for the whole `config clear` flow (no PlanClear→delete
// TOCTOU window, and only one file-backend passphrase prompt).
//
// The env vars and the plaintext-file fields are computed WITHOUT the
// keyring and are always populated, even when the keyring open fails:
// `config clear --all` is the intended recovery path and must still be
// able to scrub plaintext artifacts when the keyring is broken. On open
// failure the returned *Store is nil and the error is returned alongside
// the (file/env-populated) plan so the caller can decide: `--all`
// proceeds with file cleanup; a default single-key clear hard-fails.
//
// Store-ownership contract: a NON-NIL returned *Store is open and owned
// by the caller, which MUST Close it (the cobra wrappers do so with a
// `if store != nil { defer store.Close() }`). On every error path the
// store is closed internally and nil is returned, so the rule is simply
// "Close it iff it is non-nil" — never both close-internally and
// transfer.
// all selects the superset allowlist (incl. residual deprecated per-tool
// keys) so `config clear --all` can delete B3 leftovers — the supported
// recovery path after a divergent-deprecated migration conflict. A
// default (non-all) clear uses the strict single-key allowlist.
func PlanClear(tool string, all bool) (ClearPlan, *Store, error) {
	p := ClearPlan{Ref: Ref}

	for _, name := range envVarsFor(tool) {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			p.EnvActive = append(p.EnvActive, name)
		}
	}
	if sp := credstore.DefaultPath(); fileExists(sp) {
		p.SharedConfigPath = sp
	}
	for _, lp := range []string{credstore.LegacyCFLPath(), credstore.LegacyJTKPath()} {
		if lp != "" && fileExists(lp) {
			p.LegacyPaths = append(p.LegacyPaths, lp)
		}
	}

	open := OpenNoMigrate
	if all {
		open = OpenForClearAll
	}
	s, err := open()
	if err != nil {
		return p, nil, err
	}

	existing, err := s.ExistingKeys()
	if err != nil {
		_ = s.Close()
		return p, nil, err
	}
	p.ExistingKeys = existing

	for _, e := range existing {
		if e == KeyAPIToken {
			p.ToolKey = KeyAPIToken
			break
		}
	}

	return p, s, nil
}

// ClearAll is the destructive reset with the safe ordering baked into ONE
// place: plaintext artifacts are scrubbed/removed FIRST (legacy files,
// then the shared non-secret config), and the keyring bundle is cleared
// LAST. If a plaintext scrub fails, the secret therefore still survives
// in the keyring — the safer, recoverable state — rather than the reverse
// (keyring wiped, plaintext token still on disk).
//
// store may be nil (the keyring could not be opened): the plaintext
// cleanup still runs and bundleCleared is reported false so the caller
// can tell the user the bundle was intentionally left intact. Passing the
// already-open store from PlanClear keeps the whole flow to one open.
func ClearAll(store *Store) (bundleCleared bool, err error) {
	if err := ClearFiles(); err != nil {
		return false, err
	}
	if store == nil {
		return false, nil
	}
	if err := store.ClearBundle(); err != nil {
		return false, err
	}
	return true, nil
}

// ClearFiles removes the shared non-secret config file and scrubs any
// surviving pre-migration legacy plaintext files. It is keyring-
// independent so the `--all` recovery path can clean plaintext even when
// the keyring cannot be opened. Every source is attempted even if an
// earlier one fails, and all failures are reported together (errors.Join)
// so a single unparseable file does not hide — or block cleanup of — the
// others; the joined error still names each offending path.
func ClearFiles() error {
	var errs []error
	if err := scrubLegacyFile(credstore.LegacyCFLPath()); err != nil {
		errs = append(errs, err)
	}
	if err := scrubLegacyFile(credstore.LegacyJTKPath()); err != nil {
		errs = append(errs, err)
	}
	if sp := credstore.DefaultPath(); fileExists(sp) {
		if err := os.Remove(sp); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
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
