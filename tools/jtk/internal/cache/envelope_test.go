package cache

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestReadResourceWriteResourceRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	data := []int{1, 2, 3}
	err := WriteResource("test-resource", "24h", data)
	testutil.NoError(t, err)

	env, err := ReadResource[[]int]("test-resource")
	testutil.NoError(t, err)

	testutil.Equal(t, env.Resource, "test-resource")
	testutil.Equal(t, env.TTL, "24h")
	testutil.Equal(t, env.Version, 1)
	testutil.Equal(t, env.Data, data)

	// Check FetchedAt is recent (within last minute)
	now := time.Now().UTC()
	age := now.Sub(env.FetchedAt)
	if age < 0 || age > time.Minute {
		t.Errorf("FetchedAt not recent: got %v, want within 1 minute of now", env.FetchedAt)
	}
}

func TestReadResourceCacheMiss(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	env, err := ReadResource[[]int]("nonexistent")

	testutil.Equal(t, env, Envelope[[]int]{})
	if !errors.Is(err, ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss, got %v", err)
	}
}

func TestWriteResourceFileMode(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	data := []string{"test"}
	err := WriteResource("test-resource", "24h", data)
	testutil.NoError(t, err)

	path, err := ResourceFile("test-resource")
	testutil.NoError(t, err)

	stat, err := os.Stat(path)
	testutil.NoError(t, err)

	mode := stat.Mode().Perm()
	expected := os.FileMode(0o600)
	if mode != expected {
		t.Errorf("expected file mode %o, got %o", expected, mode)
	}
}

func TestWriteResourceNoStrayTempFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	data := []string{"test"}
	err := WriteResource("test-resource", "24h", data)
	testutil.NoError(t, err)

	path, err := ResourceFile("test-resource")
	testutil.NoError(t, err)

	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	testutil.NoError(t, err)

	jsonFiles := 0
	tmpFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			if filepath.Ext(entry.Name()) == ".json" {
				jsonFiles++
			}
			if filepath.Ext(entry.Name()) == ".tmp" {
				tmpFiles++
			}
		}
	}

	if jsonFiles != 1 {
		t.Errorf("expected 1 .json file, found %d", jsonFiles)
	}
	if tmpFiles != 0 {
		t.Errorf("expected 0 .tmp files, found %d", tmpFiles)
	}
}

func TestEnvelopeVersion(t *testing.T) {
	if Version != 1 {
		t.Errorf("expected Version == 1, got %d", Version)
	}
}

// A version-mismatched envelope on disk is reported as ErrCacheMiss so the
// next write can overwrite it with the current schema. This keeps schema
// bumps self-healing without surfacing cryptic errors to users.
func TestReadResource_VersionMismatchTreatedAsMiss(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()
	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	// Write a current-version envelope, then overwrite the on-disk file with a
	// bumped version to simulate a future schema.
	testutil.NoError(t, WriteResource("x", "24h", []int{1, 2}))

	path, err := ResourceFile("x")
	testutil.NoError(t, err)
	data, err := os.ReadFile(path) //nolint:gosec // path is a test-owned tempdir
	testutil.NoError(t, err)
	mutated := strings.Replace(string(data), `"version": 1`, `"version": 99`, 1)
	testutil.NoError(t, os.WriteFile(path, []byte(mutated), 0o600)) //nolint:gosec // path is a test-owned tempdir

	env, err := ReadResource[[]int]("x")
	if !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss on version mismatch, got %v", err)
	}
	testutil.Equal(t, env, Envelope[[]int]{})
}
