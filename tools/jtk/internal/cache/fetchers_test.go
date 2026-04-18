package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// newTestClient builds an api.Client pointed at `server.URL`, plus the standard
// cache-isolation plumbing (tempdir root, JIRA_URL env). Use this for any
// fetcher test that needs a live HTTP mock.
func newTestClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")
	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@example.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	return client
}

func TestFetchIssueTypes_MissingProjectsCache(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no API calls should be made when projects cache is absent")
	}))
	defer server.Close()

	client := newTestClient(t, server)

	_, err := fetchIssueTypes(context.Background(), client)
	testutil.Error(t, err)
	// Surfaces a clear hint per fetchers.go.
	if !strings.Contains(err.Error(), "refresh projects first") {
		t.Fatalf("expected error to mention 'refresh projects first', got: %v", err)
	}
	// Envelope must not have been written.
	if _, err := ReadResource[map[string][]api.IssueType]("issuetypes"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected no issuetypes envelope on disk, got err=%v", err)
	}
}

func TestFetchIssueTypes_MultiProject(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	// Seed the projects cache with two projects. Note: /rest/api/3/project/{key}
	// returns a full ProjectDetail with an `issueTypes` field; the api method
	// extracts it. See api/move.go:GetProjectIssueTypes.
	calls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		switch r.URL.Path {
		case "/rest/api/3/project/MON":
			_, _ = w.Write([]byte(`{"issueTypes":[{"id":"1","name":"Task"},{"id":"2","name":"Epic"}]}`))
		case "/rest/api/3/project/ON":
			_, _ = w.Write([]byte(`{"issueTypes":[{"id":"3","name":"Sub-task"}]}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)

	testutil.RequireNoError(t, WriteResource("projects", "24h", []api.Project{
		{Key: "MON", Name: "Platform"},
		{Key: "ON", Name: "Onboarding"},
	}))

	count, err := fetchIssueTypes(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, count, 3) // 2 + 1
	testutil.Equal(t, calls["/rest/api/3/project/MON"], 1)
	testutil.Equal(t, calls["/rest/api/3/project/ON"], 1)

	env, err := ReadResource[map[string][]api.IssueType]("issuetypes")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(env.Data["MON"]), 2)
	testutil.Equal(t, len(env.Data["ON"]), 1)
	testutil.Equal(t, env.Data["MON"][0].Name, "Task")
	testutil.Equal(t, env.Data["ON"][0].Name, "Sub-task")
}

func TestFetchIssueTypes_PerProjectAPIError(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/project/OK":
			_, _ = w.Write([]byte(`{"issueTypes":[{"id":"1","name":"Task"}]}`))
		case "/rest/api/3/project/BROKEN":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errorMessages":["boom"]}`))
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)

	testutil.RequireNoError(t, WriteResource("projects", "24h", []api.Project{
		{Key: "OK"},
		{Key: "BROKEN"},
	}))

	_, err := fetchIssueTypes(context.Background(), client)
	testutil.Error(t, err)
	if !strings.Contains(err.Error(), "BROKEN") {
		t.Fatalf("expected error to name the failing project, got: %v", err)
	}
	// Partial results must not be persisted: no issuetypes envelope.
	if _, err := ReadResource[map[string][]api.IssueType]("issuetypes"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected no issuetypes envelope after partial failure, got err=%v", err)
	}
}

func TestFetchStatuses_MultiProject(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/project/MON/statuses":
			_, _ = w.Write([]byte(`[{"id":"10","name":"Epic","subtask":false,"statuses":[{"id":"1","name":"To Do"},{"id":"2","name":"Done"}]}]`))
		case "/rest/api/3/project/ON/statuses":
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)

	testutil.RequireNoError(t, WriteResource("projects", "24h", []api.Project{{Key: "MON"}, {Key: "ON"}}))

	count, err := fetchStatuses(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, count, 1) // one top-level issue type on MON; ON is empty

	env, err := ReadResource[map[string][]api.ProjectStatus]("statuses")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(env.Data["MON"]), 1)
	testutil.Equal(t, len(env.Data["ON"]), 0)
	testutil.Equal(t, env.Data["MON"][0].Name, "Epic")
	testutil.Equal(t, len(env.Data["MON"][0].Statuses), 2)
}

