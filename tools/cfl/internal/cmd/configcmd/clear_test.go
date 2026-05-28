package configcmd

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/prompt"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func newClearOpts(force bool, stdin string) (*clearOptions, *bytes.Buffer, *bytes.Buffer) {
	out, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	return &clearOptions{
		Options: &root.Options{Output: "table", NoColor: true, Stdout: out, Stderr: errBuf},
		force:   force,
		stdin:   strings.NewReader(stdin),
	}, out, errBuf
}

func tokenPresent(t *testing.T, key string) bool {
	t.Helper()
	s, err := keyring.OpenNoMigrate()
	testutil.RequireNoError(t, err)
	defer func() { _ = s.Close() }()
	ok, err := s.HasToken(key)
	testutil.RequireNoError(t, err)
	return ok
}

func TestRunClear_NothingToClear(t *testing.T) {
	credtest.Hermetic(t)
	opts, _, errBuf := newClearOpts(true, "")
	testutil.RequireNoError(t, runClear(opts))
	testutil.Contains(t, errBuf.String(), "nothing to clear")
}

func TestRunClear_DeletesSharedKey_WithForce(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "shared-secret")

	opts, _, errBuf := newClearOpts(true, "")
	testutil.RequireNoError(t, runClear(opts))

	testutil.False(t, tokenPresent(t, keyring.KeyAPIToken))
	// Shared-default deletion must warn that the sibling loses access.
	testutil.Contains(t, errBuf.String(), "jtk will also lose access")
}

func TestRunClear_DeletesSharedKey_Confirmed(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "shared-secret")

	opts, _, errBuf := newClearOpts(false, "y\n")
	testutil.RequireNoError(t, runClear(opts))

	// One key per logical credential (§1.11.10): a confirmed clear removes
	// the single shared api_token and warns the sibling loses access.
	testutil.False(t, tokenPresent(t, keyring.KeyAPIToken))
	testutil.Contains(t, errBuf.String(), "jtk will also lose access")
	// Removed per-tool override keys must never be advised again.
	testutil.NotContains(t, errBuf.String(), "cfl_api_token")
	testutil.NotContains(t, errBuf.String(), "override")
	// §1.11.11 via the REAL command flow: exactly empty (no stray
	// deprecated key survives a default clear).
	testutil.Equal(t, 0, len(credtest.BundleKeys(t)))
}

func TestRunClear_Cancelled(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "shared-secret")

	opts, _, _ := newClearOpts(false, "n\n")
	testutil.RequireNoError(t, runClear(opts))

	testutil.True(t, tokenPresent(t, keyring.KeyAPIToken))
}

// TestRunClear_NonInteractive_WithoutForce_ShortCircuits — §3.4 contract
// for the destructive config-clear path. The short-circuit fires BEFORE
// keyring.PlanClear so a locked/unavailable keyring can't win first AND
// the warning text never reaches stderr.
func TestRunClear_NonInteractive_WithoutForce_ShortCircuits(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "shared-secret")

	opts, out, errBuf := newClearOpts(false, "")
	opts.NonInteractive = true

	err := runClear(opts)
	if err == nil {
		t.Fatal("expected ErrConfirmationRequired")
	}
	if !errors.Is(err, prompt.ErrConfirmationRequired) {
		t.Fatalf("expected prompt.ErrConfirmationRequired, got %v", err)
	}
	// Critical: no warning text leaks before the short-circuit fires.
	if errBuf.Len() != 0 {
		t.Fatalf("stderr must be empty (no warning text before fail-loud): %q", errBuf.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout must be empty: %q", out.String())
	}
	// Token must remain — clear was rejected, not silently completed.
	testutil.True(t, tokenPresent(t, keyring.KeyAPIToken))
}

// TestRunClear_NonInteractive_WithForce_Proceeds — --force still
// bypasses confirmation under --non-interactive (existing contract).
func TestRunClear_NonInteractive_WithForce_Proceeds(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "shared-secret")

	opts, _, _ := newClearOpts(true, "")
	opts.NonInteractive = true
	testutil.RequireNoError(t, runClear(opts))

	testutil.False(t, tokenPresent(t, keyring.KeyAPIToken))
}

func TestRunClear_All(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "shared-secret")

	sharedPath := credtest.SharedConfigPath(t)
	testutil.RequireNoError(t, os.WriteFile(sharedPath, []byte("default:\n  url: https://x\n"), 0o600))

	opts, _, _ := newClearOpts(true, "")
	opts.all = true
	testutil.RequireNoError(t, runClear(opts))

	testutil.False(t, tokenPresent(t, keyring.KeyAPIToken))
	_, statErr := os.Stat(sharedPath)
	testutil.True(t, os.IsNotExist(statErr))
}
