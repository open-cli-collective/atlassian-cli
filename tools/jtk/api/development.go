package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Development is the issue-centric data shown in Jira's Development panel.
// Jira exposes this through a private API, so the response types intentionally
// contain only the fields jtk presents.
type Development struct {
	IssueKey         string
	PullRequestState string
	Commits          int
	Builds           int
	SuccessfulBuilds int
	Providers        []string
	PullRequests     []DevelopmentPullRequest
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
			ID         string `json:"id"`
			Name       string `json:"name"`
			URL        string `json:"url"`
			Status     string `json:"status"`
			LastUpdate string `json:"lastUpdate"`
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

// GetDevelopment returns the pull requests and summary counts shown in an
// issue's Jira Development panel. The /rest/dev-status API is private and may
// change without notice.
func (c *Client) GetDevelopment(ctx context.Context, issueKey string) (*Development, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
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

	type keyedPullRequest struct {
		pr DevelopmentPullRequest
	}
	byKey := make(map[string]keyedPullRequest)

	for _, providerType := range providerTypes {
		detailURL := buildURL(c.URL+"/rest/dev-status/1.0/issue/detail", map[string]string{
			"issueId":         issue.ID,
			"applicationType": providerType,
			"dataType":        "pullrequest",
		})
		body, err := c.Get(ctx, detailURL)
		if err != nil {
			return nil, fmt.Errorf("fetching development pull requests for %s: %w", providerType, err)
		}

		var response developmentDetailResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("parsing development pull requests for %s: %w", providerType, err)
		}
		if len(response.Errors) > 0 {
			return nil, fmt.Errorf("development pull requests for %s returned provider errors", providerType)
		}

		for _, detail := range response.Detail {
			providerName := providerNames[providerType]
			if detail.Instance.Name != "" {
				providerName = detail.Instance.Name
			}
			if providerName != "" {
				result.Providers = append(result.Providers, providerName)
			}

			repositoryID, repositoryName := "", ""
			if len(detail.Repositories) > 0 {
				repositoryID = detail.Repositories[0].ID
				repositoryName = detail.Repositories[0].Name
			}

			for _, raw := range detail.PullRequests {
				normalizedURL := normalizeDevelopmentURL(raw.URL)
				if repositoryName == "" {
					repositoryName = developmentRepositoryFromURL(normalizedURL)
				}
				key := normalizedURL
				if key == "" {
					key = strings.Join([]string{providerType, repositoryID, raw.ID}, "\x00")
				}
				candidate := DevelopmentPullRequest{
					ID:         raw.ID,
					Title:      raw.Name,
					URL:        normalizedURL,
					Status:     raw.Status,
					LastUpdate: raw.LastUpdate,
					Repository: repositoryName,
					Provider:   providerName,
				}
				current, ok := byKey[key]
				if !ok || developmentTime(candidate.LastUpdate).After(developmentTime(current.pr.LastUpdate)) {
					byKey[key] = keyedPullRequest{pr: candidate}
				}
			}
		}
	}

	if summary.Summary.PullRequest.Overall.Count > 0 && len(byKey) == 0 {
		return nil, fmt.Errorf("development summary reports pull requests but detail returned none")
	}

	seenProviders := make(map[string]struct{}, len(result.Providers))
	providers := result.Providers[:0]
	for _, provider := range result.Providers {
		if _, ok := seenProviders[provider]; ok {
			continue
		}
		seenProviders[provider] = struct{}{}
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	result.Providers = providers

	result.PullRequests = make([]DevelopmentPullRequest, 0, len(byKey))
	for _, item := range byKey {
		result.PullRequests = append(result.PullRequests, item.pr)
	}
	sort.Slice(result.PullRequests, func(i, j int) bool {
		left, right := result.PullRequests[i], result.PullRequests[j]
		leftTime, rightTime := developmentTime(left.LastUpdate), developmentTime(right.LastUpdate)
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}
		return left.URL < right.URL
	})

	return result, nil
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

func developmentTime(raw string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000-0700", "2006-01-02T15:04:05-0700"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
