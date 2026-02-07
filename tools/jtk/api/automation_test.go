package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClientWithServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "user@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)
	return client, server
}

func TestGetCloudID(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"abc-123-def"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		cloudID, err := client.GetCloudID()
		require.NoError(t, err)
		assert.Equal(t, "abc-123-def", cloudID)

		// Second call should return cached value without hitting server
		cloudID2, err := client.GetCloudID()
		require.NoError(t, err)
		assert.Equal(t, "abc-123-def", cloudID2)
	})

	t.Run("empty cloud ID", func(t *testing.T) {
		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":""}`))
		}))
		defer server.Close()

		_, err := client.GetCloudID()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty cloud ID")
	})

	t.Run("server error", func(t *testing.T) {
		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"internal"}`))
		}))
		defer server.Close()

		_, err := client.GetCloudID()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch cloud ID")
	})
}

func TestAutomationBaseURL(t *testing.T) {
	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"my-cloud-id"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	baseURL, err := client.AutomationBaseURL()
	require.NoError(t, err)
	assert.Equal(t, server.URL+"/gateway/api/automation/public/jira/my-cloud-id/rest/v1", baseURL)
}

func TestListAutomationRules(t *testing.T) {
	callCount := 0
	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
			return
		}

		callCount++
		w.WriteHeader(http.StatusOK)
		resp := AutomationRuleSummaryResponse{
			Total: 2,
			Values: []AutomationRuleSummary{
				{ID: json.Number("1"), Name: "Rule One", State: "ENABLED"},
				{ID: json.Number("2"), Name: "Rule Two", State: "DISABLED"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	rules, err := client.ListAutomationRules()
	require.NoError(t, err)
	assert.Len(t, rules, 2)
	assert.Equal(t, "Rule One", rules[0].Name)
	assert.Equal(t, "ENABLED", rules[0].State)
	assert.Equal(t, "Rule Two", rules[1].Name)
	assert.Equal(t, "DISABLED", rules[1].State)
}

func TestListAutomationRulesFiltered(t *testing.T) {
	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		resp := AutomationRuleSummaryResponse{
			Total: 3,
			Values: []AutomationRuleSummary{
				{ID: json.Number("1"), Name: "Enabled Rule", State: "ENABLED"},
				{ID: json.Number("2"), Name: "Disabled Rule", State: "DISABLED"},
				{ID: json.Number("3"), Name: "Another Enabled", State: "ENABLED"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Run("filter ENABLED", func(t *testing.T) {
		rules, err := client.ListAutomationRulesFiltered("ENABLED")
		require.NoError(t, err)
		assert.Len(t, rules, 2)
		for _, r := range rules {
			assert.Equal(t, "ENABLED", r.State)
		}
	})

	t.Run("filter DISABLED", func(t *testing.T) {
		// Need a fresh client to avoid cloud ID caching issues with sync.Once
		client2, server2 := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			resp := AutomationRuleSummaryResponse{
				Total: 3,
				Values: []AutomationRuleSummary{
					{ID: json.Number("1"), Name: "Enabled Rule", State: "ENABLED"},
					{ID: json.Number("2"), Name: "Disabled Rule", State: "DISABLED"},
					{ID: json.Number("3"), Name: "Another Enabled", State: "ENABLED"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server2.Close()

		rules, err := client2.ListAutomationRulesFiltered("DISABLED")
		require.NoError(t, err)
		assert.Len(t, rules, 1)
		assert.Equal(t, "Disabled Rule", rules[0].Name)
	})

	t.Run("no filter", func(t *testing.T) {
		rules, err := client.ListAutomationRulesFiltered("")
		require.NoError(t, err)
		assert.Len(t, rules, 3)
	})
}

func TestGetAutomationRule(t *testing.T) {
	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		rule := AutomationRule{
			ID:    json.Number("42"),
			Name:  "My Automation Rule",
			State: "ENABLED",
			Components: []RuleComponent{
				{Component: "TRIGGER", Type: "jira.issue.create"},
				{Component: "CONDITION", Type: "jira.issue.condition"},
				{Component: "ACTION", Type: "jira.issue.assign"},
			},
		}
		_ = json.NewEncoder(w).Encode(rule)
	}))
	defer server.Close()

	rule, err := client.GetAutomationRule("42")
	require.NoError(t, err)
	assert.Equal(t, "My Automation Rule", rule.Name)
	assert.Equal(t, "ENABLED", rule.State)
	assert.Len(t, rule.Components, 3)
	assert.Equal(t, "TRIGGER", rule.Components[0].Component)
	assert.Equal(t, "CONDITION", rule.Components[1].Component)
	assert.Equal(t, "ACTION", rule.Components[2].Component)
}

func TestGetAutomationRuleRaw(t *testing.T) {
	expectedJSON := `{"id":42,"name":"Raw Rule","state":"ENABLED"}`

	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedJSON))
	}))
	defer server.Close()

	raw, err := client.GetAutomationRuleRaw("42")
	require.NoError(t, err)
	assert.Equal(t, expectedJSON, string(raw))
}

func TestUpdateAutomationRule(t *testing.T) {
	var receivedBody json.RawMessage

	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
			return
		}

		if r.Method == http.MethodPut {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	ruleJSON := json.RawMessage(`{"name":"Updated Rule","state":"ENABLED"}`)
	err := client.UpdateAutomationRule("42", ruleJSON)
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"Updated Rule","state":"ENABLED"}`, string(receivedBody))
}

func TestSetAutomationRuleState(t *testing.T) {
	t.Run("enable", func(t *testing.T) {
		var receivedBody AutomationStateUpdate

		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
				return
			}

			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer server.Close()

		err := client.SetAutomationRuleState("42", true)
		require.NoError(t, err)
		assert.Equal(t, "ENABLED", receivedBody.RuleState)
	})

	t.Run("disable", func(t *testing.T) {
		var receivedBody AutomationStateUpdate

		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
				return
			}

			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer server.Close()

		err := client.SetAutomationRuleState("42", false)
		require.NoError(t, err)
		assert.Equal(t, "DISABLED", receivedBody.RuleState)
	})
}

func TestListAutomationRulesPagination(t *testing.T) {
	page := 0
	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
			return
		}

		page++
		w.WriteHeader(http.StatusOK)
		if page == 1 {
			resp := AutomationRuleSummaryResponse{
				Total:  3,
				Values: []AutomationRuleSummary{{ID: json.Number("1"), Name: "Rule 1", State: "ENABLED"}},
				Next:   "http://" + r.Host + r.URL.Path + "?cursor=abc",
			}
			_ = json.NewEncoder(w).Encode(resp)
		} else {
			resp := AutomationRuleSummaryResponse{
				Total: 3,
				Values: []AutomationRuleSummary{
					{ID: json.Number("2"), Name: "Rule 2", State: "ENABLED"},
					{ID: json.Number("3"), Name: "Rule 3", State: "DISABLED"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	rules, err := client.ListAutomationRules()
	require.NoError(t, err)
	assert.Len(t, rules, 3)
	assert.Equal(t, "Rule 1", rules[0].Name)
	assert.Equal(t, "Rule 2", rules[1].Name)
	assert.Equal(t, "Rule 3", rules[2].Name)
	assert.Equal(t, 2, page) // Verify two pages were fetched
}
