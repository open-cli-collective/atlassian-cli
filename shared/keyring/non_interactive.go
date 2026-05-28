package keyring

import "sync"

// SetNonInteractive wires the root-command --non-interactive policy into
// the package state consulted by the file-backend passphrase callback.
//
// Mirrors the SetBackendSelection pattern at keyring.go:28. Callers
// (each tool's root PersistentPreRunE) invoke this once before any
// Open* runs; the mutex guards against surprise concurrent callers
// (test code rebuilding the command tree, future goroutine-launching
// subcommands).
func SetNonInteractive(nonInteractive bool) {
	nonInteractiveMu.Lock()
	defer nonInteractiveMu.Unlock()
	selectedNonInteractive = nonInteractive
}

// GetNonInteractive returns the current package-level non-interactive
// state. Consumed by passphraseFunc to fail loud on --non-interactive
// even when stdin is a real TTY.
func GetNonInteractive() bool {
	nonInteractiveMu.RLock()
	defer nonInteractiveMu.RUnlock()
	return selectedNonInteractive
}

var (
	nonInteractiveMu       sync.RWMutex
	selectedNonInteractive bool
)
