package keyring

import (
	"errors"
	"os"
	"strings"
	"testing"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"
)

// keyringUnavailableError must wrap the underlying backend error (so
// errors.Is still classifies it) and name every documented runtime
// escape hatch so the headless / no-keyring case (issue #384) is
// self-service. It must be a no-op on a nil error.
func TestKeyringUnavailableError_WrapsAndNamesEscapeHatches(t *testing.T) {
	t.Parallel()

	if got := keyringUnavailableError(ToolCFL, nil); got != nil {
		t.Fatalf("nil error must not be wrapped; got %v", got)
	}

	base := errors.New("dbus: keyring is locked")
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
		// Never the secret material itself (there is none here, but guard
		// the message shape against a future refactor that interpolates a
		// value).
		if strings.Contains(msg, "dbus: keyring is locked") {
			// The underlying message is allowed (it carries no secret); this
			// assertion just documents that we wrap, not replace, it.
			if !strings.Contains(msg, "could not read the API token") {
				t.Fatalf("%s: wrapper prose missing; got %v", tc.tool, err)
			}
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
