// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// ConfigPresenter creates presentation models for config commands.
type ConfigPresenter struct{}

// TestResult contains the outcome of a config test operation.
type TestResult struct {
	URL         string    // Jira URL being tested (empty if not configured)
	User        *api.User // User info if auth succeeded, nil otherwise
	ClientError error     // Error from client creation, nil if successful
	AuthError   error     // Error from authentication, nil if successful
}

// PresentTestResult creates a complete output model for the config test command.
func (ConfigPresenter) PresentTestResult(r TestResult) *present.OutputModel {
	var sections []present.Section

	// Case 1: No URL configured
	if r.URL == "" {
		sections = append(sections,
			&present.MessageSection{Kind: present.MessageError, Message: "No Jira URL configured", Stream: present.StreamStderr},
			&present.MessageSection{Kind: present.MessageInfo, Message: "Configure with: jtk init"},
			&present.MessageSection{Kind: present.MessageInfo, Message: "Or set environment variable: JIRA_URL"},
		)
		return &present.OutputModel{Sections: sections}
	}

	// Show what we're testing
	sections = append(sections,
		&present.MessageSection{Kind: present.MessageInfo, Message: fmt.Sprintf("Testing connection to %s...", r.URL)},
	)

	// Case 2: Client creation failed
	if r.ClientError != nil {
		sections = append(sections,
			&present.MessageSection{Kind: present.MessageError, Message: fmt.Sprintf("Failed to create client: %v", r.ClientError), Stream: present.StreamStderr},
			&present.MessageSection{Kind: present.MessageInfo, Message: "Check your configuration with: jtk config show"},
			&present.MessageSection{Kind: present.MessageInfo, Message: "Reconfigure with: jtk init"},
		)
		return &present.OutputModel{Sections: sections}
	}

	// Case 3: Authentication failed
	if r.AuthError != nil {
		sections = append(sections,
			&present.MessageSection{Kind: present.MessageError, Message: fmt.Sprintf("Authentication failed: %v", r.AuthError), Stream: present.StreamStderr},
			&present.MessageSection{Kind: present.MessageInfo, Message: "Check your credentials with: jtk config show"},
			&present.MessageSection{Kind: present.MessageInfo, Message: "Reconfigure with: jtk init"},
		)
		return &present.OutputModel{Sections: sections}
	}

	// Case 4: Success
	sections = append(sections,
		&present.MessageSection{Kind: present.MessageSuccess, Message: "Authentication successful"},
		&present.MessageSection{Kind: present.MessageSuccess, Message: "API access verified"},
	)

	if r.User != nil {
		sections = append(sections,
			&present.MessageSection{Kind: present.MessageInfo, Message: fmt.Sprintf("Authenticated as: %s (%s)", r.User.DisplayName, r.User.EmailAddress)},
			&present.MessageSection{Kind: present.MessageInfo, Message: fmt.Sprintf("Account ID: %s", r.User.AccountID)},
		)
	}

	return &present.OutputModel{Sections: sections}
}

// ConfigEntry represents a single configuration entry.
type ConfigEntry struct {
	Key    string
	Value  string
	Source string
}

// PresentConfig creates a table presentation model for configuration entries.
func (ConfigPresenter) PresentConfig(entries []ConfigEntry) *present.OutputModel {
	rows := make([]present.Row, len(entries))
	for i, e := range entries {
		rows[i] = present.Row{
			Cells: []string{e.Key, e.Value, e.Source},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"KEY", "VALUE", "SOURCE"},
				Rows:    rows,
			},
		},
	}
}

// PresentConfigPath creates an info message showing the config file path.
func (ConfigPresenter) PresentConfigPath(path string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("\nConfig file: %s", path),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentConfigWithPath creates config entries + path info as single output.
func (ConfigPresenter) PresentConfigWithPath(entries []ConfigEntry, configPath string) *present.OutputModel {
	rows := make([]present.Row, len(entries))
	for i, e := range entries {
		rows[i] = present.Row{
			Cells: []string{e.Key, e.Value, e.Source},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"KEY", "VALUE", "SOURCE"},
				Rows:    rows,
			},
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("\nConfig file: %s", configPath),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentCleared creates a success message for config file removal.
func (ConfigPresenter) PresentCleared(path string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Configuration file removed: %s", path),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentClearedWithEnvVars creates a success message with env var advisory.
func (ConfigPresenter) PresentClearedWithEnvVars(path string, envVars []string) *present.OutputModel {
	sections := []present.Section{
		&present.MessageSection{
			Kind:    present.MessageSuccess,
			Message: fmt.Sprintf("Configuration file removed: %s", path),
			Stream:  present.StreamStdout,
		},
	}

	if len(envVars) > 0 {
		sections = append(sections,
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("\nNote: The following are still configured via environment variables: %s", strings.Join(envVars, ", ")),
				Stream:  present.StreamStderr,
			},
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "These will continue to be used. Unset them if you want to fully clear configuration.",
				Stream:  present.StreamStderr,
			},
		)
	}

	return &present.OutputModel{Sections: sections}
}

// PresentNoConfig creates an info message when no config file exists.
func (ConfigPresenter) PresentNoConfig(path string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No configuration file found at %s", path),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentClearCancelled creates an info message for cancelled config clear.
func (ConfigPresenter) PresentClearCancelled() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "Cancelled.",
				Stream:  present.StreamStdout,
			},
		},
	}
}
