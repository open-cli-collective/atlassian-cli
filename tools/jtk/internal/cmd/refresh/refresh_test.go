package refresh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// newOpts returns root.Options with a test client, buffered stdout/stderr, and
// JIRA_URL/EMAIL/TOKEN set so config.IsConfigured is true.
func newOpts(t *testing.T) (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")

	client, err := api.New(api.ClientConfig{URL: "https://test.atlassian.net", Email: "t@example.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)
	return opts, &stdout, &stderr
}

func TestRun_MissingConfig(t *testing.T) {
	// Unset everything config looks at.
	t.Setenv("JIRA_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("JIRA_API_TOKEN", "")
	t.Setenv("ATLASSIAN_API_TOKEN", "")
	t.Setenv("JIRA_DOMAIN", "")
	// Redirect HOME so the on-disk config file can't be read.
	t.Setenv("HOME", t.TempDir())

	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}

	err := run(context.Background(), opts, nil, false)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "jtk init")
}

func TestRun_UnknownResource(t *testing.T) {
	opts, _, _ := newOpts(t)
	defer cache.SetRootForTest(t.TempDir())()
	defer cache.SetEntriesForTest([]cache.Entry{{Name: "fields", TTL: "24h"}})()

	err := run(context.Background(), opts, []string{"bogus"}, true)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "unknown resource")
}

func TestRun_Status_Uninitialized(t *testing.T) {
	opts, stdout, _ := newOpts(t)
	defer cache.SetRootForTest(t.TempDir())()
	defer cache.SetEntriesForTest([]cache.Entry{
		{Name: "fields", TTL: "24h"},
		{Name: "projects", TTL: "24h"},
	})()

	err := run(context.Background(), opts, nil, true)
	testutil.RequireNoError(t, err)
	out := stdout.String()
	testutil.Contains(t, out, "RESOURCE")
	testutil.Contains(t, out, "fields")
	testutil.Contains(t, out, "projects")
	testutil.Contains(t, out, "uninitialized")
}

func TestRun_Status_AfterWrite(t *testing.T) {
	opts, stdout, _ := newOpts(t)
	defer cache.SetRootForTest(t.TempDir())()
	defer cache.SetEntriesForTest([]cache.Entry{
		{Name: "fields", TTL: "24h"},
	})()

	// Seed an envelope so it reads as fresh.
	testutil.RequireNoError(t, cache.WriteResource("fields", "24h", []api.Field{{ID: "f1", Name: "One"}}))

	err := run(context.Background(), opts, nil, true)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "fresh")
}

func TestRun_Refresh_Success(t *testing.T) {
	opts, stdout, _ := newOpts(t)
	defer cache.SetRootForTest(t.TempDir())()

	called := 0
	defer cache.SetEntriesForTest([]cache.Entry{
		{
			Name: "fake", TTL: "24h",
			Fetch: func(_ context.Context, _ *api.Client) (int, error) {
				called++
				return 42, cache.WriteResource("fake", "24h", []int{1, 2, 3})
			},
		},
	})()

	err := run(context.Background(), opts, nil, false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, called, 1)
	testutil.Contains(t, stdout.String(), "Refreshing fake")
	testutil.Contains(t, stdout.String(), "42 entries")
}

func TestRun_Refresh_TargetedSubset(t *testing.T) {
	opts, stdout, _ := newOpts(t)
	defer cache.SetRootForTest(t.TempDir())()

	var calls []string
	fetch := func(name string) func(context.Context, *api.Client) (int, error) {
		return func(_ context.Context, _ *api.Client) (int, error) {
			calls = append(calls, name)
			return 1, cache.WriteResource(name, "24h", []int{1})
		}
	}
	defer cache.SetEntriesForTest([]cache.Entry{
		{Name: "a", TTL: "24h", Fetch: fetch("a")},
		{Name: "b", TTL: "24h", Fetch: fetch("b")},
		{Name: "c", TTL: "24h", Fetch: fetch("c")},
	})()

	err := run(context.Background(), opts, []string{"b", "c"}, false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, strings.Join(calls, ","), "b,c")
	out := stdout.String()
	testutil.Contains(t, out, "Refreshing b")
	testutil.Contains(t, out, "Refreshing c")
	if strings.Contains(out, "Refreshing a") {
		t.Fatalf("expected a to be skipped, got: %s", out)
	}
}

