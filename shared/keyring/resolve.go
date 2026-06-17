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
// Keyring errors propagate (never folded into "absent"), but they are
// wrapped with the headless/no-keyring escape hatches (see
// keyringUnavailableError) so a user on a machine without a usable OS
// keyring — or running the CLI headless under an agent, where the unlock
// prompt never surfaces (issue #384) — gets an actionable, self-service
// message instead of an opaque backend error.
func ResolveToken(tool string) (string, TokenSource, error) {
	if v, ok := envToken(tool); ok {
		return v, SourceEnv, nil
	}
	s, err := Open()
	if err != nil {
		// A corrupt shared CONFIG file only blocks the migration source;
		// it must not kill the command. Defer migration, warn once, and
		// still resolve the token from the keyring (non-migrating).
		// Genuine keyring-backend errors still propagate (wrapped).
		if errors.Is(err, credstore.ErrCorruptStore) {
			warnCorruptOnce(err)
			ns, nerr := OpenNoMigrate()
			if nerr != nil {
				return "", SourceNone, keyringUnavailableError(tool, nerr)
			}
			defer func() { _ = ns.Close() }()
			tok, src, rerr := resolveFromStore(ns)
			if rerr != nil {
				return "", SourceNone, keyringUnavailableError(tool, rerr)
			}
			return tok, src, nil
		}
		return "", SourceNone, keyringUnavailableError(tool, err)
	}
	defer func() { _ = s.Close() }()
	tok, src, rerr := resolveFromStore(s)
	if rerr != nil {
		return "", SourceNone, keyringUnavailableError(tool, rerr)
	}
	return tok, src, nil
}

// keyringUnavailableError wraps a genuine keyring open/read failure with
// the documented escape hatches so the headless / no-keyring case
// (issue #384) is self-service instead of an opaque backend error. It is
// applied ONLY to real backend errors — the §1.8 corrupt-shared-config
// graceful path never reaches it, and a benign "no token present" result
// is a nil error that is never wrapped.
//
// The wrapped error keeps the original via %w, so callers and tests that
// classify with errors.Is (e.g. credstore.ErrSecretServiceFailClosed)
// still match. The message names, in resolution order, the runtime escape
// hatches the Secret-Handling Standard already supports:
//   - the per-tool / shared API-token env vars (resolved before the
//     keyring is ever opened — the canonical headless answer);
//   - the encrypted-file backend (--backend file) and its passphrase env
//     var, plus the pass backend (--backend pass) for password-store users.
//
// No new plaintext path is introduced: the env var and file backend are
// the standard's own §1.4 fallbacks, surfaced here so users discover them
// instead of hitting a silent keyring-unlock prompt that never appears.
func keyringUnavailableError(tool string, err error) error {
	if err == nil {
		return nil
	}
	envVars := strings.Join(envVarsFor(tool), " or ")
	return fmt.Errorf(
		"could not read the API token from the OS keyring (%w). "+
			"If you are running headless or have no usable keyring, supply the token via %s, "+
			"or select the encrypted-file backend with --backend file "+
			"(passphrase via %s) or the pass backend with --backend pass. "+
			"See `cfl config show` / `jtk config show` for the active backend",
		err, envVars, passphraseEnvVar(Service))
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
