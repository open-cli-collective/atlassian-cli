package keyring

import (
	"strings"
	"testing"
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
// is NOT set. The error message format is different (includes "or run
// interactively") so the two cases stay distinguishable in logs.
func TestPassphraseFunc_NonInteractiveOff_NonTTYPath(t *testing.T) {
	SetNonInteractive(false)
	// In the test process os.Stdin may or may not be a TTY depending on
	// where the tests run; the assertion below only fires when it isn't.
	// Both branches (--non-interactive=true above, the non-TTY fallback
	// below) cover the loud-fail surface; the interactive prompt branch
	// can't be unit-tested without a real PTY.
	fn := passphraseFunc("atlassian-cli")
	got, err := fn()
	if err == nil {
		// Real TTY — the prompt would run; nothing to assert here.
		return
	}
	if got != "" {
		t.Fatalf("passphrase must be empty on failure, got %q", got)
	}
	if !strings.Contains(err.Error(), "or run interactively") {
		t.Fatalf("non-TTY fallback error must mention 'or run interactively', got %v", err)
	}
}
