package credstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrRelocationConflict is the stable identity for a §3.2 old↔new
// shared-config divergence: the prior hand-rolled location and the
// statedir-resolved location both exist but hold different durable
// config. init fails loud naming BOTH absolute paths and mutates
// nothing — it never precedence-picks a winner (no values shown).
var ErrRelocationConflict = errors.New("credstore: prior and current shared config diverge")

// oldSharedPath reproduces the PRE-statedir hand-rolled shared location
// ($XDG_CONFIG_HOME|~/.config /atlassian-cli/config.yml) so a user who
// configured under the old layout is not silently abandoned by the
// macOS/Windows resolver move. It applies the SAME relative-
// $XDG_CONFIG_HOME rejection as the new resolver: it must NEVER
// reintroduce the old cwd-relative ./.atlassian-cli fallback. A
// relative/unresolvable old base ⇒ ("", nil): the old-shared probe is
// skipped entirely (no enumeration, no copy, no cleanup target), never
// silently cwd-relative.
func oldSharedPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		if !filepath.IsAbs(xdg) {
			return "", nil
		}
		return filepath.Join(xdg, "atlassian-cli", "config.yml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" || !filepath.IsAbs(home) {
		return "", nil
	}
	return filepath.Join(home, ".config", "atlassian-cli", "config.yml"), nil
}

// SharedRelocation is the PURE result of §3.2 old→new shared-config
// detection. It mutates nothing; it is inert until the caller — having
// passed every conflict gate (this relocation check AND the per-tool
// connection-divergence check) — invokes ApplySharedRelocation. OldProj
// is the migration-only projection (same shape as
// LoadSharedLegacyProjection) so the keyring machinery can enumerate /
// scrub a stale plaintext token at the old path without a prior copy.
type SharedRelocation struct {
	OldPath    string                  // "" ⇒ no old-shared (skipped / path-identity / absent)
	NewPath    string                  // the statedir-resolved canonical path
	CopyNeeded bool                    // old present & new absent ⇒ copy AFTER gates pass
	OldProj    *SharedLegacyProjection // old file's pre-MON-5328 projection (nil ⇒ no old)
}

// DetectSharedRelocation is the PURE pre-token detect/enumerate phase.
// It performs NO mutation and NO copy. Contract:
//
//   - old skipped (relative XDG / unresolvable home) or path-identical
//     to new (Linux: $XDG/~/.config unchanged) ⇒ no-op (dedup; no
//     double-read, no self-copy, no double-enumeration).
//   - old absent ⇒ no-op.
//   - malformed old OR malformed new ⇒ ErrCorruptStore (fail loud,
//     mutate nothing; a malformed new is never overwritten).
//   - old present, new absent ⇒ CopyNeeded (deferred to the gated apply).
//   - both present ⇒ compared on the dedicated relocation projection
//     (canonical tool defaults + legacy per-tool conn/token + token
//     presence). Identical ⇒ no-op; divergent ⇒ ErrRelocationConflict
//     naming BOTH absolute paths, mutating nothing.
func DetectSharedRelocation(newPath string) (*SharedRelocation, error) {
	r := &SharedRelocation{NewPath: newPath}
	oldPath, _ := oldSharedPath()
	if oldPath == "" || oldPath == newPath {
		return r, nil
	}
	oldProj, err := LoadSharedLegacyProjection(oldPath)
	if err != nil {
		return nil, err
	}
	if oldProj == nil {
		return r, nil
	}
	oldStore, err := Load(oldPath)
	if err != nil {
		return nil, err
	}
	r.OldPath = oldPath
	r.OldProj = oldProj

	newProj, err := LoadSharedLegacyProjection(newPath)
	if err != nil {
		return nil, err
	}
	if newProj == nil {
		r.CopyNeeded = true
		return r, nil
	}
	newStore, err := Load(newPath)
	if err != nil {
		return nil, err
	}
	if relocationEqual(oldStore, oldProj, newStore, newProj) {
		return r, nil
	}
	return nil, fmt.Errorf(
		"%w: %s and %s hold different connection or non-secret defaults; "+
			"reconcile or remove one, then re-run init (no values shown; "+
			"secrets live only in the OS keyring)",
		ErrRelocationConflict, oldPath, newPath)
}

