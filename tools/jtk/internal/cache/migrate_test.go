package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	cccache "github.com/open-cli-collective/cli-common/cache"
)

const migInstance = "test.atlassian.net"

// writeLegacy plants a raw envelope JSON at <legacyRoot>/<instance>/<name>.json.
func writeLegacy(t *testing.T, legacyDir, name string, env cccache.Envelope[[]int]) {
	t.Helper()
	dir := filepath.Join(legacyDir, migInstance)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeLegacyRaw(t *testing.T, legacyDir, name, body string) {
	t.Helper()
	dir := filepath.Join(legacyDir, migInstance)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func validLegacyEnv(name string) cccache.Envelope[[]int] {
	return cccache.Envelope[[]int]{
		Resource:  name,
		Instance:  migInstance,
		FetchedAt: time.Now().UTC(),
		TTL:       "24h",
		Version:   cccache.Version,
		Data:      []int{1, 2, 3},
	}
}

// migEnv isolates root + legacy root + instance key for one migration test.
func migEnv(t *testing.T) (newRoot, legacyDir string) {
	t.Helper()
	newRoot = t.TempDir()
	legacyDir = t.TempDir()
	t.Cleanup(SetRootForTest(newRoot))
	t.Cleanup(SetInstanceKeyForTest(migInstance))
	t.Cleanup(SetLegacyRootForTest(legacyDir))
	return newRoot, legacyDir
}

func TestPromote_LegacyOnly_Promoted(t *testing.T) {
	newRoot, legacyDir := migEnv(t)
	writeLegacy(t, legacyDir, "boards", validLegacyEnv("boards"))

	env, err := ReadResource[[]int]("boards")
	if err != nil {
		t.Fatalf("expected promotion, got err %v", err)
	}
	if len(env.Data) != 3 {
		t.Fatalf("promoted data wrong: %+v", env.Data)
	}
	// Identity contract: a promoted envelope must carry correct metadata,
	// not just correct Data (a metadata-corrupt promote would self-miss).
	if env.Resource != "boards" || env.Instance != migInstance || env.Version != cccache.Version {
		t.Fatalf("promoted metadata wrong: resource=%q instance=%q version=%d",
			env.Resource, env.Instance, env.Version)
	}
	// The promoted envelope is now in the new root.
	if _, statErr := os.Stat(filepath.Join(newRoot, migInstance, "boards.json")); statErr != nil {
		t.Fatalf("promotion did not copy into the new root: %v", statErr)
	}
}

func TestPromote_ResourceMismatchLegacy_IsMiss(t *testing.T) {
	_, legacyDir := migEnv(t)
	bad := validLegacyEnv("boards")
	bad.Resource = "other" // legacy file at boards.json but envelope names "other"
	writeLegacy(t, legacyDir, "boards", bad)

	if _, err := ReadResource[[]int]("boards"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("resource-mismatched legacy must be a miss, got %v", err)
	}
}

func TestPromote_CopyErrorIsNonFatalMiss(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("perm-based copy-failure injection is unreliable on windows / as root")
	}
	legacyDir := t.TempDir()
	// New root is a real dir but read-only, so the promotion's WriteEnvelope
	// (MkdirAll of the instance subdir) fails. ReadResource of the (absent)
	// new file still cleanly misses, so promotion runs and its copy errors.
	newRoot := t.TempDir()
	if err := os.Chmod(newRoot, 0o500); err != nil { //nolint:gosec // directory perms: 0500 intentionally removes write to force the copy failure
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(newRoot, 0o700) }) //nolint:gosec // restore dir perms so t.TempDir cleanup can remove it
	t.Cleanup(SetRootForTest(newRoot))
	t.Cleanup(SetInstanceKeyForTest(migInstance))
	t.Cleanup(SetLegacyRootForTest(legacyDir))
	writeLegacy(t, legacyDir, "boards", validLegacyEnv("boards"))

	if _, err := ReadResource[[]int]("boards"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("a copy failure must be non-fatal (ErrCacheMiss), got %v", err)
	}
}

func TestPromote_NewPresent_LegacyIgnored(t *testing.T) {
	_, legacyDir := migEnv(t)
	// New has [9]; legacy has [1,2,3]. New must win and legacy not consulted.
	if err := WriteResource("boards", "24h", []int{9}); err != nil {
		t.Fatal(err)
	}
	writeLegacy(t, legacyDir, "boards", validLegacyEnv("boards"))

	env, err := ReadResource[[]int]("boards")
	if err != nil {
		t.Fatal(err)
	}
	if len(env.Data) != 1 || env.Data[0] != 9 {
		t.Fatalf("new root must win; got %+v", env.Data)
	}
}

func TestPromote_MalformedLegacy_IsMiss(t *testing.T) {
	_, legacyDir := migEnv(t)
	writeLegacyRaw(t, legacyDir, "boards", "{ not json")

	if _, err := ReadResource[[]int]("boards"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("malformed legacy must be a miss, got %v", err)
	}
}

func TestPromote_IdentityMismatchLegacy_IsMiss(t *testing.T) {
	_, legacyDir := migEnv(t)
	bad := validLegacyEnv("boards")
	bad.Instance = "someone-else.atlassian.net" // wrong instance
	writeLegacy(t, legacyDir, "boards", bad)

	if _, err := ReadResource[[]int]("boards"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("identity-mismatched legacy must be a miss, got %v", err)
	}
}

func TestPromote_VersionMismatchLegacy_IsMiss(t *testing.T) {
	_, legacyDir := migEnv(t)
	bad := validLegacyEnv("boards")
	bad.Version = cccache.Version + 99
	writeLegacy(t, legacyDir, "boards", bad)

	if _, err := ReadResource[[]int]("boards"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("version-mismatched legacy must be a miss, got %v", err)
	}
}

func TestPromote_NeitherPresent_IsMiss(t *testing.T) {
	migEnv(t)
	if _, err := ReadResource[[]int]("boards"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("no legacy + no new must be a miss, got %v", err)
	}
}

func TestPromote_Idempotent_NoRepromoteOverEditedNew(t *testing.T) {
	_, legacyDir := migEnv(t)
	writeLegacy(t, legacyDir, "boards", validLegacyEnv("boards")) // Data [1,2,3]

	if _, err := ReadResource[[]int]("boards"); err != nil { // promote
		t.Fatal(err)
	}
	// User refreshes the new cache to [7]; legacy still [1,2,3].
	if err := WriteResource("boards", "24h", []int{7}); err != nil {
		t.Fatal(err)
	}
	env, err := ReadResource[[]int]("boards")
	if err != nil {
		t.Fatal(err)
	}
	// Exactly [7] (the edited new value) mechanically proves the
	// never-overwrite stat-guard held: had promotion run it would have
	// returned/retained the legacy [1,2,3], which can never appear here.
	if len(env.Data) != 1 || env.Data[0] != 7 {
		t.Fatalf("second read must not re-promote over the edited new cache; got %+v", env.Data)
	}
}

func TestPromote_HermeticWhenNoLegacyOverride(t *testing.T) {
	// rootOverride set but NO SetLegacyRootForTest: legacyRoot()=="" so the
	// real ~/.jtk/cache is never probed (the B3-class leak guard).
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest(migInstance))
	if got := legacyRoot(); got != "" {
		t.Fatalf("legacyRoot() must be \"\" in isolated tests, got %q", got)
	}
	if _, err := ReadResource[[]int]("boards"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("must be a clean miss with no real-dir probe, got %v", err)
	}
}
