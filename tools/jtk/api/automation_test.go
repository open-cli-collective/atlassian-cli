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
			Links: automationLinks{},
			Data: []AutomationRuleSummary{
				{UUID: "uuid-1", Name: "Rule One", State: "ENABLED"},
				{UUID: "uuid-2", Name: "Rule Two", State: "DISABLED"},
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
			Links: automationLinks{},
			Data: []AutomationRuleSummary{
				{UUID: "uuid-1", Name: "Enabled Rule", State: "ENABLED"},
				{UUID: "uuid-2", Name: "Disabled Rule", State: "DISABLED"},
				{UUID: "uuid-3", Name: "Another Enabled", State: "ENABLED"},
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
				Links: automationLinks{},
				Data: []AutomationRuleSummary{
					{UUID: "uuid-1", Name: "Enabled Rule", State: "ENABLED"},
					{UUID: "uuid-2", Name: "Disabled Rule", State: "DISABLED"},
					{UUID: "uuid-3", Name: "Another Enabled", State: "ENABLED"},
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
			UUID:  "uuid-42",
			Name:  "My Automation Rule",
			State: "ENABLED",
			Trigger: &RuleComponent{
				Component: "TRIGGER",
				Type:      "jira.issue.create",
			},
			Components: []RuleComponent{
				{Component: "CONDITION", Type: "jira.issue.condition"},
				{Component: "ACTION", Type: "jira.issue.assign"},
			},
		}
		resp := struct {
			Rule        AutomationRule    `json:"rule"`
			Connections []json.RawMessage `json:"connections,omitempty"`
		}{Rule: rule}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	rule, err := client.GetAutomationRule("42")
	require.NoError(t, err)
	assert.Equal(t, "My Automation Rule", rule.Name)
	assert.Equal(t, "ENABLED", rule.State)
	require.NotNil(t, rule.Trigger)
	assert.Equal(t, "TRIGGER", rule.Trigger.Component)
	assert.Len(t, rule.Components, 2)
	assert.Equal(t, "CONDITION", rule.Components[0].Component)
	assert.Equal(t, "ACTION", rule.Components[1].Component)
}

func TestGetAutomationRuleRaw(t *testing.T) {
	expectedJSON := `{"rule":{"uuid":"uuid-raw-42","name":"Raw Rule","state":"ENABLED"},"connections":[]}`

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

func TestCreateAutomationRule(t *testing.T) {
	var receivedBody json.RawMessage
	var receivedMethod string

	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
			return
		}

		receivedMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":99,"ruleKey":"new-uuid-123","name":"New Rule"}`))
	}))
	defer server.Close()

	ruleJSON := json.RawMessage(`{"name":"New Rule","state":"DISABLED"}`)
	resp, err := client.CreateAutomationRule(ruleJSON)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, receivedMethod)
	assert.JSONEq(t, `{"name":"New Rule","state":"DISABLED"}`, string(receivedBody))

	var created struct {
		ID      json.Number `json:"id"`
		RuleKey string      `json:"ruleKey"`
		Name    string      `json:"name"`
	}
	require.NoError(t, json.Unmarshal(resp, &created))
	assert.Equal(t, "99", created.ID.String())
	assert.Equal(t, "new-uuid-123", created.RuleKey)
	assert.Equal(t, "New Rule", created.Name)
}

func TestAutomationRuleIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		rule     AutomationRule
		expected string
	}{
		{
			name:     "prefers UUID",
			rule:     AutomationRule{UUID: "uuid-1", RuleKey: "rk-1", ID: json.Number("42")},
			expected: "uuid-1",
		},
		{
			name:     "falls back to RuleKey",
			rule:     AutomationRule{RuleKey: "rk-1", ID: json.Number("42")},
			expected: "rk-1",
		},
		{
			name:     "falls back to numeric ID",
			rule:     AutomationRule{ID: json.Number("42")},
			expected: "42",
		},
		{
			name:     "empty when all fields absent",
			rule:     AutomationRule{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.rule.Identifier())
		})
	}
}

func TestAutomationRuleSummaryIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		summary  AutomationRuleSummary
		expected string
	}{
		{
			name:     "prefers UUID",
			summary:  AutomationRuleSummary{UUID: "uuid-1", RuleKey: "rk-1", ID: json.Number("42")},
			expected: "uuid-1",
		},
		{
			name:     "falls back to RuleKey",
			summary:  AutomationRuleSummary{RuleKey: "rk-1", ID: json.Number("42")},
			expected: "rk-1",
		},
		{
			name:     "falls back to numeric ID",
			summary:  AutomationRuleSummary{ID: json.Number("42")},
			expected: "42",
		},
		{
			name:     "empty when all fields absent",
			summary:  AutomationRuleSummary{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.summary.Identifier())
		})
	}
}

