package cache

import (
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
