package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Development is the issue-centric data shown in Jira's Development panel.
// Jira exposes this through private APIs, so the response types intentionally
// contain only the fields jtk presents.
type Development struct {
	IssueKey         string
	PullRequestState string
	Commits          int
	Builds           int
	SuccessfulBuilds int
	Providers        []string
	PullRequests     []DevelopmentPullRequest
	// PullRequestsTruncated is set when pull requests came from the dev-status
	// fallback and it returned its server-side maximum, so more may exist.
	PullRequestsTruncated bool
}

// DevelopmentPullRequest is a pull request associated with a Jira issue.
type DevelopmentPullRequest struct {
	ID         string
	Title      string
	URL        string
	Status     string
	LastUpdate string
	Repository string
	Provider   string
}

// devStatusPullRequestLimit is the most pull requests the dev-status detail
// endpoint returns for one provider. It ignores paging parameters.
const devStatusPullRequestLimit = 100

// developmentGraphPageSize and developmentGraphMaxPages bound the Graph Store
// walk; the page cap only guards against a cursor that never terminates.
const (
	developmentGraphPageSize = 50
	developmentGraphMaxPages = 200
)

const developmentGraphQuery = `query JtkIssueAssociatedPullRequests($id: ID!, $first: Int, $after: String) {
  graphStore_issueAssociatedPr(id: $id, first: $first, after: $after) @optIn(to: "GraphStoreIssueAssociatedPr") {
    pageInfo { hasNextPage endCursor }
    edges {
      node {
        __typename
        ... on DevOpsPullRequestDetails { url title status repositoryName providerName lastUpdated }
      }
    }
  }
}`

type developmentSummaryResponse struct {
	Summary struct {
		PullRequest developmentSummaryItem `json:"pullrequest"`
		Repository  developmentSummaryItem `json:"repository"`
		Build       developmentSummaryItem `json:"build"`
	} `json:"summary"`
	Errors       []json.RawMessage `json:"errors"`
	ConfigErrors []json.RawMessage `json:"configErrors"`
}

type developmentSummaryItem struct {
	Overall struct {
		Count                int    `json:"count"`
		State                string `json:"state"`
		SuccessfulBuildCount int    `json:"successfulBuildCount"`
	} `json:"overall"`
	ByInstanceType map[string]struct {
		Name string `json:"name"`
	} `json:"byInstanceType"`
}

type developmentDetailResponse struct {
	Detail []struct {
		PullRequests []struct {
			ID             string `json:"id"`
			Name           string `json:"name"`
			URL            string `json:"url"`
			Status         string `json:"status"`
			LastUpdate     string `json:"lastUpdate"`
			RepositoryName string `json:"repositoryName"`
		} `json:"pullRequests"`
		Repositories []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"repositories"`
		Instance struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"_instance"`
	} `json:"detail"`
	Errors []json.RawMessage `json:"errors"`
}