// relocationEqual is the dedicated relocation-equality projection. It
// covers BOTH (a) the legacy per-tool connection/token fields (which the
// canonical Store drops post-MON-5328, so a pre-migration token/conn
// divergence is not masked) AND (b) the canonical non-secret tool
// defaults (default_space/output_format/default_project, which the
// legacy projection does not carry, so a durable-defaults divergence is
// not masked). Neither projection alone suffices. URLs/auth_method are
// canonicalized so a cosmetic difference does not false-conflict.
func relocationEqual(oS *Store, oP *SharedLegacyProjection, nS *Store, nP *SharedLegacyProjection) bool {
	// Token comparison is presence-aware but migration-skew tolerant: a
	// token on ONE side only is the EXPECTED pre-/post-migration state
	// (the keyring machinery relocates a stale plaintext token), not a
	// durable-config divergence. Only TWO DIFFERENT non-empty tokens is
	// a true conflict — and the keyring planMigration also fails loud on
	// that (defense in depth, consistent fail-loud semantics).
	tokenCompatible := func(a, b string) bool { return a == b || a == "" || b == "" }
	connEq := func(a, b SharedLegacyConn) bool {
		return NormalizeBaseURL(a.URL) == NormalizeBaseURL(b.URL) &&
			a.Email == b.Email &&
			tokenCompatible(a.APIToken, b.APIToken) &&
			canonAuthMethod(a.AuthMethod) == canonAuthMethod(b.AuthMethod) &&
			a.CloudID == b.CloudID
	}
	return connEq(oP.Default, nP.Default) &&
		connEq(oP.CFL, nP.CFL) &&
		connEq(oP.JTK, nP.JTK) &&
		oS.CFL.DefaultSpace == nS.CFL.DefaultSpace &&
		oS.CFL.OutputFormat == nS.CFL.OutputFormat &&
		oS.JTK.DefaultProject == nS.JTK.DefaultProject
}

// ApplySharedRelocation is the GATED apply/copy phase. copy-leave-old:
// the old file is intentionally NOT removed (a stale plaintext token
// there is handled by the keyring migration/scrub machinery; a stale
// LATER old write is caught by always-reconcile at the next load rather
// than silently winning). It is a no-op unless detection found an
// old-only file. The new file is written atomically (temp+rename,
// 0700 dir / 0600 file) so a crash never leaves a half-written shared
// config. The caller MUST invoke this only AFTER every conflict gate
// (relocation + per-tool connection divergence) has passed.
func ApplySharedRelocation(r *SharedRelocation) error {
	if r == nil || !r.CopyNeeded || r.OldPath == "" {
		return nil
	}
	data, err := os.ReadFile(r.OldPath) //nolint:gosec // CLI relocating its own config
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("relocating shared config: reading %s: %w", r.OldPath, err)
	}
	dir := filepath.Dir(r.NewPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("relocating shared config: creating %s: %w", dir, err)
	}
	tmp := r.NewPath + ".tmp"
	//nolint:gosec // NewPath is the resolver-derived shared config path, not user input
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("relocating shared config: writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, r.NewPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("relocating shared config: renaming -> %s: %w", r.NewPath, err)
	}
	return nil
}

// OldSharedConnCandidates yields the origin-labeled connection
// candidates contributed by the prior hand-rolled shared file, so the
// per-tool connection-divergence detector COMPOSES with it (a copy is
// gated on this passing — "no copy while a per-tool divergence is
// pending"). It reuses the canonical ConnCandidates assembly (default +
// pre-MON-5328 per-tool effective overrides) over the old file, relabeled
// "prior shared config" so a conflict message names the old path
// distinctly. Empty when there is no old-shared file.
func OldSharedConnCandidates(r *SharedRelocation) []NamedConn {
	if r == nil || r.OldProj == nil || r.OldPath == "" {
		return nil
	}
	oldDef := Section{
		URL:        r.OldProj.Default.URL,
		Email:      r.OldProj.Default.Email,
		AuthMethod: r.OldProj.Default.AuthMethod,
		CloudID:    r.OldProj.Default.CloudID,
	}
	cands := ConnCandidates(r.OldPath, oldDef, r.OldProj, nil, nil)
	for i := range cands {
		cands[i].Label = "prior " + cands[i].Label
	}
	return cands
}

// OldSharedProjection returns the migration-only projection of the prior
// hand-rolled shared location plus its absolute path, for ADDITIVE
// inclusion in the keyring token-migration source set (so a stale
// plaintext api_token at the old location is enumerated/scrubbed before
// token resolution, never left behind). ("", nil, nil) when there is no
// addressable, distinct, present old-shared file (skipped / path-
// identity with the current resolver / absent). A parse failure returns
// ErrCorruptStore so the secret machinery fails loud rather than
// scrubbing blindly.
func OldSharedProjection(newPath string) (string, *SharedLegacyProjection, error) {
	oldPath, _ := oldSharedPath()
	if oldPath == "" || oldPath == newPath {
		return "", nil, nil
	}
	proj, err := LoadSharedLegacyProjection(oldPath)
	if err != nil {
		return "", nil, err
	}
	if proj == nil {
		return "", nil, nil
	}
	return oldPath, proj, nil
}

// OldSharedConfigPath returns the prior hand-rolled shared-config path
// when it is addressable AND distinct from the current resolver path
// (path-identity dedup so `config clear --all` does not double-list /
// double-remove the same file on Linux). Existence is the caller's
// concern. "" ⇒ no distinct old-shared path to clear.
func OldSharedConfigPath(newPath string) string {
	oldPath, _ := oldSharedPath()
	if oldPath == "" || oldPath == newPath {
		return ""
	}
	return oldPath
}