func TestFetchStatuses_MissingProjectsCache(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no API calls should be made when projects cache is absent")
	}))
	defer server.Close()

	client := newTestClient(t, server)

	_, err := fetchStatuses(context.Background(), client)
	testutil.Error(t, err)
	if !strings.Contains(err.Error(), "refresh projects first") {
		t.Fatalf("expected error to mention 'refresh projects first', got: %v", err)
	}
}

func TestFetchBoards_Pagination(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	// Simulate three pages: 50, 50, 20 boards, with isLast=false,false,true.
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/agile/1.0/board")
		startAt, _ := strconv.Atoi(r.URL.Query().Get("startAt"))
		maxResults, _ := strconv.Atoi(r.URL.Query().Get("maxResults"))
		testutil.Equal(t, maxResults, 50)
		calls++

		var values []api.Board
		var isLast bool
		switch startAt {
		case 0:
			for i := 0; i < 50; i++ {
				values = append(values, api.Board{ID: i + 1, Name: "b" + strconv.Itoa(i+1), Type: "scrum"})
			}
		case 50:
			for i := 50; i < 100; i++ {
				values = append(values, api.Board{ID: i + 1, Name: "b" + strconv.Itoa(i+1), Type: "scrum"})
			}
		case 100:
			for i := 100; i < 120; i++ {
				values = append(values, api.Board{ID: i + 1, Name: "b" + strconv.Itoa(i+1), Type: "scrum"})
			}
			isLast = true
		default:
			t.Errorf("unexpected startAt: %d", startAt)
		}
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{
			StartAt:    startAt,
			MaxResults: maxResults,
			IsLast:     isLast,
			Values:     values,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)

	count, err := fetchBoards(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, count, 120)
	testutil.Equal(t, calls, 3)

	env, err := ReadResource[[]api.Board]("boards")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(env.Data), 120)
	testutil.Equal(t, env.Data[0].ID, 1)
	testutil.Equal(t, env.Data[119].ID, 120)
}

// A misbehaving server that never sets IsLast=true must not spin the fetcher
// forever. fetchBoardsMax caps iteration.
func TestFetchBoards_IterationCeiling(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	const pageSize = 50
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		values := make([]api.Board, pageSize)
		for i := range values {
			values[i] = api.Board{ID: calls*1000 + i}
		}
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{IsLast: false, Values: values})
	}))
	defer server.Close()

	client := newTestClient(t, server)

	_, err := fetchBoards(context.Background(), client)
	testutil.RequireNoError(t, err)

	// Each page returns pageSize entries and IsLast=false; loop should exit
	// when startAt reaches fetchBoardsMax. With pageSize=50 and max=5000, that
	// caps at 100 API calls.
	maxCalls := fetchBoardsMax / pageSize
	if calls > maxCalls {
		t.Fatalf("exceeded iteration ceiling: %d calls (cap is %d)", calls, maxCalls)
	}
}

// Short first page (fewer than maxResults) is the alternate termination path.
// Covered via the `isLast` flag above; this variant explicitly tests the other
// branch of `if resp.IsLast || len(resp.Values) == 0 { break }`.
func TestFetchBoards_StopsOnEmptyPage(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		// Return an empty page with isLast=false — fetcher should still terminate.
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{IsLast: false, Values: nil})
	}))
	defer server.Close()

	client := newTestClient(t, server)

	count, err := fetchBoards(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, count, 0)
	testutil.Equal(t, calls, 1)
}

func TestFetchSprints_MissingBoardsCache(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no API calls should be made when boards cache is absent")
	}))
	defer server.Close()

	client := newTestClient(t, server)

	_, err := fetchSprints(context.Background(), client)
	testutil.Error(t, err)
	if !strings.Contains(err.Error(), "refresh boards first") {
		t.Fatalf("expected error to mention 'refresh boards first', got: %v", err)
	}
	if _, err := ReadResource[map[int][]api.Sprint]("sprints"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected no sprints envelope on disk, got err=%v", err)
	}
}

