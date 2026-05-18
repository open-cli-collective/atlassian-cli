package keyring

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"
)

// BackendEnvVar selects the credstore backend (§1.4). Empty = auto-select
// (OS keyring, fail-closed on Linux); "file" = the encrypted-file backend
// (passphrase via ATLASSIAN_CLI_KEYRING_PASSPHRASE or a no-echo TTY
// prompt). There is no config-file backend field — selection is env-only.
const BackendEnvVar = "ATLASSIAN_CLI_KEYRING_BACKEND"

// ErrTokenNotFound indicates no API token exists in the keyring.
var ErrTokenNotFound = errors.New("no API token found in secure storage")

// Store is an open handle to the shared Atlassian credential bundle.
// Construct with an Open* function; always Close.
type Store struct {
	cs      *cccredstore.Store
	service string
	profile string
	ref     string
	allow   []string // effective allowlist this handle was opened with
}

// Open opens the fixed shared ref and runs the one-time §1.8 migration
// (used by API commands, `config test`, and `init`). A legacy-vs-keyring
// effective-value conflict surfaces here as a hard error.
func Open() (*Store, error) { return open(false, true) }

// OpenForClearAll opens the bundle WITHOUT migration but with the
// migration/superset allowlist (so `config clear --all` can delete any
// residual deprecated per-tool key — the supported recovery path when a
// divergent-deprecated state made migration fail loud). Runtime and
// OpenNoMigrate keep the strict single-key allowlist (§1.11.11).
func OpenForClearAll() (*Store, error) { return openWithAllow(false, false, migrationAllowedKeys) }

// (There is intentionally no exported overwrite-migration entry point:
// no user-facing `--overwrite` command exists, so the open(overwrite=…)
// seam is reached only by tests of the pure conflict resolver.)

// OpenNoMigrate opens WITHOUT the one-time migration — diagnostic /
// remediation only (`config show`, `config clear`), so they stay usable
// during an unresolved conflict.
func OpenNoMigrate() (*Store, error) { return open(false, false) }

func open(overwrite, runMigration bool) (*Store, error) {
	// The migration path needs to read and delete the deprecated per-tool
	// keys (B3 upgrade), so it opens with the superset allowlist;
	// non-migrating opens stay strict single-key (§1.11.11).
	allow := allowedKeys
	if runMigration {
		allow = migrationAllowedKeys
	}
	return openWithAllow(overwrite, runMigration, allow)
}

func openWithAllow(overwrite, runMigration bool, allow []string) (*Store, error) {
	s, err := openRef(Ref, allow)
	if err != nil {
		return nil, err
	}
	if runMigration {
		if err := migrateLegacyOverwrite(s, overwrite); err != nil {
			_ = s.cs.Close()
			return nil, err
		}
	}
	return s, nil
}

// openCanonical opens the one fixed shared bundle WITHOUT running the
// migration. Internal-only ingress helper (PersistToken, SetCredential):
// the ref is a compile-time constant — there is no caller-supplied ref
// in the fixed-ref architecture.
func openCanonical() (*Store, error) {
	return openRef(Ref, allowedKeys)
}

func openRef(ref string, allow []string) (*Store, error) {
	service, profile, err := cccredstore.ParseRef(ref)
	if err != nil {
		return nil, fmt.Errorf("invalid credential ref %q: %w", ref, err)
	}
	opts := &cccredstore.Options{AllowedKeys: allow}
	switch b := strings.TrimSpace(os.Getenv(BackendEnvVar)); b {
	case "":
		// Auto-select per §1.4 (credstore decides; fail-closed on Linux).
	case "file":
		opts.ConfigBackend = cccredstore.BackendFile
	default:
		// Fail closed: an unrecognized backend must not silently degrade.
		// Be explicit that the OS keyring is the unset/auto default — a
		// user typing "keychain"/"secret-service" is reaching for what
		// they already get by leaving this unset.
		return nil, fmt.Errorf(
			"invalid %s %q: the only recognized value is \"file\" (opt-in encrypted-file backend); leave %s unset to auto-select the OS keyring (macOS Keychain / Linux Secret Service / Windows Credential Manager)",
			BackendEnvVar, b, BackendEnvVar)
	}
	opts.FilePassphrase = passphraseFunc(service)

	cs, err := cccredstore.Open(service, opts)
	if err != nil {
		return nil, err
	}
	return &Store{cs: cs, service: service, profile: profile, ref: ref, allow: allow}, nil
}

// Close releases the backing store. Safe on a nil receiver.
func (s *Store) Close() error {
	if s == nil || s.cs == nil {
		return nil
	}
	return s.cs.Close()
}

// Ref / Service are non-secret; safe to display.
func (s *Store) Ref() string     { return s.ref }
func (s *Store) Service() string { return s.service }

// Backend reports the credstore backend and how it was selected (§1.6).
func (s *Store) Backend() (cccredstore.Backend, cccredstore.Source) { return s.cs.Backend() }

// Token returns the shared api_token. One key per logical credential
// (§1.11.10): jtk and cfl resolve the same key. Keyring errors propagate
// (never folded into "absent").
func (s *Store) Token() (string, bool, error) {
	return s.get(KeyAPIToken)
}

func (s *Store) get(key string) (string, bool, error) {
	v, err := s.cs.Get(s.profile, key)
	if errors.Is(err, cccredstore.ErrNotFound) || (err == nil && v == "") {
		return "", false, nil
	}
	if err != nil {
		// Never embed the value; naming ref/key/op is allowed (§1.12).
		return "", false, fmt.Errorf("read %s from %s: %w", key, s.ref, err)
	}
	return v, true, nil
}

