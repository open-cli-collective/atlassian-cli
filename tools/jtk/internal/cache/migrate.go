package cache

import (
	"encoding/json"
	"os"
	"path/filepath"

	cccache "github.com/open-cli-collective/cli-common/cache"
)

// promoteLegacyOnMiss implements the one-time, per-resource, non-destructive
// re-migration from the legacy ~/.jtk/cache root to the new
// os.UserCacheDir()/jtk root (working-with-state.md §6.4, cache variant).
//
// It runs ONLY when cli-common ReadResource already returned ErrCacheMiss. It
// never moves or deletes legacy, never overwrites an existing new envelope,
// and never fails a command: any problem (absent / unreadable / malformed /
// version- or identity-mismatched legacy, or a copy error) yields ok=false,
// which the caller surfaces as a normal miss → refetch. Cache is disposable.
//
// Promotion copies exactly one valid envelope via the shared verbatim writer,
// so the promoted file is byte-faithful and immediately readable (the same
// identity contract the next ReadResource enforces).
func promoteLegacyOnMiss[T any](loc cccache.Locator, name string) (Envelope[T], bool) {
	var zero Envelope[T]

	legRoot := legacyRoot()
	if legRoot == "" {
		return zero, false // hermetic test mode or no home → never probe a real dir
	}

	// Best-effort "do not overwrite an existing new envelope": derived from
	// the SAME already-resolved locator as the write path (not a second
	// independent Root()/InstanceKey() resolution) so the guard and the
	// write can never anchor to different paths. This is not locked against
	// a concurrent WriteResource landing between this stat and the promote
	// write — jtk is a short-lived single process so that race is accepted;
	// worst case is one stale entry until TTL, never corruption.
	newPath := filepath.Join(loc.Root, loc.InstanceKey, name+".json")
	if _, statErr := os.Stat(newPath); statErr == nil {
		return zero, false
	}

	legPath := filepath.Join(legRoot, loc.InstanceKey, name+".json")
	data, err := os.ReadFile(legPath) //nolint:gosec // name is always a hardcoded registry key (never user input); legRoot is the fixed ~/.jtk/cache or a test override
	if err != nil {
		return zero, false // absent / unreadable → plain miss
	}

	var env cccache.Envelope[T]
	if json.Unmarshal(data, &env) != nil {
		return zero, false // malformed legacy → miss, not an error
	}
	if env.Version != cccache.Version || env.Resource != name || env.Instance != loc.InstanceKey {
		return zero, false // stale schema / misplaced file → miss
	}

	if cccache.WriteEnvelope(loc, env) != nil {
		return zero, false // copy failure is non-fatal → behave as a miss
	}
	return env, true
}