func TestFetchSprints_MultiBoard(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	calls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		switch r.URL.Path {
		case "/rest/agile/1.0/board/23/sprint":
			_ = json.NewEncoder(w).Encode(api.SprintsResponse{
				IsLast: true,
				Values: []api.Sprint{
					{ID: 125, Name: "MON Sprint 70", State: "active"},
					{ID: 124, Name: "MON Sprint 69", State: "closed"},
				},
			})
		case "/rest/agile/1.0/board/24/sprint":
			_ = json.NewEncoder(w).Encode(api.SprintsResponse{
				IsLast: true,
				Values: []api.Sprint{{ID: 200, Name: "ON Sprint 1", State: "active"}},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)

	testutil.RequireNoError(t, WriteResource("boards", "24h", []api.Board{
		{ID: 23, Name: "MON board", Type: "scrum"},
		{ID: 24, Name: "ON board", Type: "kanban"},
	}))

	count, err := fetchSprints(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, count, 3)
	testutil.Equal(t, calls["/rest/agile/1.0/board/23/sprint"], 1)
	testutil.Equal(t, calls["/rest/agile/1.0/board/24/sprint"], 1)

	env, err := ReadResource[map[int][]api.Sprint]("sprints")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(env.Data[23]), 2)
	testutil.Equal(t, len(env.Data[24]), 1)
	testutil.Equal(t, env.Data[23][0].Name, "MON Sprint 70")
	testutil.Equal(t, env.Data[24][0].Name, "ON Sprint 1")
}

func TestFetchSprints_Pagination(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/agile/1.0/board/23/sprint")
		startAt, _ := strconv.Atoi(r.URL.Query().Get("startAt"))
		maxResults, _ := strconv.Atoi(r.URL.Query().Get("maxResults"))
		testutil.Equal(t, maxResults, 50)
		calls++

		var values []api.Sprint
		var isLast bool
		switch startAt {
		case 0:
			for i := 0; i < 50; i++ {
				values = append(values, api.Sprint{ID: i + 1, Name: "s" + strconv.Itoa(i+1)})
			}
		case 50:
			for i := 50; i < 70; i++ {
				values = append(values, api.Sprint{ID: i + 1, Name: "s" + strconv.Itoa(i+1)})
			}
			isLast = true
		default:
			t.Errorf("unexpected startAt: %d", startAt)
		}
		_ = json.NewEncoder(w).Encode(api.SprintsResponse{
			StartAt:    startAt,
			MaxResults: maxResults,
			IsLast:     isLast,
			Values:     values,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)

	testutil.RequireNoError(t, WriteResource("boards", "24h", []api.Board{{ID: 23}}))

	count, err := fetchSprints(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, count, 70)
	testutil.Equal(t, calls, 2)
}

// Partial-success: one board fails, others succeed. The fetch writes a
// partial envelope (non-empty map) and returns no error so one stale
// board doesn't poison the whole sprints cache.
func TestFetchSprints_PerBoardErrorSkipped(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	// Capture the warning writer so we can assert the per-board error surface
	// (not just the count). This also exercises SetWarnWriter, documenting
	// the warning-text contract.
	var warnBuf bytes.Buffer
	defer SetWarnWriter(&warnBuf)()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/agile/1.0/board/1/sprint":
			_ = json.NewEncoder(w).Encode(api.SprintsResponse{IsLast: true, Values: []api.Sprint{{ID: 10, Name: "Only"}}})
		case "/rest/agile/1.0/board/2/sprint":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errorMessages":["boom"]}`))
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)

	testutil.RequireNoError(t, WriteResource("boards", "24h", []api.Board{{ID: 1}, {ID: 2}}))

	count, err := fetchSprints(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, count, 1)

	env, err := ReadResource[map[int][]api.Sprint]("sprints")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(env.Data[1]), 1)
	if _, present := env.Data[2]; present {
		t.Fatalf("expected board 2 to be skipped after failure, got entry: %v", env.Data[2])
	}

	warnText := warnBuf.String()
	if !strings.Contains(warnText, "board 2") || !strings.Contains(warnText, "skipping") {
		t.Errorf("expected per-board skip warning for board 2, got: %q", warnText)
	}
}

// When every board errors, the fetch fails overall — there's nothing
// useful to cache and callers should see a clear error.
func TestFetchSprints_AllBoardsFailHardError(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClient(t, server)

	testutil.RequireNoError(t, WriteResource("boards", "24h", []api.Board{{ID: 1}, {ID: 2}}))

	_, err := fetchSprints(context.Background(), client)
	testutil.Error(t, err)
	if !strings.Contains(err.Error(), "all boards failed") {
		t.Fatalf("expected 'all boards failed' in error, got: %v", err)
	}
	if _, err := ReadResource[map[int][]api.Sprint]("sprints"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected no sprints envelope after total failure, got err=%v", err)
	}
}
