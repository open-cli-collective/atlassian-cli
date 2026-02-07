package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ListAutomationRules returns summaries of all automation rules.
func (c *Client) ListAutomationRules() ([]AutomationRuleSummary, error) {
	base, err := c.AutomationBaseURL()
	if err != nil {
		return nil, err
	}

	var all []AutomationRuleSummary
	urlStr := fmt.Sprintf("%s/rule/summary", base)

	for urlStr != "" {
		body, err := c.get(urlStr)
		if err != nil {
			return nil, fmt.Errorf("failed to list automation rules: %w", err)
		}

		var resp AutomationRuleSummaryResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse automation rules response: %w", err)
		}

		all = append(all, resp.Values...)

		if resp.Next != "" {
			urlStr = resp.Next
		} else {
			urlStr = ""
		}
	}

	return all, nil
}

// ListAutomationRulesFiltered returns rule summaries filtered by state.
func (c *Client) ListAutomationRulesFiltered(state string) ([]AutomationRuleSummary, error) {
	rules, err := c.ListAutomationRules()
	if err != nil {
		return nil, err
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
func (c *Client) GetAutomationRule(ruleID string) (*AutomationRule, error) {
	base, err := c.AutomationBaseURL()
	if err != nil {
		return nil, err
	}

	urlStr := fmt.Sprintf("%s/rule/%s", base, url.PathEscape(ruleID))
	body, err := c.get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get automation rule %s: %w", ruleID, err)
	}

	var rule AutomationRule
	if err := json.Unmarshal(body, &rule); err != nil {
		return nil, fmt.Errorf("failed to parse automation rule: %w", err)
	}

	return &rule, nil
}

// GetAutomationRuleRaw returns the full rule definition as raw JSON bytes.
// This is used for the export command to preserve exact JSON for round-tripping.
func (c *Client) GetAutomationRuleRaw(ruleID string) ([]byte, error) {
	base, err := c.AutomationBaseURL()
	if err != nil {
		return nil, err
	}

	urlStr := fmt.Sprintf("%s/rule/%s", base, url.PathEscape(ruleID))
	body, err := c.get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get automation rule %s: %w", ruleID, err)
	}

	return body, nil
}

// UpdateAutomationRule replaces a rule definition with the provided raw JSON.
// The caller should have obtained the JSON via GetAutomationRuleRaw or export,
// modified it, and passed it back here.
func (c *Client) UpdateAutomationRule(ruleID string, ruleJSON json.RawMessage) error {
	base, err := c.AutomationBaseURL()
	if err != nil {
		return err
	}

	urlStr := fmt.Sprintf("%s/rule/%s", base, url.PathEscape(ruleID))
	_, err = c.put(urlStr, ruleJSON)
	if err != nil {
		return fmt.Errorf("failed to update automation rule %s: %w", ruleID, err)
	}

	return nil
}

// CreateAutomationRule creates a new automation rule from raw JSON.
// The JSON should be in the same shape as the GET response. The API
// auto-generates new IDs; any existing 'id' or 'ruleKey' fields are ignored.
func (c *Client) CreateAutomationRule(ruleJSON json.RawMessage) (json.RawMessage, error) {
	base, err := c.AutomationBaseURL()
	if err != nil {
		return nil, err
	}

	urlStr := fmt.Sprintf("%s/rule", base)
	body, err := c.post(urlStr, ruleJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create automation rule: %w", err)
	}

	return body, nil
}

// SetAutomationRuleState enables or disables an automation rule.
func (c *Client) SetAutomationRuleState(ruleID string, enabled bool) error {
	base, err := c.AutomationBaseURL()
	if err != nil {
		return err
	}

	state := "DISABLED"
	if enabled {
		state = "ENABLED"
	}

	urlStr := fmt.Sprintf("%s/rule/%s/state", base, url.PathEscape(ruleID))
	_, err = c.put(urlStr, AutomationStateUpdate{RuleState: state})
	if err != nil {
		return fmt.Errorf("failed to set automation rule %s state to %s: %w", ruleID, state, err)
	}

	return nil
}