type developmentGraphResponse struct {
	Data struct {
		Connection *struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Edges []struct {
				Node *struct {
					TypeName       string `json:"__typename"`
					URL            string `json:"url"`
					Title          string `json:"title"`
					Status         string `json:"status"`
					RepositoryName string `json:"repositoryName"`
					ProviderName   string `json:"providerName"`
					LastUpdated    string `json:"lastUpdated"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"graphStore_issueAssociatedPr"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// GetDevelopment returns the pull requests and summary counts shown in an
// issue's Jira Development panel. Pull requests come from the Graph Store
// issue-to-PR relationship, which pages past the 100-item cap of
// /rest/dev-status; dev-status is the fallback. Both APIs are private and may
// change without notice.
func (c *Client) GetDevelopment(ctx context.Context, issueKey string) (*Development, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}
	if c.IsBearerAuth() {
		return nil, ErrDevelopmentUnavailable
	}

	issue, err := c.GetIssue(ctx, issueKey)
	if err != nil {
		return nil, err
	}
	if issue.ID == "" {
		return nil, fmt.Errorf("issue %s returned an empty numeric ID", issueKey)
	}

	summaryURL := buildURL(c.URL+"/rest/dev-status/1.0/issue/summary", map[string]string{"issueId": issue.ID})
	body, err := c.Get(ctx, summaryURL)
	if err != nil {
		return nil, fmt.Errorf("fetching development summary: %w", err)
	}

	var summary developmentSummaryResponse
	if err := json.Unmarshal(body, &summary); err != nil {
		return nil, fmt.Errorf("parsing development summary: %w", err)
	}
	if len(summary.Errors) > 0 || len(summary.ConfigErrors) > 0 {
		return nil, fmt.Errorf("development summary returned provider errors")
	}

	result := &Development{
		IssueKey:         issue.Key,
		PullRequestState: summary.Summary.PullRequest.Overall.State,
		Commits:          summary.Summary.Repository.Overall.Count,
		Builds:           summary.Summary.Build.Overall.Count,
		SuccessfulBuilds: summary.Summary.Build.Overall.SuccessfulBuildCount,
	}

	providerTypes := make([]string, 0, len(summary.Summary.PullRequest.ByInstanceType))
	providerNames := make(map[string]string, len(summary.Summary.PullRequest.ByInstanceType))
	for providerType, instance := range summary.Summary.PullRequest.ByInstanceType {
		providerTypes = append(providerTypes, providerType)
		providerNames[providerType] = instance.Name
	}
	sort.Strings(providerTypes)

	if summary.Summary.PullRequest.Overall.Count == 0 && len(providerTypes) == 0 {
		return result, nil
	}

	pullRequests := newDevelopmentPullRequestSet()
	if err := c.collectGraphPullRequests(ctx, issue.ID, providerNames, pullRequests); err != nil {
		pullRequests = newDevelopmentPullRequestSet()
		truncated, err := c.collectDevStatusPullRequests(ctx, issue.ID, providerTypes, providerNames, pullRequests)
		if err != nil {
			return nil, err
		}
		result.PullRequestsTruncated = truncated
	}

	if summary.Summary.PullRequest.Overall.Count > 0 && len(pullRequests.byKey) == 0 {
		return nil, fmt.Errorf("development summary reports pull requests but detail returned none")
	}

	result.Providers = pullRequests.sortedProviders()
	result.PullRequests = pullRequests.sorted()
	return result, nil
}

// collectGraphPullRequests walks the Graph Store issue-to-PR relationship to
// the end of its cursor.
func (c *Client) collectGraphPullRequests(ctx context.Context, issueID string, providerNames map[string]string, set *developmentPullRequestSet) error {
	cloudID, err := c.GetCloudID(ctx)
	if err != nil {
		return err
	}
	issueARI := fmt.Sprintf("ari:cloud:jira:%s:issue/%s", cloudID, issueID)
	headers := map[string]string{"X-Query-Context": "ari:cloud:platform::site/" + cloudID}

	defaultProvider := ""
	for _, name := range providerNames {
		set.addProvider(name)
		if len(providerNames) == 1 {
			defaultProvider = name
		}
	}

	var after any
	for range developmentGraphMaxPages {
		request := map[string]any{
			"operationName": "JtkIssueAssociatedPullRequests",
			"query":         developmentGraphQuery,
			"variables":     map[string]any{"id": issueARI, "first": developmentGraphPageSize, "after": after},
		}
		body, err := c.DoWithHeaders(ctx, http.MethodPost, c.URL+"/gateway/api/graphql", request, headers)
		if err != nil {
			return fmt.Errorf("fetching development pull requests from graph store: %w", err)
		}

		var response developmentGraphResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("parsing development pull requests from graph store: %w", err)
		}
		if len(response.Errors) > 0 {
			return fmt.Errorf("graph store returned errors: %s", response.Errors[0].Message)
		}
		connection := response.Data.Connection
		if connection == nil {
			return fmt.Errorf("graph store returned no pull request connection")
		}

		for _, edge := range connection.Edges {
			node := edge.Node
			if node == nil || node.TypeName != "DevOpsPullRequestDetails" {
				continue
			}
			normalizedURL := normalizeDevelopmentURL(node.URL)
			provider := node.ProviderName
			if provider == "" {
				provider = defaultProvider
			}
			set.add(DevelopmentPullRequest{
				ID:         developmentPullRequestNumberFromURL(normalizedURL),
				Title:      node.Title,
				URL:        normalizedURL,
				Status:     node.Status,
				LastUpdate: node.LastUpdated,
				Repository: firstNonEmpty(node.RepositoryName, developmentRepositoryFromURL(normalizedURL)),
				Provider:   provider,
			}, normalizedURL)
		}

		if !connection.PageInfo.HasNextPage {
			return nil
		}
		if connection.PageInfo.EndCursor == "" || connection.PageInfo.EndCursor == after {
			return fmt.Errorf("graph store returned a page without a new cursor")
		}
		after = connection.PageInfo.EndCursor
	}
	return fmt.Errorf("graph store returned more than %d pages of pull requests", developmentGraphMaxPages)
}

// collectDevStatusPullRequests reads /rest/dev-status detail per provider. It
// reports truncated when any provider returned the endpoint's maximum.
func (c *Client) collectDevStatusPullRequests(ctx context.Context, issueID string, providerTypes []string, providerNames map[string]string, set *developmentPullRequestSet) (bool, error) {
	truncated := false
	for _, providerType := range providerTypes {
		detailURL := buildURL(c.URL+"/rest/dev-status/1.0/issue/detail", map[string]string{
			"issueId":         issueID,
			"applicationType": providerType,
			"dataType":        "pullrequest",
		})
		body, err := c.Get(ctx, detailURL)
		if err != nil {
			return false, fmt.Errorf("fetching development pull requests for %s: %w", providerType, err)
		}

		var response developmentDetailResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return false, fmt.Errorf("parsing development pull requests for %s: %w", providerType, err)
		}
		if len(response.Errors) > 0 {
			return false, fmt.Errorf("development pull requests for %s returned provider errors", providerType)
		}

		providerCount := 0
		for _, detail := range response.Detail {
			providerName := providerNames[providerType]
			if detail.Instance.Name != "" {
				providerName = detail.Instance.Name
			}
			set.addProvider(providerName)

			repositoryID, detailRepositoryName := "", ""
			if len(detail.Repositories) > 0 {
				repositoryID = detail.Repositories[0].ID
				detailRepositoryName = detail.Repositories[0].Name
			}

			providerCount += len(detail.PullRequests)
			for _, raw := range detail.PullRequests {
				normalizedURL := normalizeDevelopmentURL(raw.URL)
				key := normalizedURL
				if key == "" {
					key = strings.Join([]string{providerType, repositoryID, raw.ID}, "\x00")
				}
				set.add(DevelopmentPullRequest{
					ID:         raw.ID,
					Title:      raw.Name,
					URL:        normalizedURL,
					Status:     raw.Status,
					LastUpdate: raw.LastUpdate,
					Repository: firstNonEmpty(raw.RepositoryName, developmentRepositoryFromURL(normalizedURL), detailRepositoryName),
					Provider:   providerName,
				}, key)
			}
		}
		if providerCount >= devStatusPullRequestLimit {
			truncated = true
		}
	}
	return truncated, nil
}

// developmentPullRequestSet deduplicates pull requests by key, keeping the
// most recently updated copy.
type developmentPullRequestSet struct {
	byKey     map[string]DevelopmentPullRequest
	providers map[string]struct{}
}

func newDevelopmentPullRequestSet() *developmentPullRequestSet {
	return &developmentPullRequestSet{
		byKey:     make(map[string]DevelopmentPullRequest),
		providers: make(map[string]struct{}),
	}
}

func (s *developmentPullRequestSet) add(candidate DevelopmentPullRequest, key string) {
	s.addProvider(candidate.Provider)
	current, ok := s.byKey[key]
	if !ok || developmentTime(candidate.LastUpdate).After(developmentTime(current.LastUpdate)) {
		s.byKey[key] = candidate
	}
}

func (s *developmentPullRequestSet) addProvider(name string) {
	if name != "" {
		s.providers[name] = struct{}{}
	}
}

func (s *developmentPullRequestSet) sortedProviders() []string {
	providers := make([]string, 0, len(s.providers))
	for provider := range s.providers {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return providers
}

func (s *developmentPullRequestSet) sorted() []DevelopmentPullRequest {
	pullRequests := make([]DevelopmentPullRequest, 0, len(s.byKey))
	for _, pr := range s.byKey {
		pullRequests = append(pullRequests, pr)
	}
	sort.Slice(pullRequests, func(i, j int) bool {
		left, right := pullRequests[i], pullRequests[j]
		leftTime, rightTime := developmentTime(left.LastUpdate), developmentTime(right.LastUpdate)
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}
		return left.URL < right.URL
	})
	return pullRequests
}

func normalizeDevelopmentURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return strings.TrimRight(raw, "/")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String()
}

func developmentRepositoryFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Hostname(), "github.com") {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "/" + parts[1]
}

// developmentPullRequestNumberFromURL returns the trailing number of a pull or
// merge request URL such as https://github.com/owner/repo/pull/42.
func developmentPullRequestNumberFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	switch parts[len(parts)-2] {
	case "pull", "pulls", "pull-requests", "merge_requests":
		return parts[len(parts)-1]
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func developmentTime(raw string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000-0700", "2006-01-02T15:04:05-0700"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
