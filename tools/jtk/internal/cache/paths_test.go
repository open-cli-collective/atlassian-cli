package cache

import (
	"errors"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/open-cli-collective/cli-common/statedir"
	"github.com/open-cli-collective/cli-common/statedirtest"
)

func TestRoot_DefaultExpansion(t *testing.T) {
	// MON-5369: the default cache root is now os.UserCacheDir()/jtk via the
	// shared cli-common resolver (was ~/.jtk/cache). Hermetic so it never
	// touches the developer's real cache dir.
	statedirtest.Hermetic(t)
	cleanup := SetRootForTest("")
	defer cleanup()

	root, err := Root()
	testutil.NoError(t, err)

	want, err := statedir.Cache{Tool: "jtk"}.CacheDir()
	testutil.NoError(t, err)
	testutil.Equal(t, root, want)
}

func TestRoot_RespectSetRootForTest(t *testing.T) {
	statedirtest.Hermetic(t)
	tempDir := t.TempDir()

	// Override the root
	cleanup := SetRootForTest(tempDir)

	// Root should return the override
	root, err := Root()
	testutil.NoError(t, err)
	testutil.Equal(t, root, tempDir)

	// Clean up should restore prior value
	cleanup()

	// After cleanup, Root should return the default (shared resolver) again.
	root, err = Root()
	testutil.NoError(t, err)
	want, err := statedir.Cache{Tool: "jtk"}.CacheDir()
	testutil.NoError(t, err)
	testutil.Equal(t, root, want)
}

func TestInstanceKey_BasicAuth(t *testing.T) {
	t.Setenv("JIRA_URL", "https://monit.atlassian.net")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")
	t.Setenv("JIRA_DOMAIN", "")

	key, err := InstanceKey()
	testutil.NoError(t, err)
	testutil.Equal(t, key, "monit.atlassian.net")
}

func TestInstanceKey_BearerAuth(t *testing.T) {
	t.Setenv("JIRA_URL", "https://api.atlassian.com")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_CLOUD_ID", "abc-123")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")
	t.Setenv("JIRA_DOMAIN", "")

	key, err := InstanceKey()
	testutil.NoError(t, err)
	testutil.Equal(t, key, "abc-123")
}

func TestInstanceKey_NoInstance(t *testing.T) {
	// Clear all URL and CloudID env vars
	t.Setenv("JIRA_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")
	t.Setenv("JIRA_DOMAIN", "")

	// Override HOME so config file can't be found
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	_, err := InstanceKey()
	if !errors.Is(err, ErrNoInstance) {
		t.Errorf("Expected ErrNoInstance, got %v", err)
	}
}

// InstanceKey must reject any value that could escape the cache root when
// composed into a filesystem path — path separators, parent-dir tokens, etc.
// This guards against a malicious JIRA_URL or JIRA_CLOUD_ID planting cache
// files outside ~/.jtk/cache.
func TestInstanceKey_RejectsPathInjection(t *testing.T) {
	cases := []struct {
		name    string
		jiraURL string
		cloudID string
	}{
		// Host with a forward slash would only arrive via a bizarre URL, but
		// we defend anyway.
		{"hostname with parent-dir traversal", "https://../evil", ""},
		{"cloudID with path separator", "https://api.atlassian.com", "../escape"},
		{"cloudID with forward slash", "https://api.atlassian.com", "foo/bar"},
		{"cloudID with backslash", "https://api.atlassian.com", `foo\bar`},
		{"cloudID with space", "https://api.atlassian.com", "foo bar"},
		// Trailing dot: the regex would accept it, so the HasSuffix guard is
		// the sole protection (Windows strips trailing dots → collision).
		{"cloudID with trailing dot", "https://api.atlassian.com", "foo."},
		{"hostname with trailing dot", "https://foo./", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("JIRA_URL", tc.jiraURL)
			t.Setenv("ATLASSIAN_URL", "")
			t.Setenv("JIRA_CLOUD_ID", tc.cloudID)
			t.Setenv("ATLASSIAN_CLOUD_ID", "")
			t.Setenv("JIRA_DOMAIN", "")

			_, err := InstanceKey()
			if !errors.Is(err, ErrNoInstance) {
				t.Fatalf("expected ErrNoInstance for %q / %q, got %v", tc.jiraURL, tc.cloudID, err)
			}
		})
	}
}

// SetInstanceKeyForTest must refuse unsafe keys so callers can't
// accidentally bypass the path-sanitization the production InstanceKey()
// path applies.
func TestSetInstanceKeyForTest_RejectsUnsafeKeys(t *testing.T) {
	cases := []string{
		"../escape",
		"foo/bar",
		`foo\bar`,
		"",
		"foo bar",
		"foo.", // trailing dot — sole-protected by the HasSuffix guard
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic for unsafe key %q", tc)
				}
			}()
			_ = SetInstanceKeyForTest(tc)
		})
	}
}

func TestResourceFile(t *testing.T) {
	tempDir := t.TempDir()
	cleanup := SetRootForTest(tempDir)
	defer cleanup()

	t.Setenv("JIRA_URL", "https://monit.atlassian.net")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")
	t.Setenv("JIRA_DOMAIN", "")

	path, err := ResourceFile("fields")
	testutil.NoError(t, err)

	expected := tempDir + "/monit.atlassian.net/fields.json"
	testutil.Equal(t, path, expected)
}
