// Package mutation provides a shared write-then-fetch-then-present helper
// for non-destructive CLI mutations.  After a successful write the helper
// re-fetches the entity with bounded retries so the user sees post-state
// output that mirrors the corresponding "get" command.
package mutation

import (
	"context"
	"strings"
	"time"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

// Config configures a write → fetch → present cycle.
type Config struct {
	// Write executes the mutation.  Returns the entity identifier
	// (issue key, project key, comment ID, …) consumed by Fetch and
	// Fallback.
	Write func(ctx context.Context) (id string, err error)

	// Fetch retrieves the entity after the write and builds its
	// presentation model.  Called immediately after Write succeeds,
	// then retried on error or staleness up to len(BackoffSchedule)−1
	// times.
	Fetch func(ctx context.Context, id string) (*present.OutputModel, error)

	// IsFresh inspects the fetched model and returns true when it
	// reflects the write.  When nil every successful fetch is accepted.
	// When non-nil and false the fetch is retried.
	IsFresh func(model *present.OutputModel) bool

	// Fallback builds the model emitted when every fetch attempt fails.
	// Receives the entity ID from Write.  Must preserve the full
	// semantic context of the mutation (assign vs unassign, create URL, …).
	Fallback func(id string) *present.OutputModel
}

// BackoffSchedule is the fixed retry schedule for post-write fetches.
// Package-level so tests can override it to zero-duration entries.
var BackoffSchedule = []time.Duration{
	0,
	200 * time.Millisecond,
	500 * time.Millisecond,
	1 * time.Second,
}

// WriteAndPresent executes a mutation, fetches the post-state with
// bounded retries, and emits the result.  The write error is fatal;
// the fetch error is non-fatal (mutation succeeded → exit 0).
func WriteAndPresent(ctx context.Context, opts *root.Options, cfg Config) error {
	id, err := cfg.Write(ctx)
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{id})
	}

	var lastModel *present.OutputModel

	for i, delay := range BackoffSchedule {
		if i > 0 && delay > 0 {
			select {
			case <-ctx.Done():
				return emitBestAvailable(opts, lastModel, cfg.Fallback, id)
			case <-time.After(delay):
			}
		}

		model, fetchErr := cfg.Fetch(ctx, id)
		if fetchErr != nil {
			if ctx.Err() != nil {
				break
			}
			continue
		}

		lastModel = model

		if cfg.IsFresh == nil || cfg.IsFresh(model) {
			return jtkpresent.Emit(opts, model)
		}
	}

	return emitBestAvailable(opts, lastModel, cfg.Fallback, id)
}

// emitBestAvailable emits the last fetched model if available (stale but real
// data), otherwise falls back to the Fallback builder plus a stderr advisory.
func emitBestAvailable(opts *root.Options, lastModel *present.OutputModel, fallback func(string) *present.OutputModel, id string) error {
	if lastModel != nil {
		return jtkpresent.Emit(opts, lastModel)
	}

	advisory := jtkpresent.MutationPresenter{}.Advisory("post-state unavailable; showing confirmation only")
	_ = jtkpresent.Emit(opts, advisory)

	return jtkpresent.Emit(opts, fallback(id))
}

// ModelContainsStatus checks whether an issue detail OutputModel contains
// the given status name in its "Status: <name>" line.  Issue detail models
// are built from MessageSection lines (via msg() in present/issue.go),
// not KVSection pairs.
func ModelContainsStatus(model *present.OutputModel, targetStatus string) bool {
	prefix := "Status: " + targetStatus
	for _, section := range model.Sections {
		if ms, ok := section.(*present.MessageSection); ok {
			if strings.Contains(ms.Message, prefix) {
				return true
			}
		}
	}
	return false
}
