package cache

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestTouch_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	err := Touch("nonexistent")
	testutil.NoError(t, err)

	// Verify no file was created.
	_, err = ReadResource[json.RawMessage]("nonexistent")
	if !errors.Is(err, ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss after Touch on missing resource, got %v", err)
	}
}

func TestTouch_Single(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	// Write an envelope with fresh FetchedAt.
	data := []string{"a", "b", "c"}
	err := WriteResource("test-resource", "24h", data)
	testutil.NoError(t, err)

	// Read it back and verify it's fresh.
	env, err := ReadResource[[]string]("test-resource")
	testutil.NoError(t, err)

	if env.FetchedAt.IsZero() {
		t.Fatal("expected fresh envelope before Touch, got zero FetchedAt")
	}

	// Touch it to mark stale.
	err = Touch("test-resource")
	testutil.NoError(t, err)

	// Read it back and verify it's stale (FetchedAt is zero).
	env, err = ReadResource[[]string]("test-resource")
	testutil.NoError(t, err)

	if !env.FetchedAt.IsZero() {
		t.Errorf("expected stale envelope after Touch, got FetchedAt=%v", env.FetchedAt)
	}

	// Verify data is preserved.
	testutil.Equal(t, env.Data, data)

	// Verify status is stale.
	status := Classify(env.FetchedAt, env.TTL, time.Now().UTC())
	if status != StatusStale {
		t.Errorf("expected StatusStale after Touch, got %s", status)
	}
}

func TestTouch_Multi(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	// Write two envelopes.
	err := WriteResource("resource1", "24h", []int{1, 2})
	testutil.NoError(t, err)

	err = WriteResource("resource2", "24h", []string{"x", "y"})
	testutil.NoError(t, err)

	// Touch both in one call.
	err = Touch("resource1", "resource2")
	testutil.NoError(t, err)

	// Verify both are stale.
	env1, err := ReadResource[[]int]("resource1")
	testutil.NoError(t, err)
	if !env1.FetchedAt.IsZero() {
		t.Errorf("resource1: expected stale, got FetchedAt=%v", env1.FetchedAt)
	}

	env2, err := ReadResource[[]string]("resource2")
	testutil.NoError(t, err)
	if !env2.FetchedAt.IsZero() {
		t.Errorf("resource2: expected stale, got FetchedAt=%v", env2.FetchedAt)
	}
}

func TestAppendOnCreate_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	err := AppendOnCreate("nonexistent", "item")
	testutil.NoError(t, err)

	// Verify no file was created.
	_, err = ReadResource[[]string]("nonexistent")
	if !errors.Is(err, ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss after AppendOnCreate on missing resource, got %v", err)
	}
}

func TestAppendOnCreate_Existing(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	// Write an initial envelope.
	initial := []string{"a", "b"}
	err := WriteResource("items", "24h", initial)
	testutil.NoError(t, err)

	// Append an item.
	err = AppendOnCreate("items", "c")
	testutil.NoError(t, err)

	// Read back and verify.
	env, err := ReadResource[[]string]("items")
	testutil.NoError(t, err)

	expected := []string{"a", "b", "c"}
	testutil.Equal(t, env.Data, expected)
}

func TestRemoveOnDelete_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	err := RemoveOnDelete("nonexistent", func(string) bool { return true })
	testutil.NoError(t, err)

	// Verify no file was created.
	_, err = ReadResource[[]string]("nonexistent")
	if !errors.Is(err, ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss after RemoveOnDelete on missing resource, got %v", err)
	}
}

func TestRemoveOnDelete_Matches(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	// Write an initial envelope.
	initial := []string{"a", "b", "c"}
	err := WriteResource("items", "24h", initial)
	testutil.NoError(t, err)

	// Remove items matching "b".
	err = RemoveOnDelete("items", func(s string) bool { return s == "b" })
	testutil.NoError(t, err)

	// Read back and verify.
	env, err := ReadResource[[]string]("items")
	testutil.NoError(t, err)

	expected := []string{"a", "c"}
	testutil.Equal(t, env.Data, expected)
}

func TestRemoveOnDelete_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := SetRootForTest(tmpDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://test.atlassian.net")

	// Write an initial envelope.
	initial := []string{"a", "b"}
	err := WriteResource("items", "24h", initial)
	testutil.NoError(t, err)

	// Get the file's modification time.
	path, err := ResourceFile("items")
	testutil.NoError(t, err)

	stat1, err := os.Stat(path)
	testutil.NoError(t, err)
	mtime1 := stat1.ModTime()

	// Wait a bit to ensure timestamp difference would be observable.
	time.Sleep(10 * time.Millisecond)

	// Remove items matching something that doesn't exist.
	err = RemoveOnDelete("items", func(string) bool { return false })
	testutil.NoError(t, err)

	// Read back and verify data is unchanged.
	env, err := ReadResource[[]string]("items")
	testutil.NoError(t, err)

	testutil.Equal(t, env.Data, initial)

	// Verify the file was not rewritten (by checking mtime is unchanged).
	stat2, err := os.Stat(path)
	testutil.NoError(t, err)
	mtime2 := stat2.ModTime()

	if !mtime1.Equal(mtime2) {
		t.Errorf("expected file to not be rewritten; mtime changed from %v to %v", mtime1, mtime2)
	}
}
