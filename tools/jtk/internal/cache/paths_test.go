package cache

import (
	"errors"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestRoot_DefaultExpansion(t *testing.T) {
	// Clear any override
	cleanup := SetRootForTest("")
	defer cleanup()

	root, err := Root()
	testutil.NoError(t, err)

	// Verify it ends with /.jtk/cache
	if !strings.HasSuffix(root, "/.jtk/cache") {
		t.Errorf("Root() should end with /.jtk/cache, got %q", root)
	}
}

func TestRoot_RespectSetRootForTest(t *testing.T) {
	tempDir := t.TempDir()

	// Override the root
	cleanup := SetRootForTest(tempDir)

	// Root should return the override
	root, err := Root()
	testutil.NoError(t, err)
	testutil.Equal(t, root, tempDir)

	// Clean up should restore prior value
	cleanup()

	// After cleanup, Root should return default again
	root, err = Root()
	testutil.NoError(t, err)
	if !strings.HasSuffix(root, "/.jtk/cache") {
		t.Errorf("After cleanup, Root() should end with /.jtk/cache, got %q", root)
	}
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
