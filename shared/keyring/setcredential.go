package keyring

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/atlassian-go/credstore"
)

// SetCredentialOpts collects the §1.5.2 ingress inputs surfaced by the
// `set-credential` cobra wrappers. The shared/keyring layer is the only
// place these are validated and dispatched, so both jtk and cfl get the
// same enforcement automatically.
type SetCredentialOpts struct {
	Stdin     io.Reader // bound from cobra opts; required when UseStdin is true
	Ref       string    // explicit value from --ref; empty triggers config-presence defaulting
	Key       string    // explicit value from --key; required (no defaulting)
	FromEnv   string    // env-var name from --from-env (xor with UseStdin)
	UseStdin  bool      // true when --stdin was passed
	Overwrite bool      // true when --overwrite was passed
}

// SetCredentialResult is the §1.5.2 control-plane envelope shape (also the
// pure-library return value). Backend is empty for pre-keyring failures
// (flag validation, config-presence probe) and populated once the keyring
// has been opened and the backend selected.
type SetCredentialResult struct {
	Ref     string `json:"ref"`
	Key     string `json:"key"`
	Backend string `json:"backend"`
	Written bool   `json:"written"`
	Error   string `json:"error,omitempty"`
}

// SetCredentialV2 is the §1.5.2 ingress chokepoint shared by jtk and cfl.
// It validates the flags, resolves --ref defaulting against the shared
// config, opens the keyring (running the §1.8 migration up front via the
// same Open() path used everywhere else), and writes the token. The
// returned result is the JSON envelope (caller decides whether to emit
// it); the error is non-nil whenever Written is false.
//
// The token is never echoed; the value never appears in the Result, the
// returned error, or any log output. Empty / whitespace-only sources are
// rejected with a generic "refusing to store an empty API token" message.
func SetCredentialV2(opts SetCredentialOpts) (SetCredentialResult, error) {
	pre := func(err error) (SetCredentialResult, error) {
		return SetCredentialResult{
			Ref:     opts.Ref,
			Key:     opts.Key,
			Backend: "",
			Written: false,
			Error:   err.Error(),
		}, err
	}

	if opts.Key == "" {
		return pre(fmt.Errorf("--key is required; pass --key %s", KeyAPIToken))
	}
	if !isAllowedKey(opts.Key) {
		return pre(fmt.Errorf("--key %q not supported; today only %s is valid", opts.Key, KeyAPIToken))
	}
	if opts.Ref == "" {
		has, herr := credstore.HasSharedConfig()
		if herr != nil {
			return pre(fmt.Errorf("probe shared config presence: %w", herr))
		}
		if !has {
			return pre(fmt.Errorf("--ref is required when no shared config exists; pass --ref %s", Ref))
		}
		opts.Ref = Ref
	}
	if opts.Ref != Ref {
		return pre(fmt.Errorf("--ref %q not supported; today only %s is valid", opts.Ref, Ref))
	}

	switch {
	case opts.UseStdin && opts.FromEnv != "":
		return pre(errors.New("--stdin and --from-env are mutually exclusive; pick one"))
	case !opts.UseStdin && opts.FromEnv == "":
		return pre(errors.New("no token source: pass --stdin to read from stdin or --from-env VAR"))
	}

	token, err := readToken(opts)
	if err != nil {
		return pre(err)
	}

	s, err := Open()
	if err != nil {
		return SetCredentialResult{
			Ref:     opts.Ref,
			Key:     opts.Key,
			Backend: "",
			Written: false,
			Error:   err.Error(),
		}, err
	}
	backend := storeBackendName(s)
	post := func(err error) (SetCredentialResult, error) {
		if cerr := s.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close keyring %s: %w", s.ref, cerr)
		}
		return SetCredentialResult{
			Ref:     opts.Ref,
			Key:     opts.Key,
			Backend: backend,
			Written: false,
			Error:   err.Error(),
		}, err
	}

	// §1.5.2 default-fail-on-existing: use the atomic strict path so a
	// concurrent writer between an exists-check and a set cannot slip a
	// silent overwrite through. SetTokenStrict surfaces ErrExists from the
	// underlying credstore; translate to the actionable hint.
	var writeErr error
	if opts.Overwrite {
		writeErr = s.SetToken(opts.Key, token)
	} else {
		writeErr = s.SetTokenStrict(opts.Key, token)
		if writeErr != nil && errors.Is(writeErr, cccredstore.ErrExists) {
			return post(fmt.Errorf("entry exists at %s/%s; pass --overwrite to replace", opts.Ref, opts.Key))
		}
	}
	if writeErr != nil {
		return post(writeErr)
	}
	if cerr := s.Close(); cerr != nil {
		return SetCredentialResult{
			Ref: opts.Ref, Key: opts.Key, Backend: backend, Written: false,
			Error: fmt.Errorf("close keyring %s: %w", s.ref, cerr).Error(),
		}, fmt.Errorf("close keyring %s: %w", s.ref, cerr)
	}
	return SetCredentialResult{
		Ref:     opts.Ref,
		Key:     opts.Key,
		Backend: backend,
		Written: true,
	}, nil
}