// SetToken stores a token under an allowlisted key (explicit ingress:
// PersistToken / SetCredential). Ingress is an intentional user action,
// so it overwrites any existing value.
func (s *Store) SetToken(key, val string) error {
	return s.setToken(key, val, true)
}

// setToken is the single guarded write chokepoint. With overwrite=false a
// value created between a caller's read and this write surfaces as
// cccredstore.ErrExists instead of being silently clobbered — the §1.8
// migration relies on this so a concurrent writer can never make it
// re-introduce "pick a winner".
func (s *Store) setToken(key, val string, overwrite bool) error {
	// Enforce the allowlist at the lowest write chokepoint: SetCredential
	// validates earlier (better message), but PersistToken (init) and any
	// future caller reach the keyring only through here, so the security
	// boundary for "what may be stored under the fixed ref" lives in one
	// place rather than relying on each caller to re-check.
	// Intentional asymmetry: writes are ALWAYS restricted to the strict
	// conforming set (allowedKeys = {api_token}), never s.allow — even on
	// a store opened with migrationAllowedKeys. That wider allowlist only
	// exists so the migration / clear-all can READ and DELETE the
	// deprecated per-tool keys to clean them up; nothing may ever WRITE a
	// deprecated key back (§1.11.11). ExistingKeys/DeleteToken use s.allow
	// (read/delete the residue); setToken stays strict (no resurrection).
	if !slices.Contains(allowedKeys, key) {
		return fmt.Errorf("refusing to store under non-allowlisted key %q at %s (allowed: %s)",
			key, s.ref, strings.Join(allowedKeys, ", "))
	}
	// Reject empty values for ALL ingress paths (SetCredential already
	// trims+rejects; this also covers PersistToken).
	if val == "" {
		return fmt.Errorf("refusing to store an empty value at %s/%s", s.ref, key)
	}
	opts := []cccredstore.SetOpt{}
	if overwrite {
		opts = append(opts, cccredstore.WithOverwrite())
	}
	if err := s.cs.Set(s.profile, key, val, opts...); err != nil {
		return fmt.Errorf("store %s at %s: %w", key, s.ref, err)
	}
	return nil
}

// HasToken reports presence of a specific key without returning the value.
// A genuine keyring error is surfaced, not folded into false.
func (s *Store) HasToken(key string) (bool, error) {
	ok, err := s.cs.Exists(s.profile, key)
	if err != nil {
		return false, fmt.Errorf("check %s at %s: %w", key, s.ref, err)
	}
	return ok, nil
}

// DeleteToken removes one key (idempotent: absent is not an error).
func (s *Store) DeleteToken(key string) error {
	ok, err := s.cs.Exists(s.profile, key)
	if err != nil {
		return fmt.Errorf("check %s at %s: %w", key, s.ref, err)
	}
	if !ok {
		return nil
	}
	if err := s.cs.Delete(s.profile, key); err != nil && !errors.Is(err, cccredstore.ErrNotFound) {
		return fmt.Errorf("delete %s at %s: %w", key, s.ref, err)
	}
	return nil
}

// ExistingKeys returns which allowlist keys currently hold a value — used
// by `config clear` to choose what to delete from keyring state ALONE
// (never the env-first resolver: env cannot be cleared).
func (s *Store) ExistingKeys() ([]string, error) {
	var out []string
	for _, k := range s.allow {
		ok, err := s.cs.Exists(s.profile, k)
		if err != nil {
			return nil, fmt.Errorf("check %s at %s: %w", k, s.ref, err)
		}
		if ok {
			out = append(out, k)
		}
	}
	return out, nil
}

// ClearBundle removes every key under the active profile (`config clear
// --all`). Idempotent.
func (s *Store) ClearBundle() error {
	_, err := s.cs.DeleteBundle(s.profile)
	return err
}

// PersistToken stores token under an allowlisted key at the canonical
// shared ref — the in-memory ingress path for `init` (the form already
// holds the token, so there is no io.Reader to read from). No migration
// runs: init calls EnsureMigrated up front, so the §1.8 source is already
// resolved before the new token is written.
func PersistToken(token string) (err error) {
	s, err := openCanonical()
	if err != nil {
		return err
	}
	// Surface the Close error on this WRITE path: the encrypted-file
	// backend may flush/sync on Close, so a swallowed Close error after a
	// "successful" SetToken could mean the token was never durably
	// written. Read-only callers (HasToken, EnsureMigrated) keep the
	// cheap discard — there a Close error changes nothing.
	defer func() {
		if cerr := s.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("persist token: close keyring %s: %w", s.ref, cerr)
		}
	}()
	return s.SetToken(KeyAPIToken, token)
}

// HasToken reports whether the shared api_token is already present in the
// keyring WITHOUT running the migration or consulting env. Used by `init`
// detection to compose readiness with credstore.HasUsableConfig. A
// genuine keyring error is surfaced, never folded into false.
func HasToken() (bool, error) {
	s, err := OpenNoMigrate()
	if err != nil {
		return false, err
	}
	defer func() { _ = s.Close() }()
	_, ok, err := s.Token()
	return ok, err
}

// EnsureMigrated runs (and resolves) the one-time §1.8 migration up front
// via the full Open() path, then closes. Shared by `cfl init` /
// `jtk init` / `set-credential` (default ref) so the migration guarantee
// lives in exactly one place.
func EnsureMigrated() error {
	s, err := Open()
	if err != nil {
		return err
	}
	return s.Close()
}