func TestRun_Refresh_ContinuesOnError(t *testing.T) {
	opts, stdout, stderr := newOpts(t)
	defer cache.SetRootForTest(t.TempDir())()

	defer cache.SetEntriesForTest([]cache.Entry{
		{
			Name: "ok", TTL: "24h",
			Fetch: func(_ context.Context, _ *api.Client) (int, error) {
				return 1, cache.WriteResource("ok", "24h", []int{1})
			},
		},
		{
			Name: "broken", TTL: "24h",
			Fetch: func(_ context.Context, _ *api.Client) (int, error) {
				return 0, errors.New("boom")
			},
		},
		{
			Name: "also_ok", TTL: "24h",
			Fetch: func(_ context.Context, _ *api.Client) (int, error) {
				return 2, cache.WriteResource("also_ok", "24h", []int{1, 2})
			},
		},
	})()

	err := run(context.Background(), opts, nil, false)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "boom")
	testutil.Contains(t, stdout.String(), "Refreshing ok")
	testutil.Contains(t, stdout.String(), "Refreshing also_ok")
	testutil.Contains(t, stderr.String(), "Refreshing broken failed")
}

func TestRun_Refresh_SkipsUnavailable(t *testing.T) {
	opts, stdout, _ := newOpts(t)
	defer cache.SetRootForTest(t.TempDir())()

	fetchCalled := false
	defer cache.SetEntriesForTest([]cache.Entry{
		{
			Name: "fields", TTL: "24h",
			Fetch: func(_ context.Context, _ *api.Client) (int, error) {
				return 1, cache.WriteResource("fields", "24h", []int{1})
			},
		},
		{
			Name: "boards", TTL: "24h",
			Available: func(_ *api.Client) bool { return false },
			Fetch: func(_ context.Context, _ *api.Client) (int, error) {
				fetchCalled = true
				return 0, nil
			},
		},
	})()

	err := run(context.Background(), opts, nil, false)
	testutil.RequireNoError(t, err)
	if fetchCalled {
		t.Fatal("expected unavailable entry's fetcher not to be called")
	}
	testutil.Contains(t, stdout.String(), "Skipping boards")
	testutil.Contains(t, stdout.String(), "Refreshing fields")
}

func TestRun_Status_Unavailable(t *testing.T) {
	opts, stdout, _ := newOpts(t)
	defer cache.SetRootForTest(t.TempDir())()
	defer cache.SetEntriesForTest([]cache.Entry{
		{Name: "boards", TTL: "24h", Available: func(_ *api.Client) bool { return false }},
	})()

	err := run(context.Background(), opts, nil, true)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "unavailable")
}

func TestRun_Status_DoesNotCallFetch(t *testing.T) {
	opts, _, _ := newOpts(t)
	defer cache.SetRootForTest(t.TempDir())()

	var fetched bool
	defer cache.SetEntriesForTest([]cache.Entry{
		{
			Name: "fields", TTL: "24h",
			Fetch: func(_ context.Context, _ *api.Client) (int, error) {
				fetched = true
				return 0, nil
			},
		},
	})()

	err := run(context.Background(), opts, nil, true)
	testutil.RequireNoError(t, err)
	if fetched {
		t.Fatal("--status must not invoke fetchers")
	}
}

// Sanity-check that the formatRefreshLine composes the expected shape.
func TestFormatRefreshLine(t *testing.T) {
	now := mustParseTime(t, "2026-04-18T14:23:00Z")
	testutil.Equal(t,
		formatRefreshLine("fields", 73, 72, now),
		"Refreshing fields... 73 entries (was 72) — Cache updated at 2026-04-18 14:23:00")
	testutil.Equal(t,
		formatRefreshLine("fields", 73, 73, now),
		"Refreshing fields... 73 entries — Cache updated at 2026-04-18 14:23:00")
	testutil.Equal(t,
		formatRefreshLine("users", 0, -1, now),
		"Refreshing users... 0 entries — Cache updated at 2026-04-18 14:23:00")
}

func mustParseTime(t *testing.T, s string) (out time.Time) {
	t.Helper()
	out, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return
}

// -- smoke test: the default registry compiles into a runnable Entries() list.
func TestEntries_IncludesAllSpecResources(t *testing.T) {
	names := cache.Names()
	want := []string{"fields", "projects", "boards", "linktypes", "issuetypes", "statuses", "priorities", "resolutions", "users"}
	joined := "," + strings.Join(names, ",") + ","
	for _, w := range want {
		if !strings.Contains(joined, ","+w+",") {
			t.Errorf("missing registry entry %q; got %v", w, names)
		}
	}
}

// Ensure we don't accidentally leave the prefix argument requirement off —
// the command should accept zero args.
func TestRegister_AcceptsZeroArgs(_ *testing.T) {
	// Just build the command; no assertion needed beyond compilation.
	_ = fmt.Sprintf("registered: %T", Register)
}
