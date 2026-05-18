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

// One-time §1.8 migration. The access secret is the Atlassian api_token,
// and there is exactly ONE keyring key for it (§1.11.10): jtk and cfl
// share `api_token`. This migration unifies every legacy source onto that
// one key:
//
//   - legacy plaintext: shared config.yml (Default/CFL/JTK .APIToken) and
//     the legacy per-tool files (cfl yml, jtk json);
//   - deprecated keyring keys cfl_api_token / jtk_api_token left by an
//     earlier build (B3 upgrade path: a user may hold ONLY these, with
//     plaintext already scrubbed).
//
// Behavior (amended §1.8): collect every non-empty migration-source value;
// if more than one DISTINCT value exists across all sources it is a hard
// conflict (fail loud, name every source, never print a secret, never
// precedence-pick). With exactly one distinct value it is compared to any
// existing `api_token`: absent → write; equal → no-op; different →
// conflict unless overwrite. All plaintext is scrubbed and both
// deprecated keyring keys deleted afterwards. The whole thing is
// strictly two-phase: collect + detect (pure, no mutation) THEN apply, so
// a failed migration leaves every source untouched.

// deprecatedKeys are the removed per-tool override keys. They are NOT in
// allowedKeys (the §1.11.11 conforming bundle is exactly {api_token});
// they exist only so this one-time migration and `config clear --all`
// can read/delete residual B3 state. The migration store is opened with
// migrationAllowedKeys so credstore permits Delete of these.
var deprecatedKeys = []string{"cfl_api_token", "jtk_api_token"} //nolint:gosec // G101: bundle key names, not credentials

// migrationAllowedKeys = allowedKeys ∪ deprecatedKeys. Used only by the
// migration open and OpenForClearAll; runtime / OpenNoMigrate stay strict.
var migrationAllowedKeys = append(append([]string{}, allowedKeys...), deprecatedKeys...)

// migrationApplyHook is a TEST-ONLY white-box seam (always nil in
// production; the nil guard means zero production behavior). It cannot
// live in a _test.go file because migrateLegacyOverwrite — production
// code — references it. It is invoked once (with the in-flight store)
// after phase-1 detect and before the phase-2 write: the only point at
// which a test can deterministically simulate a concurrent api_token
// writer racing the migration, writing through the SAME open handle (a
// second open would lock-conflict on the file backend).
//
// It has no mutex by design: it MUST never be set from a parallel test.
// The tests that set it use the hermetic harness (t.Setenv), which is
// already incompatible with t.Parallel(), so this holds structurally.
var migrationApplyHook func(s *Store)

// ErrMigrationConflict is the stable identity for a §1.8 conflict.
var ErrMigrationConflict = errors.New("keyring: API token migration sources disagree")

