package prompt

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadSecretFromIngress returns the token from exactly one of stdin
// (when useStdin is true) or the named env var (when fromEnv is
// non-empty), trimmed and validated non-empty. Returns ("", nil) when
// neither ingress is configured so the caller can fall through to the
// next resolution step (keyring backfill, interactive form, or §3.4
// non-interactive fail).
//
// The token is never echoed; the value never appears in any returned
// error message. Symmetric to keyring.readToken (which lives in
// shared/keyring with the write-context and stays unexported there).
func ReadSecretFromIngress(stdin io.Reader, useStdin bool, fromEnv string) (string, error) {
	switch {
	case useStdin && fromEnv != "":
		return "", errors.New("--token-stdin and --token-from-env are mutually exclusive; pick one")
	case useStdin:
		if stdin == nil {
			return "", errors.New("--token-stdin set but stdin reader is nil")
		}
		b, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("read API token from stdin: %w", err)
		}
		token := strings.TrimSpace(string(b))
		if token == "" {
			return "", errors.New("refusing to store an empty API token (stdin)")
		}
		return token, nil
	case fromEnv != "":
		v, ok := os.LookupEnv(fromEnv)
		if !ok || strings.TrimSpace(v) == "" {
			return "", fmt.Errorf("environment variable %s is unset or empty", fromEnv)
		}
		return strings.TrimSpace(v), nil
	}
	return "", nil
}