func TestItemsLegacyFallback(t *testing.T) {
	t.Run("returns Data when present", func(t *testing.T) {
		resp := AutomationRuleSummaryResponse{
			Data:   []AutomationRuleSummary{{UUID: "d-1"}},
			Values: []AutomationRuleSummary{{UUID: "v-1"}},
		}
		items := resp.Items()
		require.Len(t, items, 1)
		assert.Equal(t, "d-1", items[0].UUID)
	})

	t.Run("falls back to Values when Data is empty", func(t *testing.T) {
		resp := AutomationRuleSummaryResponse{
			Values: []AutomationRuleSummary{
				{ID: json.Number("1"), Name: "Legacy Rule"},
				{ID: json.Number("2"), Name: "Legacy Rule 2"},
			},
		}
		items := resp.Items()
		require.Len(t, items, 2)
		assert.Equal(t, "Legacy Rule", items[0].Name)
	})
}

func TestNextURLLegacyFallback(t *testing.T) {
	t.Run("returns Links.Next when present", func(t *testing.T) {
		next := "http://example.com/next"
		resp := AutomationRuleSummaryResponse{
			Links: automationLinks{Next: &next},
			Next:  "http://example.com/legacy-next",
		}
		assert.Equal(t, "http://example.com/next", resp.NextURL())
	})

	t.Run("falls back to top-level Next", func(t *testing.T) {
		resp := AutomationRuleSummaryResponse{
			Next: "http://example.com/legacy-next",
		}
		assert.Equal(t, "http://example.com/legacy-next", resp.NextURL())
	})

	t.Run("returns empty when no next URL", func(t *testing.T) {
		resp := AutomationRuleSummaryResponse{}
		assert.Equal(t, "", resp.NextURL())
	})
}

func TestGetAutomationRuleLegacyFallback(t *testing.T) {
	t.Run("parses top-level rule without envelope", func(t *testing.T) {
		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":42,"name":"Legacy Rule","state":"ENABLED"}`))
		}))
		defer server.Close()

		rule, err := client.GetAutomationRule("42")
		require.NoError(t, err)
		assert.Equal(t, "Legacy Rule", rule.Name)
		assert.Equal(t, "ENABLED", rule.State)
	})

	t.Run("normalizes RuleKey to UUID in legacy shape", func(t *testing.T) {
		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ruleKey":"rk-99","name":"RuleKey Rule","state":"DISABLED"}`))
		}))
		defer server.Close()

		rule, err := client.GetAutomationRule("rk-99")
		require.NoError(t, err)
		assert.Equal(t, "rk-99", rule.UUID)
		assert.Equal(t, "rk-99", rule.RuleKey)
	})

	t.Run("normalizes RuleKey to UUID in envelope shape", func(t *testing.T) {
		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"rule":{"ruleKey":"rk-envelope","name":"Envelope RuleKey","state":"ENABLED"}}`))
		}))
		defer server.Close()

		rule, err := client.GetAutomationRule("rk-envelope")
		require.NoError(t, err)
		assert.Equal(t, "rk-envelope", rule.UUID)
		assert.Equal(t, "Envelope RuleKey", rule.Name)
	})
}

func TestListAutomationRulesLegacyShape(t *testing.T) {
	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total":2,"values":[{"id":1,"name":"Old Rule 1","state":"ENABLED"},{"id":2,"name":"Old Rule 2","state":"DISABLED"}]}`))
	}))
	defer server.Close()

	rules, err := client.ListAutomationRules()
	require.NoError(t, err)
	assert.Len(t, rules, 2)
	assert.Equal(t, "Old Rule 1", rules[0].Name)
	assert.Equal(t, "Old Rule 2", rules[1].Name)
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
			next := "http://" + r.Host + r.URL.Path + "?cursor=abc"
			resp := AutomationRuleSummaryResponse{
				Links: automationLinks{Next: &next},
				Data:  []AutomationRuleSummary{{UUID: "uuid-1", Name: "Rule 1", State: "ENABLED"}},
			}
			_ = json.NewEncoder(w).Encode(resp)
		} else {
			resp := AutomationRuleSummaryResponse{
				Links: automationLinks{},
				Data: []AutomationRuleSummary{
					{UUID: "uuid-2", Name: "Rule 2", State: "ENABLED"},
					{UUID: "uuid-3", Name: "Rule 3", State: "DISABLED"},
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
