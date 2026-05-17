package keyring

import (
	"os"
	"strings"
)

// TokenSource describes where a resolved API token came from (for
// `config show`). Never the value.
type TokenSource string

const (
	SourceNone   TokenSource = "unset"
	SourceEnv    TokenSource = "environment"
	SourceKeyAPI TokenSource = "keyring (api_token)"
	SourceKeyCFL TokenSource = "keyring (cfl_api_token)"
	SourceKeyJTK TokenSource = "keyring (jtk_api_token)"
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

func sourceForKey(tool string) TokenSource {
	switch KeyFor(tool) {
	case KeyCFLAPIToken:
		return SourceKeyCFL
	case KeyJTKAPIToken:
		return SourceKeyJTK
	default:
		return SourceKeyAPI
	}
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
		return "", SourceNone, err
	}
	defer func() { _ = s.Close() }()
	return resolveFromStore(s, tool)
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
	return resolveFromStore(s, tool)
}

func resolveFromStore(s *Store, tool string) (string, TokenSource, error) {
	// Per-tool override key first, then the shared default — surfacing the
	// real source for the diagnostic column.
	if k := KeyFor(tool); k != "" {
		if v, ok, err := s.get(k); err != nil {
			return "", SourceNone, err
		} else if ok {
			return v, sourceForKey(tool), nil
		}
	}
	if v, ok, err := s.get(KeyAPIToken); err != nil {
		return "", SourceNone, err
	} else if ok {
		return v, SourceKeyAPI, nil
	}
	return "", SourceNone, nil
}
