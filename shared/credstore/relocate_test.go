package credstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/cli-common/statedirtest"
)

// writeFile writes content creating parent dirs, failing the test on error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// oldBase points oldSharedPath at an absolute temp base distinct from
// whatever newPath the test passes, so the relocation logic is exercised
// on every OS (not only where the resolver actually moved).
func oldBase(t *testing.T) string {
	t.Helper()
	root := statedirtest.Hermetic(t)
	base := filepath.Join(root, "oldbase")
	t.Setenv("XDG_CONFIG_HOME", base)
	return filepath.Join(base, "atlassian-cli", "config.yml")
}

func TestOldSharedPath_RelativeXDGSkipped(t *testing.T) {
	statedirtest.Hermetic(t)
	t.Setenv("XDG_CONFIG_HOME", "relative/not/abs")
	if got := oldSharedPath(); got != "" {
		t.Fatalf("relative $XDG_CONFIG_HOME must skip the old-shared probe, got %q", got)
	}
}

func TestDetectSharedRelocation_PathIdentityShortCircuit(t *testing.T) {
	oldPath := oldBase(t)
	writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n")
	// new == old ⇒ no-op, no double-read, no copy.
	rel, err := DetectSharedRelocation(oldPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel.OldPath != "" || rel.CopyNeeded {
		t.Fatalf("path-identity must be a no-op, got %+v", rel)
	}
}

func TestDetectSharedRelocation_OldOnlyCopyNeeded(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n  email: u@x.io\n")

	rel, err := DetectSharedRelocation(newPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rel.CopyNeeded || rel.OldPath != oldPath || rel.OldProj == nil {
		t.Fatalf("old-only must set CopyNeeded with OldProj, got %+v", rel)
	}
	if _, statErr := os.Stat(newPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal("detection must not have copied anything (pure phase)")
	}

	if err := ApplySharedRelocation(rel); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, statErr := os.Stat(newPath); statErr != nil {
		t.Fatalf("apply must materialize new path: %v", statErr)
	}
	if _, statErr := os.Stat(oldPath); statErr != nil {
		t.Fatalf("copy-leave-old: old must remain, got %v", statErr)
	}
}

func TestDetectSharedRelocation_BothEqualNoOp(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	body := "default:\n  url: https://acme.atlassian.net\n  email: u@x.io\ncfl:\n  default_space: ENG\n"
	writeFile(t, oldPath, body)
	writeFile(t, newPath, body)

	rel, err := DetectSharedRelocation(newPath)
	if err != nil {
		t.Fatalf("identical old/new must be a no-op, got err %v", err)
	}
	if rel.CopyNeeded {
		t.Fatal("identical old/new must not copy")
	}
}

func TestDetectSharedRelocation_BothDivergentConflict(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, oldPath, "default:\n  url: https://OLD.atlassian.net\n")
	writeFile(t, newPath, "default:\n  url: https://NEW.atlassian.net\n")

	_, err := DetectSharedRelocation(newPath)
	if !errors.Is(err, ErrRelocationConflict) {
		t.Fatalf("divergent old/new must fail loud with ErrRelocationConflict, got %v", err)
	}
}

func TestDetectSharedRelocation_TokenSkewTolerated(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	// Same durable config; token only on old (the expected pre-migration
	// state). Must NOT false-conflict.
	writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n  api_token: SECRET\n")
	writeFile(t, newPath, "default:\n  url: https://acme.atlassian.net\n")

	rel, err := DetectSharedRelocation(newPath)
	if err != nil {
		t.Fatalf("token-only-on-old must be tolerated, got %v", err)
	}
	if rel.CopyNeeded {
		t.Fatal("both present ⇒ no copy")
	}
}

func TestDetectSharedRelocation_TwoDifferentTokensConflict(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n  api_token: TOK_A\n")
	writeFile(t, newPath, "default:\n  url: https://acme.atlassian.net\n  api_token: TOK_B\n")

	_, err := DetectSharedRelocation(newPath)
	if !errors.Is(err, ErrRelocationConflict) {
		t.Fatalf("two different non-empty tokens must conflict, got %v", err)
	}
}

func TestDetectSharedRelocation_MalformedFailsLoud(t *testing.T) {
	t.Run("old", func(t *testing.T) {
		oldPath := oldBase(t)
		newPath := filepath.Join(t.TempDir(), "new", "config.yml")
		writeFile(t, oldPath, "{not: valid: yaml: ::::")
		_, err := DetectSharedRelocation(newPath)
		if !errors.Is(err, ErrCorruptStore) {
			t.Fatalf("malformed old must fail loud, got %v", err)
		}
	})
	t.Run("new", func(t *testing.T) {
		oldPath := oldBase(t)
		newPath := filepath.Join(t.TempDir(), "new", "config.yml")
		writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n")
		writeFile(t, newPath, "{not: valid: yaml: ::::")
		_, err := DetectSharedRelocation(newPath)
		if !errors.Is(err, ErrCorruptStore) {
			t.Fatalf("malformed new must fail loud (never overwritten), got %v", err)
		}
		// new untouched
		b, _ := os.ReadFile(newPath) //nolint:gosec // test reads its own temp file
		if string(b) != "{not: valid: yaml: ::::" {
			t.Fatal("malformed new must not be mutated")
		}
	})
}

func TestOldSharedConnCandidates_RelabelAndDefaultsCovered(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n  email: u@x.io\n")

	rel, err := DetectSharedRelocation(newPath)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	cands := OldSharedConnCandidates(rel)
	if len(cands) == 0 {
		t.Fatal("old-shared must contribute a connection candidate")
	}
	for _, c := range cands {
		if c.Label != "prior shared config" {
			t.Fatalf("candidate must be relabeled, got %q", c.Label)
		}
		if c.Path != oldPath {
			t.Fatalf("candidate path must name the old file, got %q", c.Path)
		}
	}
}

// On Linux oldSharedPath() ≡ DefaultPath() (both honor
// $XDG_CONFIG_HOME), so old-shared MUST NOT be enumerated as a second
// token/clear source for the very same file (Codex r2: "old==new not
// double-enumerated"). These assert the dedup at the seams the keyring
// migrate/clear sets consume.
func TestOldSharedProjection_PathIdentityNotEnumerated(t *testing.T) {
	oldPath := oldBase(t)
	writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n  api_token: SECRET\n")
	// new == old: the keyring source set already covers this file via
	// the canonical DefaultPath; old-shared must contribute nothing.
	gotPath, proj, err := OldSharedProjection(oldPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "" || proj != nil {
		t.Fatalf("path-identity must not double-enumerate, got path=%q proj=%v", gotPath, proj)
	}
}

func TestOldSharedConfigPath_PathIdentityDeduped(t *testing.T) {
	oldPath := oldBase(t)
	if got := OldSharedConfigPath(oldPath); got != "" {
		t.Fatalf("path-identity must dedup the clear path set, got %q", got)
	}
}

func TestOldSharedProjection_EnumeratesDistinctOldToken(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n  api_token: STALE\n")

	gotPath, proj, err := OldSharedProjection(newPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != oldPath || proj == nil || proj.Default.APIToken != "STALE" {
		t.Fatalf("distinct old-shared with a token must be enumerated, got path=%q proj=%v", gotPath, proj)
	}
}

// Minor 1 (Codex r2 architectural requirement): the relocation
// projection MUST also catch a durable tool-default divergence — the
// legacy projection alone (conn/token only) would mask it.
func TestDetectSharedRelocation_ToolDefaultDivergenceConflict(t *testing.T) {
	for _, tc := range []struct{ name, oldBody, newBody string }{
		{"cfl.default_space",
			"cfl:\n  default_space: ENG\n",
			"cfl:\n  default_space: OPS\n"},
		{"cfl.output_format",
			"cfl:\n  output_format: json\n",
			"cfl:\n  output_format: table\n"},
		{"jtk.default_project",
			"jtk:\n  default_project: MON\n",
			"jtk:\n  default_project: INT\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldPath := oldBase(t)
			newPath := filepath.Join(t.TempDir(), "new", "config.yml")
			writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n"+tc.oldBody)
			writeFile(t, newPath, "default:\n  url: https://acme.atlassian.net\n"+tc.newBody)
			if _, err := DetectSharedRelocation(newPath); !errors.Is(err, ErrRelocationConflict) {
				t.Fatalf("durable %s divergence must fail loud, got %v", tc.name, err)
			}
		})
	}
}

func TestLoadSharedRuntime_OldOnlyReadFallbackNoCopy(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n  email: u@x.io\n")
	// new absent: runtime reads old transparently, mutating nothing.
	st, err := loadSharedRuntime(newPath)
	if err != nil {
		t.Fatalf("old-only runtime read: %v", err)
	}
	if st.Default.URL != "https://acme.atlassian.net" || st.Default.Email != "u@x.io" {
		t.Fatalf("old-only must be read as the effective store, got %+v", st.Default)
	}
	if _, statErr := os.Stat(newPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal("runtime read path must NOT copy/create the new file")
	}
}

func TestLoadSharedRuntime_DivergentReturnsCanonicalPlusError(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, oldPath, "default:\n  url: https://OLD.atlassian.net\n")
	writeFile(t, newPath, "default:\n  url: https://NEW.atlassian.net\n")

	st, err := loadSharedRuntime(newPath)
	if !errors.Is(err, ErrRelocationConflict) {
		t.Fatalf("divergence must be surfaced, got %v", err)
	}
	if st == nil || st.Default.URL != "https://NEW.atlassian.net" {
		t.Fatalf("divergence must still yield the canonical store so commands work, got %+v", st)
	}
}

func TestLoadSharedRuntime_BothEqualUsesCanonical(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	body := "default:\n  url: https://acme.atlassian.net\n  email: u@x.io\n"
	writeFile(t, oldPath, body)
	writeFile(t, newPath, body)
	st, err := loadSharedRuntime(newPath)
	if err != nil {
		t.Fatalf("equal old/new must be clean, got %v", err)
	}
	if st.Default.Email != "u@x.io" {
		t.Fatalf("unexpected store %+v", st.Default)
	}
}

func TestLoadSharedRuntime_PathIdentityUsesCanonical(t *testing.T) {
	// Linux reality: old≡new ⇒ no fallback, no double-handling.
	oldPath := oldBase(t)
	writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n")
	st, err := loadSharedRuntime(oldPath)
	if err != nil {
		t.Fatalf("path-identity must be clean, got %v", err)
	}
	if st.Default.URL != "https://acme.atlassian.net" {
		t.Fatalf("unexpected store %+v", st.Default)
	}
}

func TestApplySharedRelocation_NoOpWhenNotNeeded(t *testing.T) {
	if err := ApplySharedRelocation(nil); err != nil {
		t.Fatalf("nil ⇒ no-op, got %v", err)
	}
	if err := ApplySharedRelocation(&SharedRelocation{}); err != nil {
		t.Fatalf("not-needed ⇒ no-op, got %v", err)
	}
}
