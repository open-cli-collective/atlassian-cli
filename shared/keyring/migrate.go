package keyring

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/atlassian-go/credstore"
)

// One-time §1.8 migration (§2.x analog). The access secret is the
// Atlassian api_token. Sources of a legacy plaintext token, in today's
// persisted precedence (env is never persisted; the shared Store outranks
// a legacy per-tool file):
//
//   - shared config.yml: Store.Default / Store.CFL / Store.JTK .APIToken
//   - legacy cfl file (~/.config/cfl/config.yml)
//   - legacy jtk file (os.UserConfigDir()/jira-ticket-cli/config.json)
//
// The migrated value per scope is that scope's CURRENT EFFECTIVE token:
//
//   - api_token      <- Store.Default.APIToken
//   - cfl_api_token  <- cfl's own-scope effective token, written ONLY when
//     it is a genuine override (differs from the effective default and its
//     source outranks the default). A legacy cfl FILE value shadowed by a
//     non-empty shared Store value is dead data: scrubbed, never written,
//     never a conflict. A value equal to the effective default is
//     cleanup-only (no override key — writing it would pin cfl off future
//     default changes).
//   - jtk_api_token  <- symmetric.
//
// All plaintext token bytes are scrubbed afterwards (shared config.yml AND
// the legacy files), whether or not they were written to a key. The only
// fail-loud is a genuine EFFECTIVE value disagreeing with an existing
// keyring value (cross-tool idempotency: equal => silent no-op + scrub).

// ErrMigrationConflict is the stable identity for a §1.8 conflict.
var ErrMigrationConflict = errors.New("keyring: legacy plaintext API token conflicts with the existing keyring value")

