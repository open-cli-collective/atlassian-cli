package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// MaxRankIssues is the most issues the Agile rank endpoint takes in one call.
const MaxRankIssues = 50

// RankFailure is one issue the rank endpoint refused (an entry of a 207
// Multi-Status response).
type RankFailure struct {
	Key    string
	Status int
	Errors []string
}

type rankResponse struct {
	Entries []struct {
		IssueKey string   `json:"issueKey"`
		Status   int      `json:"status"`
		Errors   []string `json:"errors"`
	} `json:"entries"`
}

// RankIssues moves issueKeys, in the order given, to sit immediately before
// (before != "") or after (after != "") the anchor issue in Jira's rank - the
// order boards and backlogs show. Exactly one of before and after must be set.
//
// The endpoint (PUT /rest/agile/1.0/issue/rank) takes at most MaxRankIssues
// issues per call, so longer lists go in chunks that keep the given order:
// every "before" chunk is placed immediately before the anchor, so later
// chunks land between the earlier ones and the anchor; each "after" chunk
// anchors on the previous chunk's last issue. Issues Jira refuses in a 207
// response come back as RankFailures; a transport or API error stops at the
// failing chunk and is returned with the failures so far.
func (c *Client) RankIssues(ctx context.Context, issueKeys []string, before, after string) ([]RankFailure, error) {
	if (before == "") == (after == "") {
		return nil, errors.New("ranking issues: exactly one of before and after is required")
	}
	urlStr := c.AgileURL + "/issue/rank"
	anchorAfter := after
	var failures []RankFailure
	for start := 0; start < len(issueKeys); start += MaxRankIssues {
		chunk := issueKeys[start:min(start+MaxRankIssues, len(issueKeys))]
		req := map[string]any{"issues": chunk}
		if before != "" {
			req["rankBeforeIssue"] = before
		} else {
			req["rankAfterIssue"] = anchorAfter
		}
		body, err := c.Put(ctx, urlStr, req)
		if err != nil {
			return failures, fmt.Errorf("ranking issues: %w", err)
		}
		refused, err := parseRankFailures(body)
		if err != nil {
			return failures, err
		}
		failures = append(failures, refused...)
		anchorAfter = chunk[len(chunk)-1]
	}
	return failures, nil
}

// parseRankFailures reads a 207 body's refused entries. A 204 has no body.
func parseRankFailures(body []byte) ([]RankFailure, error) {
	if len(body) == 0 {
		return nil, nil
	}
	var resp rankResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("ranking issues: parsing the response: %w", err)
	}
	var failures []RankFailure
	for _, e := range resp.Entries {
		if e.Status == http.StatusOK && len(e.Errors) == 0 {
			continue
		}
		failures = append(failures, RankFailure{Key: e.IssueKey, Status: e.Status, Errors: e.Errors})
	}
	return failures, nil
}
