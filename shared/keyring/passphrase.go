package keyring

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// passphraseEnvVar is the §1.4 named exception: <SERVICE>_KEYRING_PASSPHRASE,
// SERVICE being the upper-snake-cased service segment (atlassian-cli ->
// ATLASSIAN_CLI_KEYRING_PASSPHRASE). Service segments are [A-Za-z0-9_-], so
// only '-' needs translating.
func passphraseEnvVar(service string) string {
	return strings.ToUpper(strings.ReplaceAll(service, "-", "_")) + "_KEYRING_PASSPHRASE"
}

// passphraseFunc is credstore Options.FilePassphrase: consulted only for the
// encrypted-file backend, and only after credstore has already checked
// <SERVICE>_KEYRING_PASSPHRASE itself. So this is the interactive fallback:
// a no-echo TTY prompt. Headless with no env var set is a hard, actionable
// error — never a silent empty passphrase (that would create an
// effectively-unencrypted keyring).
func passphraseFunc(service string) func() (string, error) {
	return func() (string, error) {
		// Constraint: the file backend's interactive passphrase prompt
		// needs a TTY on stdin. When the token itself is piped in (e.g.
		// `echo tok | cfl set-credential`), stdin is the token stream and
		// not a terminal, so this falls to the headless error — the user
		// must supply ATLASSIAN_CLI_KEYRING_PASSPHRASE in that case (the
		// error message says so). Token delivery and passphrase entry
		// cannot share stdin.
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return "", fmt.Errorf(
				"file keyring backend needs a passphrase: set %s, or run interactively",
				passphraseEnvVar(service))
		}
		fmt.Fprintf(os.Stderr, "Passphrase for the %s file keyring: ", service)
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("read passphrase: %w", err)
		}
		p := strings.TrimRight(string(b), "\r\n")
		if p == "" {
			return "", fmt.Errorf("empty passphrase rejected")
		}
		return p, nil
	}
}

// PassphraseSource describes, for `config show`, where the file-backend
// passphrase would come from (§1.4). Only meaningful when the file backend
// is in use.
func PassphraseSource(service string) string {
	if os.Getenv(passphraseEnvVar(service)) != "" {
		return "env (" + passphraseEnvVar(service) + ")"
	}
	return "interactive prompt"
}
