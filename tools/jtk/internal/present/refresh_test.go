package present

import (
	"errors"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

func TestRefreshPresenter_PresentStatus(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 18, 14, 23, 0, 0, time.UTC)
	model := RefreshPresenter{}.PresentStatus([]StatusRow{
		{Resource: "fields", TTL: "24h", FetchedAt: now.Add(-8 * time.Hour), Status: cache.StatusFresh},
		{Resource: "users", TTL: "24h", Status: cache.StatusUninitialized},
		{Resource: "boards", TTL: "24h", Status: cache.StatusUnavailable},
	}, now)

	if len(model.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(model.Sections))
	}
	table, ok := model.Sections[0].(*present.TableSection)
	if !ok {
		t.Fatalf("expected TableSection, got %T", model.Sections[0])
	}
	if got := table.Headers; len(got) != 5 || got[0] != "RESOURCE" || got[4] != "STATUS" {
		t.Fatalf("unexpected headers: %v", got)
	}

	// Fresh row
	if got := table.Rows[0].Cells; got[0] != "fields" || got[2] != "8h" || got[4] != "fresh" {
		t.Fatalf("fields row: %v", got)
	}
	// Uninitialized row: dashes for FetchedAt and Age
	if got := table.Rows[1].Cells; got[1] != "-" || got[2] != "-" || got[4] != "uninitialized" {
		t.Fatalf("users row: %v", got)
	}
	// Unavailable row
	if got := table.Rows[2].Cells; got[4] != "unavailable" {
		t.Fatalf("boards row: %v", got)
	}
}

func TestRefreshPresenter_PresentRefresh_SuccessAndFailure(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 4, 18, 14, 23, 0, 0, time.UTC)
	model := RefreshPresenter{}.PresentRefresh([]RefreshResult{
		{Name: "fields", Count: 73, Previous: 72, At: at},
		{Name: "users", Count: 0, Previous: -1, At: at},
		{Name: "boards", Err: errors.New("boom"), At: at},
	})

	if len(model.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(model.Sections))
	}

	s0 := model.Sections[0].(*present.MessageSection)
	if s0.Kind != present.MessageSuccess || s0.Stream != present.StreamStdout {
		t.Fatalf("expected success on stdout, got kind=%v stream=%v", s0.Kind, s0.Stream)
	}
	if s0.Message != "Refreshing fields... 73 entries (was 72) — Cache updated at 2026-04-18 14:23:00" {
		t.Fatalf("success message: %q", s0.Message)
	}

	// No delta when previous is unknown (-1).
	s1 := model.Sections[1].(*present.MessageSection)
	if s1.Message != "Refreshing users... 0 entries — Cache updated at 2026-04-18 14:23:00" {
		t.Fatalf("no-delta message: %q", s1.Message)
	}

	// Failure goes to stderr.
	s2 := model.Sections[2].(*present.MessageSection)
	if s2.Kind != present.MessageError || s2.Stream != present.StreamStderr {
		t.Fatalf("expected error on stderr, got kind=%v stream=%v", s2.Kind, s2.Stream)
	}
	if s2.Message != "Refreshing boards failed: boom" {
		t.Fatalf("failure message: %q", s2.Message)
	}
}

// Delta is suppressed when previous count equals current count.
func TestRefreshPresenter_PresentRefresh_NoDeltaWhenEqual(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 4, 18, 14, 23, 0, 0, time.UTC)
	model := RefreshPresenter{}.PresentRefresh([]RefreshResult{
		{Name: "fields", Count: 73, Previous: 73, At: at},
	})
	s := model.Sections[0].(*present.MessageSection)
	if s.Message != "Refreshing fields... 73 entries — Cache updated at 2026-04-18 14:23:00" {
		t.Fatalf("equal-count message: %q", s.Message)
	}
}
