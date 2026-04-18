package present

import (
	"fmt"
	"time"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

// RefreshPresenter builds presentation models for the `jtk refresh` command.
type RefreshPresenter struct{}

// StatusRow is the presenter-level input for one row of the `--status` table.
// The presenter formats FetchedAt / Age / status label — callers only supply
// structured data.
type StatusRow struct {
	Resource  string
	TTL       string
	Status    cache.Status
	FetchedAt time.Time // zero when Status is Uninitialized or Unavailable
}

// PresentStatus builds the `--status` freshness table.
func (RefreshPresenter) PresentStatus(rows []StatusRow, now time.Time) *present.OutputModel {
	tableRows := make([]present.Row, len(rows))
	for i, r := range rows {
		fetchedAt := "-"
		age := "-"
		if !r.FetchedAt.IsZero() {
			fetchedAt = r.FetchedAt.UTC().Format(time.RFC3339)
			age = cache.Age(r.FetchedAt, now)
		}
		tableRows[i] = present.Row{Cells: []string{
			r.Resource, fetchedAt, age, r.TTL, r.Status.String(),
		}}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"RESOURCE", "FETCHED_AT", "AGE", "TTL", "STATUS"},
				Rows:    tableRows,
			},
		},
	}
}

// RefreshResult is the presenter-level input for one per-resource outcome of
// a refresh run. Err is non-nil on failure; on success Count and Previous
// describe the entry-count delta for the "was N" indicator.
type RefreshResult struct {
	Name     string
	Count    int
	Previous int // -1 when prior count is unknown (first-ever fetch or read error)
	Err      error
	At       time.Time
}

// PresentRefresh emits per-resource success/failure lines. Failures go to
// stderr; successes go to stdout.
func (RefreshPresenter) PresentRefresh(results []RefreshResult) *present.OutputModel {
	sections := make([]present.Section, 0, len(results))
	for _, r := range results {
		if r.Err != nil {
			sections = append(sections, &present.MessageSection{
				Kind:    present.MessageError,
				Stream:  present.StreamStderr,
				Message: fmt.Sprintf("Refreshing %s failed: %v", r.Name, r.Err),
			})
			continue
		}
		delta := ""
		if r.Previous >= 0 && r.Previous != r.Count {
			delta = fmt.Sprintf(" (was %d)", r.Previous)
		}
		sections = append(sections, &present.MessageSection{
			Kind:   present.MessageSuccess,
			Stream: present.StreamStdout,
			Message: fmt.Sprintf("Refreshing %s... %d entries%s — Cache updated at %s",
				r.Name, r.Count, delta, r.At.Format("2006-01-02 15:04:05")),
		})
	}
	return &present.OutputModel{Sections: sections}
}
