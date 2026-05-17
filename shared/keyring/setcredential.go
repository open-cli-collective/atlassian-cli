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
// key against the bundle allowlist, and stores it in the one canonical
// shared bundle (the ref is a compile-time constant — there is no
// user-facing ref: runtime resolution, migration, show, and clear all
// target the same bundle, so storing elsewhere would be unreadable).
//
// The token is never echoed: it is read, trimmed, and written; no caller
// branch logs or returns it. The one-time §1.8 migration runs first so a
// pre-existing legacy token cannot later collide.
func SetCredential(in io.Reader, key, envVar string) error {
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
		if in == nil {
			return errors.New("no token source: provide it on stdin or use --from-env")
		}
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

	if err := EnsureMigrated(); err != nil {
		return err
	}
	s, err := openCanonical() // the one fixed shared bundle
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	return s.SetToken(key, token)
}
