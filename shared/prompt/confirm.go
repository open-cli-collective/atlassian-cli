// Package prompt provides user interaction utilities.
package prompt

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Confirm prompts the user for confirmation and returns true if they answer yes.
// It reads a single line from stdin and returns true if the answer is "y" or "Y".
func Confirm(stdin io.Reader) (bool, error) {
	scanner := bufio.NewScanner(stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(scanner.Text())
		return answer == "y" || answer == "Y", nil
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	// EOF with no input means no confirmation
	return false, nil
}

// ConfirmOrForce returns true if force is true, or if the user confirms interactively.
// If force is true, it returns immediately without reading from stdin.
func ConfirmOrForce(force bool, stdin io.Reader) (bool, error) {
	if force {
		return true, nil
	}
	return Confirm(stdin)
}

// ErrConfirmationRequired is the §3.4 sentinel: destructive operations
// under --non-interactive without --force MUST fail loud rather than
// block on stdin or silently cancel. The text is the actionable hint.
var ErrConfirmationRequired = errors.New("--non-interactive: confirmation required; re-run with --force to proceed")

// ConfirmOrFail upgrades ConfirmOrForce with the §3.4 non-interactive
// gate. Precedence: --force wins (returns true); --non-interactive
// without --force returns ErrConfirmationRequired; otherwise prompts
// via Confirm.
func ConfirmOrFail(force, nonInteractive bool, stdin io.Reader) (bool, error) {
	if force {
		return true, nil
	}
	if nonInteractive {
		return false, ErrConfirmationRequired
	}
	return Confirm(stdin)
}

// WantPrompt reports whether interactive prompts should run. Returns
// false under --non-interactive (regardless of stdin) AND when stdin is
// not a TTY. Used by init wizards to choose between a huh form (TTY +
// interactive) and a fail-loud validator (non-interactive or non-TTY).
func WantPrompt(nonInteractive bool, stdin io.Reader) bool {
	if nonInteractive {
		return false
	}
	return isTerminal(stdin)
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
