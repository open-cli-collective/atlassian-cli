package keyring

import (
	"fmt"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"
)

// Info is the non-secret description of the keyring state for a tool,
// rendered by `config show`. It NEVER carries the token value or any
// prefix/suffix of it — only whether one is configured and from where.
type Info struct {
	Ref              string // canonical bundle ref (a constant)
	Backend          string // e.g. "keychain", "secret-service", "file"
	BackendSource    string // how the backend was selected
	PassphraseSource string // file backend only; "" otherwise
	TokenConfigured  bool   // a token resolves (env or keyring)
	TokenSource      string // display label (TokenSource), never the value
}

// InspectForTool gathers the non-secret keyring description for tool via
// the NON-migrating path (diagnostic). Env is honored for the token
// source (it outranks the keyring at runtime). On an open error the Ref
// (a constant) and any env-derived token source are still returned with
// the error, so `config show` can degrade gracefully.
func InspectForTool(tool string) (Info, error) {
	info := Info{Ref: Ref, TokenSource: string(SourceNone)}

	if v, ok := envToken(tool); ok {
		info.TokenConfigured = v != ""
		info.TokenSource = string(SourceEnv)
	}

	s, err := OpenNoMigrate()
	if err != nil {
		return info, err
	}
	defer func() { _ = s.Close() }()

	b, src := s.Backend()
	info.Backend = fmt.Sprintf("%v", b)
	info.BackendSource = fmt.Sprintf("%v", src)
	// Decide passphrase relevance from the TYPED backend, not its string
	// form — robust if cli-common ever changes Backend.String().
	if b == cccredstore.BackendFile {
		info.PassphraseSource = PassphraseSource(s.Service())
	}

	// Env already won — don't override its source, but a keyring read
	// error must still surface.
	if info.TokenSource == string(SourceEnv) {
		return info, nil
	}
	v, tokSrc, gerr := resolveFromStore(s, tool)
	if gerr != nil {
		return info, gerr
	}
	info.TokenConfigured = v != ""
	info.TokenSource = string(tokSrc)
	return info, nil
}
