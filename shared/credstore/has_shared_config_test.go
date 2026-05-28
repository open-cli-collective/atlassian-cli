package credstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/cli-common/statedirtest"
)

// TestHasSharedConfig_Neither_False — empty hermetic state dir, no canonical
// and no old shared file, expect (false, nil) so the §1.5.2 caller routes to
// "no config → require --ref".
func TestHasSharedConfig_Neither_False(t *testing.T) {
	_ = oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")

	has, err := hasSharedConfigAt(newPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Fatal("empty state must report absent")
	}
}

// TestHasSharedConfig_CanonicalOnly_True — canonical present registers as
// present, no fallback needed.
func TestHasSharedConfig_CanonicalOnly_True(t *testing.T) {
	_ = oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, newPath, "default:\n  url: https://acme.atlassian.net\n  email: u@x.io\n")

	has, err := hasSharedConfigAt(newPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Fatal("canonical present must register as true")
	}
}

// TestHasSharedConfig_OldOnly_True — pre-relocation user with config only at
// the old hand-rolled path must register as present (matches
// LoadSharedRuntime's transparent read-fallback).
func TestHasSharedConfig_OldOnly_True(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n  email: u@x.io\n")

	has, err := hasSharedConfigAt(newPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Fatal("old-only config must register as true (relocation fallback)")
	}
}

// TestHasSharedConfig_BothPresent_True — divergence is irrelevant for the
// presence question; canonical short-circuits the old-path check. Conflict is
// surfaced at config-read time via LoadSharedRuntime, not here.
func TestHasSharedConfig_BothPresent_True(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, oldPath, "default:\n  url: https://acme.atlassian.net\n")
	writeFile(t, newPath, "default:\n  url: https://other.atlassian.net\n")

	has, err := hasSharedConfigAt(newPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Fatal("both-present must register as true (canonical short-circuits)")
	}
}

// TestHasSharedConfig_CanonicalCorrupt_SurfacesError — a corrupt canonical
// file MUST NOT silently route the caller to the no-config branch.
// set-credential expects the error to propagate so the §1.5.2 envelope
// reports the real failure, not a misleading "missing --ref" hint.
func TestHasSharedConfig_CanonicalCorrupt_SurfacesError(t *testing.T) {
	_ = oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, newPath, "[unclosed_array: yes\n")

	has, err := hasSharedConfigAt(newPath)
	if err == nil {
		t.Fatal("corrupt canonical must surface error")
	}
	if !errors.Is(err, ErrCorruptStore) {
		t.Fatalf("error must wrap ErrCorruptStore, got %v", err)
	}
	if has {
		t.Fatal("corrupt canonical must report has=false")
	}
}

// TestHasSharedConfig_OldCorrupt_SurfacesError — canonical absent, old
// corrupt. Same loud-fail contract as the canonical-corrupt case.
func TestHasSharedConfig_OldCorrupt_SurfacesError(t *testing.T) {
	oldPath := oldBase(t)
	newPath := filepath.Join(t.TempDir(), "new", "config.yml")
	writeFile(t, oldPath, "[unclosed_array: yes\n")

	has, err := hasSharedConfigAt(newPath)
	if err == nil {
		t.Fatal("corrupt old must surface error")
	}
	if !errors.Is(err, ErrCorruptStore) {
		t.Fatalf("error must wrap ErrCorruptStore, got %v", err)
	}
	if has {
		t.Fatal("corrupt old must report has=false")
	}
}

// TestHasSharedConfig_StatedirResolution — exercises the exported wrapper
// HasSharedConfig() through the production statedir resolver. On any OS it
// must report false in an empty hermetic root and true after writing to the
// resolved canonical path.
func TestHasSharedConfig_StatedirResolution(t *testing.T) {
	statedirtest.Hermetic(t)

	has, err := HasSharedConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Fatal("empty hermetic root must report absent")
	}

	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, p, "default:\n  url: https://acme.atlassian.net\n")

	has, err = HasSharedConfig()
	if err != nil {
		t.Fatalf("unexpected error after write: %v", err)
	}
	if !has {
		t.Fatal("present canonical must register as true via exported wrapper")
	}
}
