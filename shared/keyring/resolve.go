package keyring

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/open-cli-collective/atlassian-go/credstore"
)

// The "corrupt shared store → warn once, keep working" runtime contract:
// a malformed ~/.config config.yml is a CONFIG-FILE problem, not a
// secret-store failure. It must not run (or scrub) the §1.8 migration,
// but it must also not de-authenticate every command — the keyring still
// resolves via the non-migrating path.
//
// State is a mutex-guarded bool rather than a sync.Once: the test seam
// must be able to re-arm it, and reassigning a sync.Once while another
// goroutine may be in .Do() is a data race (the race detector flags it
// under `go test -race` when credtest.Hermetic resets concurrently).
// This mirrors sink.go's sinkMu pattern.
var (
	corruptMu     sync.Mutex
	corruptWarned bool
)

func warnCorruptOnce(err error) {
	corruptMu.Lock()
	defer corruptMu.Unlock()
	if corruptWarned {
		return
	}
	corruptWarned = true
	fmt.Fprintf(os.Stderr,
		"warning: shared config store is unreadable (%v); the one-time keyring migration is deferred. Run `cfl init`/`jtk init` to fix.\n",
		err)
}

// ResetCorruptWarnOnce re-arms the one-shot corrupt-config warning (test
// seam, mirrors ResetMigrationNotice). credtest.Hermetic calls it so a
// test that exercises the corrupt path does not silently suppress the
// warning for every later test in the same process.
func ResetCorruptWarnOnce() {
	corruptMu.Lock()
	corruptWarned = false
	corruptMu.Unlock()
}

// TokenSource describes where a resolved API token came from (for
// `config show`). Never the value.
type TokenSource string

const (
	SourceNone   TokenSource = "unset"
	SourceEnv    TokenSource = "environment"
	SourceKeyAPI TokenSource = "keyring (api_token)"
)

// envVarsFor returns the ordered API-token env vars for a tool: the
// tool-specific var first, then the shared ATLASSIAN_API_TOKEN. Env is
// runtime-only and never persisted.
func envVarsFor(tool string) []string {
	switch tool {
	case ToolCFL:
		return []string{"CFL_API_TOKEN", "ATLASSIAN_API_TOKEN"}
	case ToolJTK:
		return []string{"JIRA_API_TOKEN", "ATLASSIAN_API_TOKEN"}
	default:
		return []string{"ATLASSIAN_API_TOKEN"}
	}
}

func envToken(tool string) (string, bool) {
	for _, name := range envVarsFor(tool) {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v, true
		}
	}
	return "", false
}

// ResolveToken is the RUNTIME token resolver (API commands, `config test`,
// `init` credential need): env wins; otherwise the keyring is opened with
// the one-time §1.8 migration (Open) and the effective key is read. Env
// winning does not force a keyring open — the migration then runs on the
// next invocation that does open it (opportunistic, template-consistent).
// Keyring errors propagate (never folded into "absent").
func ResolveToken(tool string) (string, TokenSource, error) {
	if v, ok := envToken(tool); ok {
		return v, SourceEnv, nil
	}
	s, err := Open()
	if err != nil {
		// A corrupt shared CONFIG file only blocks the migration source;
		// it must not kill the command. Defer migration, warn once, and
		// still resolve the token from the keyring (non-migrating).
		// Genuine keyring-backend errors still propagate.
		if errors.Is(err, credstore.ErrCorruptStore) {
			warnCorruptOnce(err)
			ns, nerr := OpenNoMigrate()
			if nerr != nil {
				return "", SourceNone, nerr
			}
			defer func() { _ = ns.Close() }()
			return resolveFromStore(ns)
		}
		return "", SourceNone, err
	}
	defer func() { _ = s.Close() }()
	return resolveFromStore(s)
}

// ResolveTokenNoMigrate is the DIAGNOSTIC resolver (`config show` source
// column only): env, then a non-migrating keyring read. NOT used by
// `config clear` (which inspects keyring bundle state directly).
func ResolveTokenNoMigrate(tool string) (string, TokenSource, error) {
	if v, ok := envToken(tool); ok {
		return v, SourceEnv, nil
	}
	s, err := OpenNoMigrate()
	if err != nil {
		return "", SourceNone, err
	}
	defer func() { _ = s.Close() }()
	return resolveFromStore(s)
}

// resolveFromStore reads the single shared api_token. One key per logical
// credential (§1.11.10): jtk and cfl resolve the same key.
func resolveFromStore(s *Store) (string, TokenSource, error) {
	if v, ok, err := s.get(KeyAPIToken); err != nil {
		return "", SourceNone, err
	} else if ok {
		return v, SourceKeyAPI, nil
	}
	return "", SourceNone, nil
}
