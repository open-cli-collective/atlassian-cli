package cache

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	cccache "github.com/open-cli-collective/cli-common/cache"
	"github.com/open-cli-collective/cli-common/statedir"

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
// path component: only allowed characters, no parent-dir traversal, and no
// trailing dot (Windows strips it, which would collide two instances). This
// mirrors cli-common/cache's component guard so the facade and the shared
// library agree on what a valid instance key is.
func isSafeInstanceKey(k string) bool {
	if k == "" || !instanceKeySafe.MatchString(k) {
		return false
	}
	if strings.Contains(k, "..") || strings.HasSuffix(k, ".") {
		return false
	}
	return true
}

// rootOverride / instanceOverride / legacyRootOverride are package-level
// overrides for tests, guarded by overrideMu. Parallel tests still race for
// the values (last writer wins) but the reads/writes are synchronized so the
// race detector is satisfied.
var (
	overrideMu         sync.RWMutex
	rootOverride       string
	instanceOverride   string
	legacyRootOverride string
)

func getRootOverride() string { overrideMu.RLock(); defer overrideMu.RUnlock(); return rootOverride }
func getInstanceOverride() string {
	overrideMu.RLock()
	defer overrideMu.RUnlock()
	return instanceOverride
}
func getLegacyRootOverride() string {
	overrideMu.RLock()
	defer overrideMu.RUnlock()
	return legacyRootOverride
}

func setRootOverride(v string)       { overrideMu.Lock(); rootOverride = v; overrideMu.Unlock() }
func setInstanceOverride(v string)   { overrideMu.Lock(); instanceOverride = v; overrideMu.Unlock() }
func setLegacyRootOverride(v string) { overrideMu.Lock(); legacyRootOverride = v; overrideMu.Unlock() }

// Root returns the cache root directory. It is now os.UserCacheDir()/jtk via
// the shared cli-common resolver (was ~/.jtk/cache; see legacyRoot for the
// one-time promotion source). A test override short-circuits it.
func Root() (string, error) {
	if o := getRootOverride(); o != "" {
		return o, nil
	}
	return statedir.Cache{Tool: "jtk"}.CacheDir()
}

// legacyRoot returns the pre-migration cache root (~/.jtk/cache), used ONLY
// as a read-only promotion source by promoteLegacyOnMiss.
//
// Hermeticity rule: if a root override is set (test mode) but no explicit
// legacy override is, return "" so tests never probe the developer's real
// ~/.jtk/cache (the B3-class leak). Migration tests call
// SetLegacyRootForTest to point the probe at a temp dir.
func legacyRoot() string {
	if o := getLegacyRootOverride(); o != "" {
		return o
	}
	if getRootOverride() != "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".jtk", "cache")
}

// InstanceKey derives a per-Jira-instance directory name.
// Basic auth: the hostname of config.GetURL(). Bearer auth: config.GetCloudID()
// when the URL is the api.atlassian.com gateway. ErrNoInstance if none valid.
func InstanceKey() (string, error) {
	if o := getInstanceOverride(); o != "" {
		return o, nil
	}
	urlStr := config.GetURL()
	if urlStr == "" {
		return "", ErrNoInstance
	}

	parsed, err := url.Parse(urlStr)
	if err != nil || parsed.Host == "" {
		return "", ErrNoInstance
	}

	if parsed.Host == "api.atlassian.com" {
		cloudID := config.GetCloudID()
		if !isSafeInstanceKey(cloudID) {
			return "", ErrNoInstance
		}
		return cloudID, nil
	}

	if !isSafeInstanceKey(parsed.Host) {
		return "", ErrNoInstance
	}
	return parsed.Host, nil
}

// locator builds the cli-common cache Locator from the resolved root +
// instance key. The shared library validates Root (absolute) and the
// instance key / resource name before composing any path.
func locator() (cccache.Locator, error) {
	root, err := Root()
	if err != nil {
		return cccache.Locator{}, err
	}
	key, err := InstanceKey()
	if err != nil {
		return cccache.Locator{}, err
	}
	return cccache.Locator{Root: root, InstanceKey: key}, nil
}

// ResourceFile returns the absolute path for a resource's envelope file under
// the (new) cache root, e.g. <os.UserCacheDir>/jtk/monit.atlassian.net/fields.json.
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

// SetRootForTest overrides the cache root directory for testing. Returns a
// cleanup that restores the prior value. Must only be called from tests.
func SetRootForTest(dir string) func() {
	overrideMu.Lock()
	old := rootOverride
	rootOverride = dir
	overrideMu.Unlock()
	return func() { setRootOverride(old) }
}

// SetInstanceKeyForTest overrides the derived instance-key name. Pairs with
// SetRootForTest to give tests a fully isolated cache dir without touching
// JIRA_URL/config state.
func SetInstanceKeyForTest(key string) func() {
	if !isSafeInstanceKey(key) {
		panic("cache.SetInstanceKeyForTest: unsafe instance key: " + key)
	}
	overrideMu.Lock()
	old := instanceOverride
	instanceOverride = key
	overrideMu.Unlock()
	return func() { setInstanceOverride(old) }
}

// SetLegacyRootForTest points the one-time legacy-promotion probe at a temp
// dir so migration tests are hermetic. Returns a cleanup. Must only be called
// from tests.
func SetLegacyRootForTest(dir string) func() {
	overrideMu.Lock()
	old := legacyRootOverride
	legacyRootOverride = dir
	overrideMu.Unlock()
	return func() { setLegacyRootOverride(old) }
}
