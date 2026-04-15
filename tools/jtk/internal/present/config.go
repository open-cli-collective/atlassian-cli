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

// PresentTestResult creates a complete output model for the config test command.
// Parameters:
//   - url: Jira URL being tested (empty if not configured)
//   - user: User info if auth succeeded, nil otherwise
//   - clientErr: Error from client creation, nil if successful
//   - authErr: Error from authentication, nil if successful
func (ConfigPresenter) PresentTestResult(url string, user *api.User, clientErr, authErr error) *present.OutputModel {
	var sections []present.Section

	// Case 1: No URL configured
	if url == "" {
		sections = append(sections,
			&present.MessageSection{Kind: present.MessageError, Message: "No Jira URL configured", Stream: present.StreamStderr},
			&present.MessageSection{Kind: present.MessageInfo, Message: "Configure with: jtk init"},
			&present.MessageSection{Kind: present.MessageInfo, Message: "Or set environment variable: JIRA_URL"},
		)
		return &present.OutputModel{Sections: sections}
	}

	// Show what we're testing
	sections = append(sections,
		&present.MessageSection{Kind: present.MessageInfo, Message: fmt.Sprintf("Testing connection to %s...", url)},
	)

	// Case 2: Client creation failed
	if clientErr != nil {
		sections = append(sections,
			&present.MessageSection{Kind: present.MessageError, Message: fmt.Sprintf("Failed to create client: %v", clientErr), Stream: present.StreamStderr},
			&present.MessageSection{Kind: present.MessageInfo, Message: "Check your configuration with: jtk config show"},
			&present.MessageSection{Kind: present.MessageInfo, Message: "Reconfigure with: jtk init"},
		)
		return &present.OutputModel{Sections: sections}
	}

	// Case 3: Authentication failed
	if authErr != nil {
		sections = append(sections,
			&present.MessageSection{Kind: present.MessageError, Message: fmt.Sprintf("Authentication failed: %v", authErr), Stream: present.StreamStderr},
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

	if user != nil {
		sections = append(sections,
			&present.MessageSection{Kind: present.MessageInfo, Message: fmt.Sprintf("Authenticated as: %s (%s)", user.DisplayName, user.EmailAddress)},
			&present.MessageSection{Kind: present.MessageInfo, Message: fmt.Sprintf("Account ID: %s", user.AccountID)},
		)
	}

	return &present.OutputModel{Sections: sections}
}

// configEntry represents a single configuration entry (internal use only).
type configEntry struct {
	key    string
	value  string
	source string
}



// PresentConfigShow creates config table + path info as single output.
// Accepts pre-computed (value, source) pairs for each config field.
func (ConfigPresenter) PresentConfigShow(
	url, urlSrc,
	email, emailSrc,
	maskedToken, tokenSrc,
	defaultProject, projectSrc,
	authMethod, authMethodSrc,
	cloudID, cloudIDSrc,
	configPath string,
) *present.OutputModel {
	entries := []configEntry{
		{key: "url", value: url, source: urlSrc},
		{key: "email", value: email, source: emailSrc},
		{key: "api_token", value: maskedToken, source: tokenSrc},
		{key: "default_project", value: defaultProject, source: projectSrc},
		{key: "auth_method", value: authMethod, source: authMethodSrc},
		{key: "cloud_id", value: cloudID, source: cloudIDSrc},
	}

	rows := make([]present.Row, len(entries))
	for i, e := range entries {
		rows[i] = present.Row{
			Cells: []string{e.key, e.value, e.source},
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
