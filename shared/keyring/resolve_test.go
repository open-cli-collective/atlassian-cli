package keyring

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"
)

// keyringUnavailableError must wrap a genuine backend-availability error
// (so errors.Is still classifies it) and name every documented runtime
// escape hatch so the headless / no-keyring case (issue #384) is
// self-service. The backend hint is parameterized off the tool. It must
// be a no-op on a nil error.
func TestKeyringUnavailableError_WrapsAndNamesEscapeHatches(t *testing.T) {
	t.Parallel()

	if got := keyringUnavailableError(ToolCFL, nil); got != nil {
		t.Fatalf("nil error must not be wrapped; got %v", got)
	}

	// A genuine backend-availability error (the only class that gets the
	// headless escape-hatch advice). It carries an underlying message we
	// can assert is preserved, not replaced.
	base := fmt.Errorf("secret-service is locked: %w", cccredstore.ErrSecretServiceFailClosed)
	for _, tc := range []struct {
		tool     string
		wantVars []string
	}{
		{ToolCFL, []string{"CFL_API_TOKEN", "ATLASSIAN_API_TOKEN"}},
		{ToolJTK, []string{"JIRA_API_TOKEN", "ATLASSIAN_API_TOKEN"}},
	} {
		err := keyringUnavailableError(tc.tool, base)
		if err == nil {
			t.Fatalf("%s: expected a wrapped error", tc.tool)
		}
		// Underlying error preserved for errors.Is classification.
		if !errors.Is(err, base) {
			t.Fatalf("%s: wrapped error must keep the underlying error; got %v", tc.tool, err)
		}
		msg := err.Error()
		// Names the per-tool and shared env vars (the headless answer:
		// resolved before the keyring is ever opened).
		for _, v := range tc.wantVars {
			if !strings.Contains(msg, v) {
				t.Fatalf("%s: error must name env var %s; got %v", tc.tool, v, err)
			}
		}
		// Names the file + pass backend escape hatches and the passphrase
		// env var (the standard's §1.4 fallbacks).
		for _, want := range []string{"--backend file", "--backend pass", passphraseEnvVar(Service)} {
			if !strings.Contains(msg, want) {
				t.Fatalf("%s: error must name %q; got %v", tc.tool, want, err)
			}
		}
		// The "active backend" hint is parameterized off the tool, not
		// hardcoded — a third tool would otherwise get a wrong `cfl`/`jtk`
		// suggestion (review comment 2).
		if !strings.Contains(msg, "`"+tc.tool+" config show`") {
			t.Fatalf("%s: backend hint must be derived from the tool; got %v", tc.tool, err)
		}
		// Wrapper prose present (we wrap, not replace, the backend message).
		if !strings.Contains(msg, "could not read the API token") {
			t.Fatalf("%s: wrapper prose missing; got %v", tc.tool, err)
		}
	}
}

// keyringUnavailableError must NOT apply the headless / no-keyring escape
// hatches to a NON-backend error (a decode failure, format mismatch,
// ErrStoreClosed, etc.): that advice would be misleading. Such errors are
// returned UNWRAPPED (review comment 1).
func TestKeyringUnavailableError_NonBackendErrorNotWrapped(t *testing.T) {
	t.Parallel()

	for _, base := range []error{
		errors.New("decode api_token: invalid base64 payload"),
		fmt.Errorf("read api_token: %w", cccredstore.ErrStoreClosed),
	} {
		got := keyringUnavailableError(ToolCFL, base)
		// Identity is preserved (returned verbatim, not re-wrapped): the
		// message is byte-for-byte the original and errors.Is still matches.
		if got.Error() != base.Error() {
			t.Fatalf("non-backend error must be returned unwrapped; got %q (want %q)", got, base)
		}
		if !errors.Is(got, base) {
			t.Fatalf("non-backend error must still classify via errors.Is; got %v", got)
		}
		if strings.Contains(got.Error(), "--backend file") ||
			strings.Contains(got.Error(), "no usable keyring") {
			t.Fatalf("non-backend error must not gain keyring escape-hatch advice; got %v", got)
		}
	}
}

// ResolveToken must surface the actionable escape-hatch message — not an
// opaque backend error — when the keyring backend cannot be opened. We
// force a genuine open failure by selecting an unknown backend, which
// cli-common rejects with ErrBackendNotImplemented (OS-independent).
func TestResolveToken_KeyringOpenFailure_WrappedActionable(t *testing.T) {
	hermetic(t)
	// Override the file backend the hermetic harness selects with a bogus
	// one so credstore.Open fails closed before any read/migration.
	t.Setenv(cccredstore.BackendEnvVar(Service), "no-such-backend")

	_, _, err := ResolveToken(ToolJTK)
	if err == nil {
		t.Fatal("expected a keyring-open error")
	}
	// Underlying classification still works.
	if !errors.Is(err, cccredstore.ErrBackendNotImplemented) {
		t.Fatalf("must preserve the underlying backend error for errors.Is; got %v", err)
	}
	msg := err.Error()
	for _, want := range []string{
		"JIRA_API_TOKEN", "ATLASSIAN_API_TOKEN",
		"--backend file", "--backend pass",
		passphraseEnvVar(Service),
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error must name %q so the failure is self-service; got %v", want, err)
		}
	}
}

