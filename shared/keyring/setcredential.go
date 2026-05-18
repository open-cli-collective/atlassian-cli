package keyring

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// SetCredential is the §Ingress write logic, kept here (a pure library)
// so each tool only needs a thin cobra wrapper — shared/ never imports
// cobra. It reads the token from envVar (when non-empty) else from in,
// trims surrounding whitespace, refuses an empty value, and stores it as
// the single shared api_token in the one canonical bundle (the ref is a
// compile-time constant and there is one key per logical credential —
// §1.11.10 — so there is no key or ref to choose: runtime resolution,
// migration, show, and clear all target the same bundle).
//
// The token is never echoed: it is read, trimmed, and written; no caller
// branch logs or returns it. The one-time §1.8 migration runs first so a
// pre-existing legacy token cannot later collide.
func SetCredential(in io.Reader, envVar string) (err error) {
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

	// A single migrating open: Open() runs the one-time §1.8 migration
	// (so a pre-existing legacy token cannot later collide) and returns
	// the open canonical bundle. Doing this in one open avoids a second
	// keyring unlock — important for the file backend, which would
	// otherwise prompt for the passphrase twice per invocation.
	s, err := Open()
	if err != nil {
		return err
	}
	// WRITE path: surface the Close error (the file backend may flush on
	// Close) so a swallowed Close after a "successful" SetToken cannot
	// hide a non-durable write.
	defer func() {
		if cerr := s.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("set-credential: close keyring %s: %w", s.ref, cerr)
		}
	}()
	return s.SetToken(KeyAPIToken, token)
}
