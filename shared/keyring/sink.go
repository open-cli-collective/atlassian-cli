package keyring

import (
	"io"
	"sync"
)

// The §1.8 one-time-migration signal. migrateLegacyOverwrite records a
// human-readable line when it actually relocates a token this run; each
// tool's root command flushes it once to stderr via a deferred
// FlushMigrationNotice so the notice survives a later command error.

var (
	sinkMu     sync.Mutex
	sinkNotice string
)

// recordMigration stores the one-time human notice (consume-once).
func recordMigration(humanLine string) {
	sinkMu.Lock()
	defer sinkMu.Unlock()
	sinkNotice = humanLine
}

// FlushMigrationNotice writes the pending §1.8 notice (if any) to w exactly
// once, then clears it. A no-op when nothing migrated this run.
func FlushMigrationNotice(w io.Writer) {
	sinkMu.Lock()
	n := sinkNotice
	sinkNotice = ""
	sinkMu.Unlock()
	if n != "" {
		_, _ = io.WriteString(w, n+"\n")
	}
}

// ResetMigrationNotice clears any pending notice (test seam).
func ResetMigrationNotice() {
	sinkMu.Lock()
	sinkNotice = ""
	sinkMu.Unlock()
}