func migrateLegacyOverwrite(s *Store, overwrite bool) error {
	// ---- Phase 1: collect (no mutation) ----------------------------------
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

	// Existing target value (NOT a migration source — it is the target).
	curAPI, _, err := s.get(KeyAPIToken)
	if err != nil {
		return err
	}

	// Migration sources: value -> sorted, de-duplicated locations.
	srcLoc := map[string]map[string]struct{}{}
	add := func(val, loc string) {
		if val == "" {
			return
		}
		if srcLoc[val] == nil {
			srcLoc[val] = map[string]struct{}{}
		}
		srcLoc[val][loc] = struct{}{}
	}
	add(store.Default.APIToken, "shared config default ("+sharedPath+")")
	add(store.CFL.APIToken, "shared cfl override ("+sharedPath+")")
	add(store.JTK.APIToken, "shared jtk override ("+sharedPath+")")
	if legacyCFL != nil {
		add(legacyCFL.APIToken, "legacy cfl config ("+legacyCFL.Path+")")
	}
	if legacyJTK != nil {
		add(legacyJTK.APIToken, "legacy jtk config ("+legacyJTK.Path+")")
	}
	depPresent := map[string]string{} // key -> value, for deletion in phase 2
	for _, dk := range deprecatedKeys {
		v, ok, gerr := s.get(dk)
		if gerr != nil {
			return gerr
		}
		if ok {
			depPresent[dk] = v
			add(v, "keyring deprecated key "+dk+" ("+s.ref+")")
		}
	}

	anyPlaintext := store.Default.APIToken != "" || store.CFL.APIToken != "" ||
		store.JTK.APIToken != "" ||
		(legacyCFL != nil && legacyCFL.APIToken != "") ||
		(legacyJTK != nil && legacyJTK.APIToken != "")

	// ---- Phase 1: detect (pure) ------------------------------------------
	plan, conflictLocs := planMigration(curAPI, srcLoc, overwrite)
	if len(conflictLocs) > 0 {
		// When an api_token already exists it is one of the disagreeing
		// parties (source-vs-existing, or "and the keyring value too" in a
		// multi-source split) — name it so the message isn't "divergence
		// across" a single source.
		if curAPI != "" {
			conflictLocs = append(conflictLocs, "keyring "+KeyAPIToken+" ("+s.ref+")")
			sort.Strings(conflictLocs)
		}
		return conflictError(conflictLocs, s.ref)
	}

	// ---- Phase 2: apply (only reached with zero conflicts) ---------------
	// Test seam: lets a white-box test create api_token AFTER phase-1's
	// read but BEFORE the write, deterministically exercising the
	// concurrent-writer ErrExists reconciliation below. nil in production.
	if migrationApplyHook != nil {
		migrationApplyHook(s)
	}
	changed := false
	if plan.write {
		switch err := s.setToken(KeyAPIToken, plan.value, overwrite); {
		case err == nil:
			changed = true
		case !overwrite && errors.Is(err, cccredstore.ErrExists):
			// A concurrent writer set api_token between the phase-1 read
			// and now. Re-resolve and apply the same no-precedence rule:
			// an identical value is benign (fall through to cleanup); any
			// difference is a conflict naming the now-present keyring
			// value — never silently clobber it.
			cur, _, gerr := s.get(KeyAPIToken)
			if gerr != nil {
				return gerr
			}
			if cur != plan.value {
				return conflictError(
					append(sortedLocs(srcLoc), "keyring "+KeyAPIToken+" ("+s.ref+")"),
					s.ref)
			}
			// Identical value: the racer committed exactly what we would
			// have. The §1.8 consolidation IS effective this run (the
			// notice should fire even with nothing left to scrub/delete).
			changed = true
		default:
			return err
		}
	}
	removedDeprecated := false
	for _, dk := range deprecatedKeys {
		if _, ok := depPresent[dk]; ok {
			if err := s.DeleteToken(dk); err != nil {
				return fmt.Errorf("migrated to keyring %s but could not remove deprecated key %s: %w", s.ref, dk, err)
			}
			changed = true
			removedDeprecated = true
		}
	}
	if anyPlaintext {
		if err := scrubSharedStore(store, sharedPath); err != nil {
			return fmt.Errorf("migrated to keyring %s but could not scrub %s: %w", s.ref, sharedPath, err)
		}
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
		changed = true
	}

	if changed {
		var cleaned string
		switch {
		case anyPlaintext && removedDeprecated:
			cleaned = "; the legacy plaintext copy and deprecated per-tool keyring keys were removed"
		case anyPlaintext:
			cleaned = "; the legacy plaintext copy was removed"
		case removedDeprecated:
			cleaned = "; deprecated per-tool keyring keys were removed"
		default:
			// Defensive only: changed=true with plan.write set implies a
			// non-empty srcLoc, and every srcLoc source is either plaintext
			// (anyPlaintext) or a deprecated key (removedDeprecated) — so
			// this is unreachable today. Kept so a future source kind can't
			// silently print a wrong cleanup clause.
			cleaned = ""
		}
		recordMigration(fmt.Sprintf(
			"atlassian-cli: consolidated the API token into the OS keyring %s (%s)%s",
			s.ref, KeyAPIToken, cleaned))
	}
	return nil
}

// migrationPlan is the pure decision: whether to write api_token and to
// what value. Deletes/scrubs are unconditional in phase 2 (idempotent);
// only the write is value-dependent.
type migrationPlan struct {
	write bool
	value string
}

// planMigration is the pure §1.8 resolver over the collected sources.
//   - >1 distinct source value           → conflict (every location).
//   - exactly 1 distinct value v:
//     curAPI == ""        → write v
//     curAPI == v         → no write (cleanup/scrub only)
//     curAPI != v         → conflict, unless overwrite (then write v)
//   - 0 sources                           → no write (nothing to migrate)
//
// overwrite resolves source-vs-api_token ONLY; it never resolves a
// >1-distinct-source conflict.
func planMigration(curAPI string, srcLoc map[string]map[string]struct{}, overwrite bool) (migrationPlan, []string) {
	if len(srcLoc) > 1 {
		return migrationPlan{}, sortedLocs(srcLoc)
	}
	if len(srcLoc) == 0 {
		return migrationPlan{}, nil
	}
	var v string
	for val := range srcLoc {
		v = val
	}
	switch {
	case curAPI == "":
		return migrationPlan{write: true, value: v}, nil
	case curAPI == v:
		return migrationPlan{}, nil
	case overwrite:
		return migrationPlan{write: true, value: v}, nil
	default:
		return migrationPlan{}, sortedLocs(srcLoc)
	}
}

func sortedLocs(srcLoc map[string]map[string]struct{}) []string {
	var locs []string
	for _, set := range srcLoc {
		for l := range set {
			locs = append(locs, l)
		}
	}
	sort.Strings(locs)
	return locs
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

// conflictError names every conflicting source location plus the keyring
// ref — never a value (§1.12) — and points at the supported recovery
// path now that per-tool `--key` is gone.
func conflictError(locs []string, ref string) error {
	return fmt.Errorf("%w: divergent API token values across %s; the migration will not pick a winner. "+
		"Resolve by removing/scrubbing all but one source (or `config clear --all` then `set-credential` to start clean, "+
		"or delete the conflicting entry from the OS keychain), then re-run. Keyring ref: %s",
		ErrMigrationConflict, strings.Join(locs, ", "), ref)
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
