package keyring

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

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
		return deferLegacyLoadErr(errC)
	}
	legacyJTK, errJ := credstore.LoadLegacyJTK(jtkPath)
	if errJ != nil {
		return deferLegacyLoadErr(errJ)
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
		// Only force-overwrite on the explicit overwrite path. On the
		// normal path planWrites already excluded every key that exists
		// with a different value (those are fail-loud conflicts); writing
		// without WithOverwrite keeps that contract even against a key
		// that appeared between the preflight read and here (TOCTOU) —
		// the backend errors instead of silently clobbering it.
		var opts []cccredstore.SetOpt
		if overwrite {
			opts = append(opts, cccredstore.WithOverwrite())
		}
		if _, err := s.cs.SetBundle(s.profile, toWrite, opts...); err != nil {
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
	// Scrub the legacy files via the SAME generic map round-trip
	// `config clear` uses (scrubLegacyFile): it deletes only api_token
	// and preserves every other key verbatim — including fields a user
	// hand-added or a future schema introduces. A hardcoded
	// field-allowlist rewrite would silently drop those during this
	// one-time, irreversible migration.
	if legacyCFL != nil && legacyCFL.APIToken != "" {
		if err := scrubLegacyFile(cflPath); err != nil {
			return fmt.Errorf("migrated to keyring %s but could not scrub %s: %w", s.ref, cflPath, err)
		}
	}
	if legacyJTK != nil && legacyJTK.APIToken != "" {
		if err := scrubLegacyFile(jtkPath); err != nil {
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

// deferLegacyLoadErr classifies ANY legacy-file load failure as
// ErrCorruptStore so ResolveToken's graceful path (warn once, skip
// migration, keep resolving from the keyring) handles it uniformly.
// LoadLegacyCFL/JTK already wrap parse errors; this also catches the
// merely-unreadable cases (permissions, bad encoding, missing dir) so a
// single broken legacy file can never de-authenticate every command —
// only `init` (which explicitly reconciles) should hard-fail on it.
func deferLegacyLoadErr(err error) error {
	if errors.Is(err, credstore.ErrCorruptStore) {
		return err
	}
	return fmt.Errorf("%w: %w", credstore.ErrCorruptStore, err)
}

// gatherEffective computes the genuine per-scope key writes, a non-secret
// location string per key (for conflict messages), and whether ANY
// plaintext token existed anywhere (drives scrub even when every value is
// shadowed/cleanup-only).
func gatherEffective(store *credstore.Store, lc, lj *credstore.LegacyCreds) (want, locs map[string]string, anyPlaintext bool) {
	want = map[string]string{}
	locs = map[string]string{}

	// Location strings name the concrete plaintext file so a conflict
	// error tells the user exactly which file still holds a secret to
	// remove/scrub — no separate diagnostic step.
	sharedPath := credstore.DefaultPath()

	effDefault := store.Default.APIToken
	if effDefault != "" {
		want[KeyAPIToken] = effDefault
		locs[KeyAPIToken] = "shared config default (" + sharedPath + ")"
	}

	lcTok, ljTok := "", ""
	lcPath, ljPath := "", ""
	if lc != nil {
		lcTok = lc.APIToken
		lcPath = lc.Path
	}
	if lj != nil {
		ljTok = lj.APIToken
		ljPath = lj.Path
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
		store.CFL.APIToken, "shared cfl override ("+sharedPath+")",
		shadowed(lcTok, effDefault), "legacy cfl config ("+lcPath+")")
	jtkOwn, jtkLoc := firstNonEmptyLoc(
		store.JTK.APIToken, "shared jtk override ("+sharedPath+")",
		shadowed(ljTok, effDefault), "legacy jtk config ("+ljPath+")")

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
	return fmt.Errorf("%w: %s; resolve by clearing one side (`config clear`, or remove/scrub the legacy plaintext file) then re-running",
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

// (Legacy-file scrubbing is the generic, field-preserving scrubLegacyFile
// in clear.go — shared by both the migration and `config clear` paths so
// the "delete only api_token, keep everything else" guarantee lives in
// exactly one implementation.)
