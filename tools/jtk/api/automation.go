package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// ListAutomationRules returns summaries of all automation rules.
func (c *Client) ListAutomationRules(ctx context.Context) ([]AutomationRuleSummary, error) {
	base, err := c.AutomationBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing automation rules: %w", err)
	}

	var all []AutomationRuleSummary
	urlStr := fmt.Sprintf("%s/rule/summary", base)

	for urlStr != "" {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("listing automation rules: %w", err)
		}

		body, err := c.Get(ctx, urlStr)
		if err != nil {
			return nil, fmt.Errorf("listing automation rules: %w", err)
		}

		var resp AutomationRuleSummaryResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parsing automation rules response: %w", err)
		}

		all = append(all, resp.Items()...)

		if next := resp.NextURL(); next != "" {
			urlStr = next
		} else {
			urlStr = ""
		}
	}

	return all, nil
}

// ListAutomationRulesFiltered returns rule summaries filtered by state.
func (c *Client) ListAutomationRulesFiltered(ctx context.Context, state string) ([]AutomationRuleSummary, error) {
	rules, err := c.ListAutomationRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing automation rules (filtered): %w", err)
	}

	if state == "" {
		return rules, nil
	}

	var filtered []AutomationRuleSummary
	for _, r := range rules {
		if r.State == state {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// GetAutomationRule returns the full rule definition including components.
func (c *Client) GetAutomationRule(ctx context.Context, ruleID string) (*AutomationRule, error) {
	base, err := c.AutomationBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting automation rule %s: %w", ruleID, err)
	}

	urlStr := fmt.Sprintf("%s/rule/%s", base, url.PathEscape(ruleID))
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("getting automation rule %s: %w", ruleID, err)
	}

	// Newer Cloud response shape is an envelope: {"rule": {...}, "connections": [...]}
	var env struct {
		Rule *AutomationRule `json:"rule"`
	}
	if err := json.Unmarshal(body, &env); err == nil && env.Rule != nil {
		// Normalize legacy identifier field name for callers.
		if env.Rule.UUID == "" && env.Rule.RuleKey != "" {
			env.Rule.UUID = env.Rule.RuleKey
		}
		return env.Rule, nil
	}

	// Fallback to legacy/plain shape where the rule is the top-level object.
	var rule AutomationRule
	if err := json.Unmarshal(body, &rule); err != nil {
		return nil, fmt.Errorf("parsing automation rule: %w", err)
	}
	if rule.UUID == "" && rule.RuleKey != "" {
		rule.UUID = rule.RuleKey
	}

	return &rule, nil
}

// GetAutomationRuleRaw returns the full rule definition as raw JSON bytes.
// This is used for the export command to preserve exact JSON for round-tripping.
func (c *Client) GetAutomationRuleRaw(ctx context.Context, ruleID string) ([]byte, error) {
	base, err := c.AutomationBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting automation rule %s (raw): %w", ruleID, err)
	}

	urlStr := fmt.Sprintf("%s/rule/%s", base, url.PathEscape(ruleID))
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("getting automation rule %s: %w", ruleID, err)
	}

	return body, nil
}

// UpdateAutomationRule replaces a rule definition with the provided raw JSON.
// The caller should have obtained the JSON via GetAutomationRuleRaw or export,
// modified it, and passed it back here.
func (c *Client) UpdateAutomationRule(ctx context.Context, ruleID string, ruleJSON json.RawMessage) error {
	base, err := c.AutomationBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("updating automation rule %s: %w", ruleID, err)
	}

	urlStr := fmt.Sprintf("%s/rule/%s", base, url.PathEscape(ruleID))
	_, err = c.Put(ctx, urlStr, ruleJSON)
	if err != nil {
		return fmt.Errorf("updating automation rule %s: %w", ruleID, err)
	}

	return nil
}

// CreateAutomationRule creates a new automation rule from raw JSON.
// The JSON should be in the same shape as the GET response. The API
// auto-generates new IDs; any existing 'id' or 'ruleKey' fields are ignored.
func (c *Client) CreateAutomationRule(ctx context.Context, ruleJSON json.RawMessage) (json.RawMessage, error) {
	base, err := c.AutomationBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating automation rule: %w", err)
	}

	urlStr := fmt.Sprintf("%s/rule", base)
	body, err := c.Post(ctx, urlStr, ruleJSON)
	if err != nil {
		return nil, fmt.Errorf("creating automation rule: %w", err)
	}

	return body, nil
}

// SetAutomationRuleState enables or disables an automation rule.
func (c *Client) SetAutomationRuleState(ctx context.Context, ruleID string, enabled bool) error {
	base, err := c.AutomationBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("setting automation rule %s state: %w", ruleID, err)
	}

	state := "DISABLED"
	if enabled {
		state = "ENABLED"
	}

	urlStr := fmt.Sprintf("%s/rule/%s/state", base, url.PathEscape(ruleID))
	_, err = c.Put(ctx, urlStr, AutomationStateUpdate{Value: state})
	if err != nil {
		return fmt.Errorf("setting automation rule %s state to %s: %w", ruleID, state, err)
	}

	return nil
}
