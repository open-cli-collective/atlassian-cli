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
	got, err := oldSharedPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
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
		b, _ := os.ReadFile(newPath)
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

func TestApplySharedRelocation_NoOpWhenNotNeeded(t *testing.T) {
	if err := ApplySharedRelocation(nil); err != nil {
		t.Fatalf("nil ⇒ no-op, got %v", err)
	}
	if err := ApplySharedRelocation(&SharedRelocation{}); err != nil {
		t.Fatalf("not-needed ⇒ no-op, got %v", err)
	}
}
