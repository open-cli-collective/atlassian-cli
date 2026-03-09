package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

// jsonEq is defined in types_test.go (same package)

func newTestClientWithServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "user@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)
	return client, server
}

func TestGetCloudID(t *testing.T) {
	t.Parallel()
	t.Run("successful fetch", func(t *testing.T) {
		t.Parallel()
		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"abc-123-def"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		cloudID, err := client.GetCloudID(context.Background())
		testutil.RequireNoError(t, err)
		testutil.Equal(t, cloudID, "abc-123-def")

		// Second call should return cached value without hitting server
		cloudID2, err := client.GetCloudID(context.Background())
		testutil.RequireNoError(t, err)
		testutil.Equal(t, cloudID2, "abc-123-def")
	})

	t.Run("empty cloud ID", func(t *testing.T) {
		t.Parallel()
		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":""}`))
		}))
		defer server.Close()

		_, err := client.GetCloudID(context.Background())
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "empty cloud ID")
	})

	t.Run("server error", func(t *testing.T) {
		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"internal"}`))
		}))
		defer server.Close()

		_, err := client.GetCloudID(context.Background())
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "fetching cloud ID")
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

	baseURL, err := client.AutomationBaseURL(context.Background())
	testutil.RequireNoError(t, err)
	testutil.Equal(t, baseURL, server.URL+"/gateway/api/automation/public/jira/my-cloud-id/rest/v1")
}

func TestListAutomationRules_CancelledContext(t *testing.T) {
	t.Parallel()
	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.ListAutomationRules(ctx)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "listing automation rules")
}

func TestListAutomationRules(t *testing.T) {
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
				{UUID: "uuid-1", Name: "Rule One", State: "ENABLED"},
				{UUID: "uuid-2", Name: "Rule Two", State: "DISABLED"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	rules, err := client.ListAutomationRules(context.Background())
	testutil.RequireNoError(t, err)
	testutil.Len(t, rules, 2)
	testutil.Equal(t, rules[0].Name, "Rule One")
	testutil.Equal(t, rules[0].State, "ENABLED")
	testutil.Equal(t, rules[1].Name, "Rule Two")
	testutil.Equal(t, rules[1].State, "DISABLED")
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
		rules, err := client.ListAutomationRulesFiltered(context.Background(), "ENABLED")
		testutil.RequireNoError(t, err)
		testutil.Len(t, rules, 2)
		for _, r := range rules {
			testutil.Equal(t, r.State, "ENABLED")
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

		rules, err := client2.ListAutomationRulesFiltered(context.Background(), "DISABLED")
		testutil.RequireNoError(t, err)
		testutil.Len(t, rules, 1)
		testutil.Equal(t, rules[0].Name, "Disabled Rule")
	})

	t.Run("no filter", func(t *testing.T) {
		rules, err := client.ListAutomationRulesFiltered(context.Background(), "")
		testutil.RequireNoError(t, err)
		testutil.Len(t, rules, 3)
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
				{Component: "CONDITION", Type: "jira.jql.condition"},
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

	rule, err := client.GetAutomationRule(context.Background(), "42")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, rule.Name, "My Automation Rule")
	testutil.Equal(t, rule.State, "ENABLED")
	testutil.NotNil(t, rule.Trigger)
	testutil.Equal(t, rule.Trigger.Component, "TRIGGER")
	testutil.Len(t, rule.Components, 2)
	testutil.Equal(t, rule.Components[0].Component, "CONDITION")
	testutil.Equal(t, rule.Components[1].Component, "ACTION")
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

	raw, err := client.GetAutomationRuleRaw(context.Background(), "42")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, string(raw), expectedJSON)
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
	err := client.UpdateAutomationRule(context.Background(), "42", ruleJSON)
	testutil.RequireNoError(t, err)
	jsonEq(t, string(receivedBody), `{"name":"Updated Rule","state":"ENABLED"}`)
}

func TestSetAutomationRuleState(t *testing.T) {
	t.Run("enable", func(t *testing.T) {
		var rawBody json.RawMessage

		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
				return
			}

			_ = json.NewDecoder(r.Body).Decode(&rawBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer server.Close()

		err := client.SetAutomationRuleState(context.Background(), "42", true)
		testutil.RequireNoError(t, err)
		// Verify the JSON field name is "value" per the Automation REST API spec
		jsonEq(t, string(rawBody), `{"value":"ENABLED"}`)
	})

	t.Run("disable", func(t *testing.T) {
		var rawBody json.RawMessage

		client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
				return
			}

			_ = json.NewDecoder(r.Body).Decode(&rawBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer server.Close()

		err := client.SetAutomationRuleState(context.Background(), "42", false)
		testutil.RequireNoError(t, err)
		jsonEq(t, string(rawBody), `{"value":"DISABLED"}`)
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
	resp, err := client.CreateAutomationRule(context.Background(), ruleJSON)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, receivedMethod, http.MethodPost)
	jsonEq(t, string(receivedBody), `{"name":"New Rule","state":"DISABLED"}`)

	var created struct {
		ID      json.Number `json:"id"`
		RuleKey string      `json:"ruleKey"`
		Name    string      `json:"name"`
	}
	testutil.RequireNoError(t, json.Unmarshal(resp, &created))
	testutil.Equal(t, created.ID.String(), "99")
	testutil.Equal(t, created.RuleKey, "new-uuid-123")
	testutil.Equal(t, created.Name, "New Rule")
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
			testutil.Equal(t, tt.rule.Identifier(), tt.expected)
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
			testutil.Equal(t, tt.summary.Identifier(), tt.expected)
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
		testutil.Len(t, items, 1)
		testutil.Equal(t, items[0].UUID, "d-1")
	})

	t.Run("falls back to Values when Data is empty", func(t *testing.T) {
		resp := AutomationRuleSummaryResponse{
			Values: []AutomationRuleSummary{
				{ID: json.Number("1"), Name: "Legacy Rule"},
				{ID: json.Number("2"), Name: "Legacy Rule 2"},
			},
		}
		items := resp.Items()
		testutil.Len(t, items, 2)
		testutil.Equal(t, items[0].Name, "Legacy Rule")
	})
}

func TestNextURLLegacyFallback(t *testing.T) {
	t.Run("returns Links.Next when present", func(t *testing.T) {
		next := "http://example.com/next"
		resp := AutomationRuleSummaryResponse{
			Links: automationLinks{Next: &next},
			Next:  "http://example.com/legacy-next",
		}
		testutil.Equal(t, resp.NextURL(), "http://example.com/next")
	})

	t.Run("falls back to top-level Next", func(t *testing.T) {
		resp := AutomationRuleSummaryResponse{
			Next: "http://example.com/legacy-next",
		}
		testutil.Equal(t, resp.NextURL(), "http://example.com/legacy-next")
	})

	t.Run("returns empty when no next URL", func(t *testing.T) {
		resp := AutomationRuleSummaryResponse{}
		testutil.Equal(t, resp.NextURL(), "")
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

		rule, err := client.GetAutomationRule(context.Background(), "42")
		testutil.RequireNoError(t, err)
		testutil.Equal(t, rule.Name, "Legacy Rule")
		testutil.Equal(t, rule.State, "ENABLED")
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

		rule, err := client.GetAutomationRule(context.Background(), "rk-99")
		testutil.RequireNoError(t, err)
		testutil.Equal(t, rule.UUID, "rk-99")
		testutil.Equal(t, rule.RuleKey, "rk-99")
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

		rule, err := client.GetAutomationRule(context.Background(), "rk-envelope")
		testutil.RequireNoError(t, err)
		testutil.Equal(t, rule.UUID, "rk-envelope")
		testutil.Equal(t, rule.Name, "Envelope RuleKey")
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

	rules, err := client.ListAutomationRules(context.Background())
	testutil.RequireNoError(t, err)
	testutil.Len(t, rules, 2)
	testutil.Equal(t, rules[0].Name, "Old Rule 1")
	testutil.Equal(t, rules[1].Name, "Old Rule 2")
}

func TestListAutomationRulesPagination(t *testing.T) {
	var page int32
	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
			return
		}

		p := atomic.AddInt32(&page, 1)
		w.WriteHeader(http.StatusOK)
		if p == 1 {
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

	rules, err := client.ListAutomationRules(context.Background())
	testutil.RequireNoError(t, err)
	testutil.Len(t, rules, 3)
	testutil.Equal(t, rules[0].Name, "Rule 1")
	testutil.Equal(t, rules[1].Name, "Rule 2")
	testutil.Equal(t, rules[2].Name, "Rule 3")
	testutil.Equal(t, atomic.LoadInt32(&page), int32(2)) // Verify two pages were fetched
}

func TestDeleteAutomationRule(t *testing.T) {
	t.Parallel()

	var receivedMethod string
	var receivedPath string

	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"cloud-1"}`))
			return
		}

		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := client.DeleteAutomationRule(context.Background(), "rule-uuid-123")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, receivedMethod, http.MethodDelete)
	testutil.Contains(t, receivedPath, "/rule/rule-uuid-123")
}
