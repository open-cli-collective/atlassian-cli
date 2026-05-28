package keyring

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/term"
)

// TestPassphraseFunc_NonInteractiveFailsLoud — under --non-interactive
// the file-backend passphrase callback MUST fail loud asking for the
// env var, regardless of whether stdin is a real TTY. The error message
// must NOT include the "or run interactively" hint that the non-TTY
// path uses, since the user explicitly opted out of interactive mode.
func TestPassphraseFunc_NonInteractiveFailsLoud(t *testing.T) {
	SetNonInteractive(true)
	defer SetNonInteractive(false)

	fn := passphraseFunc("atlassian-cli")
	got, err := fn()
	if err == nil {
		t.Fatalf("expected error, got passphrase %q", got)
	}
	if got != "" {
		t.Fatalf("passphrase must be empty on failure, got %q", got)
	}
	if !strings.Contains(err.Error(), "ATLASSIAN_CLI_KEYRING_PASSPHRASE") {
		t.Fatalf("error must name the env var, got %v", err)
	}
	if !strings.Contains(err.Error(), "--non-interactive") {
		t.Fatalf("error must explain the --non-interactive policy, got %v", err)
	}
	if strings.Contains(err.Error(), "or run interactively") {
		t.Fatalf("under --non-interactive the 'or run interactively' hint is wrong, got %v", err)
	}
}

// TestPassphraseFunc_NonInteractiveOff_NonTTYPath — the non-TTY-stdin
// path (the pre-existing contract) keeps working when --non-interactive
// is NOT set. Skip when os.Stdin IS a real TTY (running tests in a
// terminal) because the callback would enter term.ReadPassword and
// block. The non-interactive-true branch above covers fail-loud; the
// interactive prompt branch requires a PTY harness we don't have here.
func TestPassphraseFunc_NonInteractiveOff_NonTTYPath(t *testing.T) {
	SetNonInteractive(false)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("os.Stdin is a real TTY; the prompt path would block — skipping (covered by the --non-interactive=true branch)")
	}
	fn := passphraseFunc("atlassian-cli")
	got, err := fn()
	if err == nil {
		t.Fatalf("expected non-TTY-fallback error, got passphrase %q", got)
	}
	if got != "" {
		t.Fatalf("passphrase must be empty on failure, got %q", got)
	}
	if !strings.Contains(err.Error(), "or run interactively") {
		t.Fatalf("non-TTY fallback error must mention 'or run interactively', got %v", err)
	}
}
