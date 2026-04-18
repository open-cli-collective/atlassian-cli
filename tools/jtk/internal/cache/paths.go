// Package cache manages the jtk resource cache on disk.
package cache

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

var ErrNoInstance = errors.New("no Jira instance configured — run 'jtk init' first")

// instanceKeySafe bounds the instance-key character set to the subset we
// emit from hostname (letters, digits, dot, hyphen) and cloud-id (letters,
// digits, hyphen). Any character outside this set — path separators,
// whitespace, control chars — causes InstanceKey to return ErrNoInstance
// rather than compose a path from attacker-controlled input.
var instanceKeySafe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.\-]*$`)

// isSafeInstanceKey validates that the key is safe to use as a filesystem
// path component: only allowed characters, and no parent-dir traversal.
func isSafeInstanceKey(k string) bool {
	if k == "" || !instanceKeySafe.MatchString(k) {
		return false
	}
	// Reject `..` anywhere in the key (subdomains don't have consecutive dots).
	if strings.Contains(k, "..") {
		return false
	}
	return true
}

// rootOverride / instanceOverride are package-level overrides for tests,
// guarded by overrideMu. Tests running in parallel still race for the
// *values* (the last writer wins), but the reads/writes themselves are
// synchronized so the race detector is satisfied.
var (
	overrideMu       sync.RWMutex
	rootOverride     string
	instanceOverride string
)

func getRootOverride() string {
	overrideMu.RLock()
	defer overrideMu.RUnlock()
	return rootOverride
}

func getInstanceOverride() string {
	overrideMu.RLock()
	defer overrideMu.RUnlock()
	return instanceOverride
}

func setRootOverride(v string) {
	overrideMu.Lock()
	rootOverride = v
	overrideMu.Unlock()
}

func setInstanceOverride(v string) {
	overrideMu.Lock()
	instanceOverride = v
	overrideMu.Unlock()
}

// Root returns the cache root directory, expanded from "~/.jtk/cache".
// If SetRootForTest has overridden it, returns the override.
func Root() (string, error) {
	if o := getRootOverride(); o != "" {
		return o, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".jtk", "cache"), nil
}

// InstanceKey derives a per-Jira-instance directory name.
// For basic auth: the hostname of config.GetURL() (e.g., "monit.atlassian.net").
// For bearer auth: config.GetCloudID() when the URL is api.atlassian.com.
// Returns ErrNoInstance if no valid instance configuration is found.
func InstanceKey() (string, error) {
	if o := getInstanceOverride(); o != "" {
		return o, nil
	}
	urlStr := config.GetURL()
	if urlStr == "" {
		return "", ErrNoInstance
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "", ErrNoInstance
	}

	if parsed.Host == "" {
		return "", ErrNoInstance
	}

	// Bearer auth path: use CloudID when gateway is detected
	if parsed.Host == "api.atlassian.com" {
		cloudID := config.GetCloudID()
		if !isSafeInstanceKey(cloudID) {
			return "", ErrNoInstance
		}
		return cloudID, nil
	}

	// Basic auth path: return the hostname
	if !isSafeInstanceKey(parsed.Host) {
		return "", ErrNoInstance
	}
	return parsed.Host, nil
}

// ResourceFile returns the absolute path for a resource's envelope file.
// For example: ~/.jtk/cache/monit.atlassian.net/fields.json
func ResourceFile(name string) (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}

	key, err := InstanceKey()
	if err != nil {
		return "", err
	}

	return filepath.Join(root, key, name+".json"), nil
}

// SetRootForTest overrides the cache root directory for testing.
// Returns a cleanup function that restores the prior value.
// Must only be called from tests in the cache package.
func SetRootForTest(dir string) func() {
	overrideMu.Lock()
	oldRoot := rootOverride
	rootOverride = dir
	overrideMu.Unlock()
	return func() { setRootOverride(oldRoot) }
}

// SetInstanceKeyForTest overrides the derived instance-key name used for
// per-instance cache directories. Pairs with SetRootForTest to give tests a
// fully isolated cache directory without touching JIRA_URL/config state
// (which would conflict with t.Parallel).
func SetInstanceKeyForTest(key string) func() {
	// Defense-in-depth: validate the override with the same character set
	// the production InstanceKey() path enforces. Prevents a test author
	// from accidentally escaping the temp-root isolation set up by
	// SetRootForTest (e.g. "../outside-tmp" would otherwise traverse).
	if !isSafeInstanceKey(key) {
		panic("cache.SetInstanceKeyForTest: unsafe instance key: " + key)
	}
	overrideMu.Lock()
	oldInstance := instanceOverride
	instanceOverride = key
	overrideMu.Unlock()
	return func() { setInstanceOverride(oldInstance) }
}