// The escape hatch the wrapped message advertises must actually work: with
// the API-token env var set, ResolveToken returns it and NEVER opens the
// keyring — so a broken/locked backend (or no keyring at all, the issue
// #384 headless case) is fully bypassed. This is the config-file-present
// user's fix: export the token, don't touch the keystore.
func TestResolveToken_EnvBypassesBrokenKeyring(t *testing.T) {
	hermetic(t)
	// A backend that would fail if opened — proving env short-circuits
	// before any keyring access.
	t.Setenv(cccredstore.BackendEnvVar(Service), "no-such-backend")
	const envTok = "ENV-TOKEN-pqrSTU" //nolint:gosec // G101: test fixture, not a real credential
	t.Setenv("ATLASSIAN_API_TOKEN", envTok)

	tok, src, err := ResolveToken(ToolCFL)
	if err != nil {
		t.Fatalf("env token must bypass the keyring entirely; got error %v", err)
	}
	if tok != envTok {
		t.Fatalf("token = %q, want %q", tok, envTok)
	}
	if src != SourceEnv {
		t.Fatalf("source = %q, want %q", src, SourceEnv)
	}
}

// The companion to the open-failure case: when Open SUCCEEDS but the
// subsequent read fails with a genuine backend error (e.g. the secret
// service goes locked/denied between open and read), ResolveToken must
// still surface the actionable escape-hatch message — not the opaque
// backend error — and still classify via errors.Is. We inject a
// resolveFromStore that returns ErrSecretServiceFailClosed because no
// backend can be made to open cleanly yet fail mid-read on every OS.
func TestResolveToken_StoreReadFailure_WrappedActionable(t *testing.T) {
	hermetic(t)
	// A real, working backend so Open() succeeds and we exercise the
	// post-open read branch (not the open-failure branch).
	if err := SetCredential(strings.NewReader(secret), ""); err != nil {
		t.Fatalf("seed: %v", err)
	}
	orig := resolveFromStore
	t.Cleanup(func() { resolveFromStore = orig })
	resolveFromStore = func(*Store) (string, TokenSource, error) {
		// A backend-availability failure (the locked/denied secret-service
		// case) — exactly the class isBackendError matches, so it is wrapped.
		return "", SourceNone, fmt.Errorf("read api_token: %w", cccredstore.ErrSecretServiceFailClosed)
	}

	_, _, err := ResolveToken(ToolJTK)
	if err == nil {
		t.Fatal("expected a keyring-read error")
	}
	// Underlying classification still works through the wrapper.
	if !errors.Is(err, cccredstore.ErrSecretServiceFailClosed) {
		t.Fatalf("must preserve the underlying backend error for errors.Is; got %v", err)
	}
	msg := err.Error()
	for _, want := range []string{
		"JIRA_API_TOKEN", "ATLASSIAN_API_TOKEN",
		"--backend file", "--backend pass",
		passphraseEnvVar(Service),
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error must name %q so the failure is self-service; got %v", want, err)
		}
	}
}

// The flip side of the predicate (comment 1): a NON-backend read failure
// (e.g. a credential-decode / format error) must propagate UNWRAPPED — it
// must NOT be mislabeled with the headless / no-keyring escape-hatch
// advice, which would be misleading. The underlying error is preserved
// verbatim for errors.Is.
func TestResolveToken_NonBackendReadFailure_NotWrapped(t *testing.T) {
	hermetic(t)
	if err := SetCredential(strings.NewReader(secret), ""); err != nil {
		t.Fatalf("seed: %v", err)
	}
	orig := resolveFromStore
	t.Cleanup(func() { resolveFromStore = orig })
	decodeErr := errors.New("decode api_token: invalid base64 payload")
	resolveFromStore = func(*Store) (string, TokenSource, error) {
		return "", SourceNone, decodeErr
	}

	_, _, err := ResolveToken(ToolCFL)
	if err == nil {
		t.Fatal("expected a read error")
	}
	if !errors.Is(err, decodeErr) {
		t.Fatalf("non-backend error must propagate; got %v", err)
	}
	// The escape-hatch prose must NOT appear: this is not a no-keyring case.
	if strings.Contains(err.Error(), "no usable keyring") ||
		strings.Contains(err.Error(), "--backend file") {
		t.Fatalf("non-backend read failure must not get keyring escape-hatch advice; got %v", err)
	}
}

// Regression guard for the graceful-degradation contract: a corrupt
// shared config must still resolve the token from a WORKING keyring with
// NO error — the escape-hatch wrapper must not leak into that path (it is
// reserved for genuine backend failures).
func TestResolveToken_CorruptSharedConfig_NotWrappedWhenKeyringWorks(t *testing.T) {
	hermetic(t)
	if err := SetCredential(strings.NewReader(secret), ""); err != nil {
		t.Fatalf("seed: %v", err)
	}
	sharedPath := sharedConfigPath(t)
	// Neither valid YAML nor JSON → credstore.Load wraps ErrCorruptStore.
	if err := os.WriteFile(sharedPath, []byte(":::not yaml: ["), 0o600); err != nil {
		t.Fatal(err)
	}

	tok, src, err := ResolveToken(ToolCFL)
	if err != nil {
		t.Fatalf("corrupt shared config with a working keyring must NOT error; got %v", err)
	}
	if tok != secret || src != SourceKeyAPI {
		t.Fatalf("token=%q src=%q, want %q / %q", tok, src, secret, SourceKeyAPI)
	}
}
