package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

// rankCall is one PUT /issue/rank body the stub server received.
type rankCall struct {
	Issues []string `json:"issues"`
	Before string   `json:"rankBeforeIssue"`
	After  string   `json:"rankAfterIssue"`
}

func rankStub(t *testing.T, calls *[]rankCall, respond func(w http.ResponseWriter, c rankCall)) http.HandlerFunc {
	t.Helper()
	var mu sync.Mutex
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/agile/1.0/issue/rank" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var c rankCall
		testutil.RequireNoError(t, json.Unmarshal(body, &c))
		mu.Lock()
		*calls = append(*calls, c)
		mu.Unlock()
		respond(w, c)
	}
}

func noContent(w http.ResponseWriter, _ rankCall) { w.WriteHeader(http.StatusNoContent) }

func keys(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s-%d", prefix, i+1)
	}
	return out
}

func TestRankIssues_Before(t *testing.T) {
	t.Parallel()
	var calls []rankCall
	client, server := newTestClientWithServer(t, rankStub(t, &calls, noContent))
	defer server.Close()

	failures, err := client.RankIssues(context.Background(), []string{"PROJ-2", "PROJ-3"}, "PROJ-1", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(failures), 0)
	testutil.Equal(t, len(calls), 1)
	testutil.Equal(t, calls[0].Before, "PROJ-1")
	testutil.Equal(t, calls[0].After, "")
	testutil.Equal(t, fmt.Sprint(calls[0].Issues), "[PROJ-2 PROJ-3]")
}

func TestRankIssues_After(t *testing.T) {
	t.Parallel()
	var calls []rankCall
	client, server := newTestClientWithServer(t, rankStub(t, &calls, noContent))
	defer server.Close()

	_, err := client.RankIssues(context.Background(), []string{"PROJ-2"}, "", "PROJ-9")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(calls), 1)
	testutil.Equal(t, calls[0].After, "PROJ-9")
	testutil.Equal(t, calls[0].Before, "")
}

// A 207 Multi-Status lists the issues Jira refused; they come back as
// RankFailures and the issues it ranked are not reported.
func TestRankIssues_MultiStatusReportsTheRefusedIssues(t *testing.T) {
	t.Parallel()
	var calls []rankCall
	client, server := newTestClientWithServer(t, rankStub(t, &calls, func(w http.ResponseWriter, _ rankCall) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`{"entries":[
			{"issueId":10001,"issueKey":"PROJ-2","status":200},
			{"issueId":10002,"issueKey":"PROJ-3","status":400,"errors":["Issue is not on the board"]}]}`))
	}))
	defer server.Close()

	failures, err := client.RankIssues(context.Background(), []string{"PROJ-2", "PROJ-3"}, "PROJ-1", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(failures), 1)
	testutil.Equal(t, failures[0].Key, "PROJ-3")
	testutil.Equal(t, failures[0].Status, 400)
	testutil.Equal(t, fmt.Sprint(failures[0].Errors), "[Issue is not on the board]")
}

// More than MaxRankIssues go in chunks that keep the given order: every
// "before" chunk lands immediately before the anchor (so later chunks sit
// between the earlier ones and the anchor), and each "after" chunk anchors on
// the previous chunk's last issue.
func TestRankIssues_ChunksKeepTheGivenOrder(t *testing.T) {
	t.Parallel()
	n := MaxRankIssues + 3
	in := keys("PROJ", n)

	var calls []rankCall
	client, server := newTestClientWithServer(t, rankStub(t, &calls, noContent))
	defer server.Close()
	_, err := client.RankIssues(context.Background(), in, "ANCHOR-1", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(calls), 2)
	testutil.Equal(t, len(calls[0].Issues), MaxRankIssues)
	testutil.Equal(t, len(calls[1].Issues), 3)
	testutil.Equal(t, calls[1].Issues[0], in[MaxRankIssues])
	testutil.Equal(t, calls[0].Before, "ANCHOR-1")
	testutil.Equal(t, calls[1].Before, "ANCHOR-1")

	var afterCalls []rankCall
	client2, server2 := newTestClientWithServer(t, rankStub(t, &afterCalls, noContent))
	defer server2.Close()
	_, err = client2.RankIssues(context.Background(), in, "", "ANCHOR-1")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(afterCalls), 2)
	testutil.Equal(t, afterCalls[0].After, "ANCHOR-1")
	testutil.Equal(t, afterCalls[1].After, in[MaxRankIssues-1])
}

func TestRankIssues_NeedsExactlyOneAnchor(t *testing.T) {
	t.Parallel()
	var calls []rankCall
	client, server := newTestClientWithServer(t, rankStub(t, &calls, noContent))
	defer server.Close()

	_, err := client.RankIssues(context.Background(), []string{"PROJ-2"}, "", "")
	testutil.Error(t, err)
	_, err = client.RankIssues(context.Background(), []string{"PROJ-2"}, "PROJ-1", "PROJ-3")
	testutil.Error(t, err)
	testutil.Equal(t, len(calls), 0)
}

func TestRankIssues_APIErrorStops(t *testing.T) {
	t.Parallel()
	var calls []rankCall
	client, server := newTestClientWithServer(t, rankStub(t, &calls, func(w http.ResponseWriter, _ rankCall) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errorMessages":["Rank field is not configured"]}`))
	}))
	defer server.Close()

	_, err := client.RankIssues(context.Background(), keys("PROJ", MaxRankIssues+1), "PROJ-99", "")
	testutil.Error(t, err)
	testutil.Equal(t, len(calls), 1)
}