// readToken pulls the token from the configured source. The value is
// trimmed and validated non-empty; the value itself never appears in any
// returned error message (§1.12).
func readToken(opts SetCredentialOpts) (string, error) {
	var raw string
	if opts.FromEnv != "" {
		v, ok := os.LookupEnv(opts.FromEnv)
		if !ok || strings.TrimSpace(v) == "" {
			return "", fmt.Errorf("environment variable %s is unset or empty", opts.FromEnv)
		}
		raw = v
	} else {
		if opts.Stdin == nil {
			return "", errors.New("--stdin set but stdin reader is nil")
		}
		b, err := io.ReadAll(opts.Stdin)
		if err != nil {
			return "", fmt.Errorf("read API token: %w", err)
		}
		raw = string(b)
	}
	token := strings.TrimSpace(raw)
	if token == "" {
		return "", errors.New("refusing to store an empty API token")
	}
	return token, nil
}

func isAllowedKey(key string) bool {
	for _, k := range allowedKeys {
		if k == key {
			return true
		}
	}
	return false
}

func storeBackendName(s *Store) string {
	if s == nil {
		return ""
	}
	b, _ := s.Backend()
	return string(b)
}

// ErrSetCredentialEnvelopeEmitted is the sentinel returned by
// RunSetCredential when --json was in effect AND the operation failed:
// the JSON envelope has already been written to stdout, so the caller
// (each tool's main.go) MUST NOT double-print the underlying error to
// stderr. Use errors.Is(err, ErrSetCredentialEnvelopeEmitted) to detect.
var ErrSetCredentialEnvelopeEmitted = errors.New("set-credential envelope already written to stdout")

// RunSetCredential is the cobra-layer orchestrator: validates+writes via
// SetCredentialV2, then either emits the §1.5.2 JSON envelope to stdout
// (when emitJSON is true) or prints the one-line stderr success/no-op.
//
// Under emitJSON, on failure the returned error wraps both the
// underlying failure AND ErrSetCredentialEnvelopeEmitted — that sentinel
// is the "stderr stays empty under --json" contract: each tool's main.go
// inspects it and suppresses its usual fmt.Fprintln(stderr, err)
// fallback so the envelope is the sole stdout artifact.
//
// Migration-notice handling under --json: SetCredentialV2 calls the
// migrating Open() path, which may record the §1.8 one-time notice in
// the package sink. Each tool's main.go later calls FlushMigrationNotice
// on stderr, which would contaminate the envelope-only stdout contract.
// To keep stderr clean we drain the sink here under --json. The trade-off
// is that the §1.8 structured `_migration` JSON signal is NOT YET added
// to the §1.5.2 envelope schema — that schema expansion is tracked as a
// family-wide follow-up (matches the current nrq #107 behavior). For
// now, --json runs that perform a migration silently consume the human
// notice; the migration itself still occurred and is reflected in the
// envelope's `written=true`.
func RunSetCredential(opts SetCredentialOpts, stdout, stderr io.Writer, emitJSON bool) error {
	res, err := SetCredentialV2(opts)
	if emitJSON {
		// Drain any pending §1.8 migration notice so main.go's
		// FlushMigrationNotice has nothing left to write to stderr.
		FlushMigrationNotice(io.Discard)

		enc, mErr := json.Marshal(res)
		if mErr != nil {
			return fmt.Errorf("marshal set-credential envelope: %w", mErr)
		}
		if _, werr := fmt.Fprintln(stdout, string(enc)); werr != nil {
			return fmt.Errorf("write set-credential envelope: %w", werr)
		}
		if err != nil {
			return fmt.Errorf("%w (%w)", err, ErrSetCredentialEnvelopeEmitted)
		}
		return nil
	}
	if err != nil {
		return err
	}
	// §1.5.2 verbatim — no tool-name suffix; the binary name already
	// distinguishes jtk from cfl in the user's shell history.
	_, _ = fmt.Fprintf(stderr, "wrote %s to %s via %s\n", res.Key, res.Ref, res.Backend)
	return nil
}

// SetCredential is the pre-§1.5.2 implicit-stdin / silent-overwrite
// ingress shim retained ONLY so the shared/keyring e2e test suite
// (keyring_e2e_test.go) keeps exercising the migrating-Open path through
// the production write chokepoint. Production code (jtk + cfl cobra
// wrappers) calls SetCredentialV2 / RunSetCredential directly. A
// follow-up tracking issue will migrate the e2e tests and delete this
// shim.
//
// Deprecated: use SetCredentialV2 (or RunSetCredential at the cobra layer).
func SetCredential(in io.Reader, envVar string) error {
	_, err := SetCredentialV2(SetCredentialOpts{
		Stdin:     in,
		Ref:       Ref,
		Key:       KeyAPIToken,
		FromEnv:   envVar,
		UseStdin:  envVar == "",
		Overwrite: true,
	})
	return err
}
