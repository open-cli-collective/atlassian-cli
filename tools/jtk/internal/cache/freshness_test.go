package cache

import (
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestClassify(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		fetchedAt time.Time
		ttl       string
		want      Status
	}{
		{"fresh within window", now.Add(-1 * time.Hour), "24h", StatusFresh},
		{"stale after window", now.Add(-25 * time.Hour), "24h", StatusStale},
		{"stale exactly at boundary", now.Add(-24 * time.Hour), "24h", StatusStale},
		{"zero fetchedAt reads stale", time.Time{}, "24h", StatusStale},
		{"manual ttl reads manual", now.Add(-999 * time.Hour), "manual", StatusManual},
		{"invalid ttl reads stale", now, "nonsense", StatusStale},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Classify(tt.fetchedAt, tt.ttl, now)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func TestAge(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		fetchedAt time.Time
		want      string
	}{
		{"zero returns dash", time.Time{}, "-"},
		{"seconds", now.Add(-30 * time.Second), "30s"},
		{"minutes", now.Add(-5 * time.Minute), "5m"},
		{"hours", now.Add(-8 * time.Hour), "8h"},
		{"days", now.Add(-3 * 24 * time.Hour), "3d"},
		{"future clamps to zero seconds", now.Add(1 * time.Hour), "0s"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Age(tt.fetchedAt, now)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func TestStatusString(t *testing.T) {
	t.Parallel()
	testutil.Equal(t, StatusUninitialized.String(), "uninitialized")
	testutil.Equal(t, StatusFresh.String(), "fresh")
	testutil.Equal(t, StatusStale.String(), "stale")
	testutil.Equal(t, StatusManual.String(), "manual")
	testutil.Equal(t, StatusUnavailable.String(), "unavailable")
}
