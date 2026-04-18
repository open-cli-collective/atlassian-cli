// Package cache manages the jtk resource cache on disk.
package cache

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

// rootOverride is a package-level override for tests.
// It is set by SetRootForTest and cleared by its cleanup function.
var rootOverride string

// Root returns the cache root directory, expanded from "~/.jtk/cache".
// If SetRootForTest has overridden it, returns the override.
func Root() (string, error) {
	if rootOverride != "" {
		return rootOverride, nil
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
	oldRoot := rootOverride
	rootOverride = dir
	return func() {
		rootOverride = oldRoot
	}
}