func migrateLegacyOverwrite(s *Store, overwrite bool) error {
	sharedPath := credstore.DefaultPath()
	store, err := credstore.Load(sharedPath)
	if err != nil {
		// A corrupt shared store must not be silently overwritten; surface
		// it (callers treat ErrCorruptStore as a hard error).
		return err
	}
	cflPath := credstore.LegacyCFLPath()
	jtkPath := credstore.LegacyJTKPath()
	legacyCFL, errC := credstore.LoadLegacyCFL(cflPath)
	if errC != nil {
		return errC
	}
	legacyJTK, errJ := credstore.LoadLegacyJTK(jtkPath)
	if errJ != nil {
		return errJ
	}

	want, locs, anyPlaintext := gatherEffective(store, legacyCFL, legacyJTK)

	// Current keyring values for the allowlist.
	current := map[string]string{}
	for _, k := range allowedKeys {
		if v, ok, gerr := s.get(k); gerr != nil {
			return gerr
		} else if ok {
			current[k] = v
		}
	}

	toWrite, conflicts := planWrites(want, current, overwrite)
	if len(conflicts) > 0 {
		return conflictError(conflicts, locs, s.ref)
	}

	if len(toWrite) > 0 {
		if _, err := s.cs.SetBundle(s.profile, toWrite, cccredstore.WithOverwrite()); err != nil {
			return fmt.Errorf("migrate API token to keyring %s: %w", s.ref, err)
		}
	}

	if !anyPlaintext {
		return nil // steady state — nothing legacy on disk
	}

	// Scrub every plaintext source (whether or not it fed a key). Failure
	// here is real: we wrote the keyring but could not remove the
	// plaintext — surface it so the user knows the secret still lingers.
	if err := scrubSharedStore(store, sharedPath); err != nil {
		return fmt.Errorf("migrated to keyring %s but could not scrub %s: %w", s.ref, sharedPath, err)
	}
	if legacyCFL != nil && legacyCFL.APIToken != "" {
		if err := scrubLegacyCFL(legacyCFL); err != nil {
			return fmt.Errorf("migrated to keyring %s but could not scrub %s: %w", s.ref, cflPath, err)
		}
	}
	if legacyJTK != nil && legacyJTK.APIToken != "" {
		if err := scrubLegacyJTK(legacyJTK); err != nil {
			return fmt.Errorf("migrated to keyring %s but could not scrub %s: %w", s.ref, jtkPath, err)
		}
	}

	if len(toWrite) > 0 {
		keys := make([]string, 0, len(toWrite))
		for k := range toWrite {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		recordMigration(fmt.Sprintf(
			"atlassian-cli: migrated the API token (%s) into the OS keyring %s; the plaintext copy was removed",
			strings.Join(keys, ", "), s.ref))
	}
	return nil
}

// gatherEffective computes the genuine per-scope key writes, a non-secret
// location string per key (for conflict messages), and whether ANY
// plaintext token existed anywhere (drives scrub even when every value is
// shadowed/cleanup-only).
func gatherEffective(store *credstore.Store, lc, lj *credstore.LegacyCreds) (want, locs map[string]string, anyPlaintext bool) {
	want = map[string]string{}
	locs = map[string]string{}

	effDefault := store.Default.APIToken
	if effDefault != "" {
		want[KeyAPIToken] = effDefault
		locs[KeyAPIToken] = "shared config default"
	}

	lcTok, ljTok := "", ""
	if lc != nil {
		lcTok = lc.APIToken
	}
	if lj != nil {
		ljTok = lj.APIToken
	}
	if effDefault != "" || store.CFL.APIToken != "" || store.JTK.APIToken != "" ||
		lcTok != "" || ljTok != "" {
		anyPlaintext = true
	}

	// cfl/jtk own-scope token under today's precedence, EXCLUDING the
	// shared default (the default is its own key). The shared Store
	// override outranks the legacy file, so a legacy-file value is only
	// reachable when the Store override is empty AND no default shadows it.
	cflOwn, cflLoc := firstNonEmptyLoc(
		store.CFL.APIToken, "shared cfl override",
		shadowed(lcTok, effDefault), "legacy cfl config")
	jtkOwn, jtkLoc := firstNonEmptyLoc(
		store.JTK.APIToken, "shared jtk override",
		shadowed(ljTok, effDefault), "legacy jtk config")

	if cflOwn != "" && cflOwn != effDefault {
		want[KeyCFLAPIToken] = cflOwn
		locs[KeyCFLAPIToken] = cflLoc
	}
	if jtkOwn != "" && jtkOwn != effDefault {
		want[KeyJTKAPIToken] = jtkOwn
		locs[KeyJTKAPIToken] = jtkLoc
	}
	return want, locs, anyPlaintext
}

// shadowed returns "" when a non-empty higher-precedence default exists
// (the legacy-file value is dead data); otherwise it returns the value.
func shadowed(legacyFileTok, effDefault string) string {
	if effDefault != "" {
		return ""
	}
	return legacyFileTok
}

func firstNonEmptyLoc(a, aLoc, b, bLoc string) (string, string) {
	if a != "" {
		return a, aLoc
	}
	if b != "" {
		return b, bLoc
	}
	return "", ""
}

// planWrites is the pure §1.8 resolver: equal current value => idempotent
// skip; differing current value => conflict (unless overwrite). Returns
// only the keys to actually SetBundle.
func planWrites(want, current map[string]string, overwrite bool) (toWrite map[string]string, conflicts []string) {
	toWrite = map[string]string{}
	for k, v := range want {
		cur, present := current[k]
		switch {
		case !present:
			toWrite[k] = v
		case cur == v:
			// already migrated — idempotent no-op (still scrub plaintext)
		case overwrite:
			toWrite[k] = v
		default:
			conflicts = append(conflicts, k)
		}
	}
	sort.Strings(conflicts)
	return toWrite, conflicts
}

// conflictError names the conflicting keys and their non-secret legacy
// locations plus the keyring ref — never a value (§1.12).
func conflictError(conflictKeys []string, locs map[string]string, ref string) error {
	parts := make([]string, 0, len(conflictKeys))
	for _, k := range conflictKeys {
		parts = append(parts, fmt.Sprintf("%s (legacy %s vs keyring %s/%s)", k, locs[k], ref, k))
	}
	return fmt.Errorf("%w: %s; resolve by clearing one side then re-running (or use the overwrite path)",
		ErrMigrationConflict, strings.Join(parts, ", "))
}

// ---- plaintext scrub ------------------------------------------------------

// scrubSharedStore rewrites the shared config.yml with every api_token
// removed, preserving all non-secret fields. credstore.Save already omits
// tokens, so a plain re-save scrubs it; absent file is a no-op.
func scrubSharedStore(store *credstore.Store, path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	store.Default.APIToken = ""
	store.CFL.APIToken = ""
	store.JTK.APIToken = ""
	return store.Save(path)
}

// scrubLegacyCFL rewrites the cfl legacy YAML without api_token, keeping
// the other (non-secret) fields so the file still works as a fallback.
func scrubLegacyCFL(lc *credstore.LegacyCreds) error {
	out := map[string]string{}
	put := func(k, v string) {
		if v != "" {
			out[k] = v
		}
	}
	put("url", lc.URL)
	put("email", lc.Email)
	put("auth_method", lc.AuthMethod)
	put("cloud_id", lc.CloudID)
	put("default_space", lc.DefaultSpace)
	put("output_format", lc.OutputFormat)
	data, err := yaml.Marshal(out)
	if err != nil {
		return err
	}
	return writeScrubbed(lc.Path, data)
}

// scrubLegacyJTK rewrites the jtk legacy JSON without api_token.
func scrubLegacyJTK(lj *credstore.LegacyCreds) error {
	out := map[string]string{}
	put := func(k, v string) {
		if v != "" {
			out[k] = v
		}
	}
	put("url", lj.URL)
	put("email", lj.Email)
	put("auth_method", lj.AuthMethod)
	put("cloud_id", lj.CloudID)
	put("default_project", lj.DefaultProject)
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return writeScrubbed(lj.Path, data)
}

func writeScrubbed(path string, data []byte) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil { //nolint:gosec // G306: 0600 is correct for a config file
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
