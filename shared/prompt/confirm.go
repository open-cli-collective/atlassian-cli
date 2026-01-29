// Package prompt provides user interaction utilities.
package prompt

import (
	"bufio"
	"io"
	"strings"
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
