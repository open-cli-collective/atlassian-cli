package keyring

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

// SetCredential is the §Ingress write logic, kept here (a pure library)
// so each tool only needs a thin cobra wrapper — shared/ never imports
// cobra. It reads the token from envVar (when non-empty) else from in,
// trims surrounding whitespace, refuses an empty value, validates the
// key against the bundle allowlist, and stores it at ref.
//
// The token is never echoed: it is read, trimmed, and written; no caller
// branch logs or returns it. The default ref runs the one-time §1.8
// migration first (so a pre-existing legacy token cannot later collide);
// an explicit non-default ref does not migrate (it is a distinct bundle).
func SetCredential(in io.Reader, key, envVar, ref string) error {
	if !slices.Contains(allowedKeys, key) {
		return fmt.Errorf("unknown credential key %q (allowed: %s)",
			key, strings.Join(allowedKeys, ", "))
	}

	var raw string
	if strings.TrimSpace(envVar) != "" {
		v, ok := os.LookupEnv(envVar)
		if !ok || strings.TrimSpace(v) == "" {
			return fmt.Errorf("environment variable %s is unset or empty", envVar)
		}
		raw = v
	} else {
		b, err := io.ReadAll(in)
		if err != nil {
			return fmt.Errorf("read API token: %w", err)
		}
		raw = string(b)
	}

	token := strings.TrimSpace(raw)
	if token == "" {
		return errors.New("refusing to store an empty API token")
	}

	useDefaultRef := strings.TrimSpace(ref) == "" || ref == Ref
	if useDefaultRef {
		if err := EnsureMigrated(); err != nil {
			return err
		}
	}

	s, err := OpenRef(ref) // empty ref → canonical shared Ref
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	return s.SetToken(key, token)
}
