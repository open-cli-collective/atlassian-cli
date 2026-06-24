package present

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/keyring"
	sharedpresent "github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/confluence-cli/api"
)

type ConfigPresenter struct{}

func (ConfigPresenter) PresentTestProgress() *sharedpresent.OutputModel {
	return &sharedpresent.OutputModel{Sections: []sharedpresent.Section{
		&sharedpresent.MessageSection{
			Kind:      sharedpresent.MessageInfo,
			Message:   "Testing connection... ",
			Stream:    sharedpresent.StreamStderr,
			NoNewline: true,
		},
	}}
}

func (ConfigPresenter) PresentTestFailure() *sharedpresent.OutputModel {
	return stderrOnly(strings.Join([]string{
		"failed!",
		"",
		"Troubleshooting:",
		"  - Verify your URL is correct (should include https://)",
		"  - Check your email and API token",
		"  - Ensure your API token hasn't expired",
		"  - Verify you have permission to access Confluence",
		"",
		"To regenerate an API token:",
		"  https://id.atlassian.com/manage-profile/security/api-tokens",
	}, "\n"))
}

func (ConfigPresenter) PresentTestConnectionSuccess() *sharedpresent.OutputModel {
	return stderrOnly("success!\n")
}

func (ConfigPresenter) PresentTestSuccess(user *api.User) *sharedpresent.OutputModel {
	if user == nil {
		return stderrOnly("Your cfl configuration is working correctly.")
	}

	lines := []string{"Authentication successful", "API access verified", ""}
	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.PublicName
	}
	if displayName != "" {
		if user.Email != "" {
			lines = append(lines, fmt.Sprintf("Authenticated as: %s (%s)", displayName, user.Email))
		} else {
			lines = append(lines, fmt.Sprintf("Authenticated as: %s", displayName))
		}
	}
	if user.AccountID != "" {
		lines = append(lines, fmt.Sprintf("Account ID: %s", user.AccountID))
	}

	return stderrOnly(strings.Join(lines, "\n"))
}

func (ConfigPresenter) PresentClearDefaultPlan(plan keyring.ClearPlan) *sharedpresent.OutputModel {
	lines := []string{
		fmt.Sprintf("This will delete key %q from keyring %s.", plan.ToolKey, plan.Ref),
		"Warning: this is the SHARED token (api_token). jtk will also lose access (cfl and jtk resolve the same key).",
	}
	return stderrOnly(strings.Join(lines, "\n"))
}

func (ConfigPresenter) PresentClearAllPlan(plan keyring.ClearPlan, keyringErr error) *sharedpresent.OutputModel {
	lines := []string{fmt.Sprintf("This will remove the ENTIRE shared keyring bundle %s%s.",
		plan.Ref,
		existingKeysSuffix(plan.ExistingKeys),
	)}
	if plan.SharedConfigPath != "" {
		lines = append(lines, fmt.Sprintf("It will also delete the shared config file: %s", plan.SharedConfigPath))
	}
	if plan.OldSharedConfigPath != "" {
		lines = append(lines, fmt.Sprintf("It will also delete the prior shared config file: %s", plan.OldSharedConfigPath))
	}
	for _, lp := range plan.LegacyPaths {
		lines = append(lines, fmt.Sprintf("It will scrub the legacy plaintext file: %s", lp))
	}
	if keyringErr != nil {
		lines = append(lines, fmt.Sprintf("Note: the keyring could not be opened (%v); plaintext artifacts will still be cleaned, but the keyring bundle will be left intact.", keyringErr))
	}
	return stderrOnly(strings.Join(lines, "\n"))
}

func (ConfigPresenter) PresentClearNoStoredToken(plan keyring.ClearPlan) *sharedpresent.OutputModel {
	return stderrOnly(joinWithEnvNote(
		fmt.Sprintf("No stored API token in keyring %s for cfl; nothing to clear.", plan.Ref),
		plan.EnvActive,
	))
}

func (ConfigPresenter) PresentClearCancelled() *sharedpresent.OutputModel {
	return stderrOnly("Cancelled. Nothing was cleared.")
}

func (ConfigPresenter) PresentClearDefaultSuccess(plan keyring.ClearPlan) *sharedpresent.OutputModel {
	return stderrOnly(joinWithEnvNote(
		fmt.Sprintf("Removed key %q from keyring %s.", plan.ToolKey, plan.Ref),
		plan.EnvActive,
	))
}

func (ConfigPresenter) PresentClearAllSuccess(plan keyring.ClearPlan) *sharedpresent.OutputModel {
	return stderrOnly(joinWithEnvNote("Removed the shared keyring bundle and config file.", plan.EnvActive))
}

func existingKeysSuffix(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	return fmt.Sprintf(" (keys: %s)", strings.Join(keys, ", "))
}

func joinWithEnvNote(message string, envActive []string) string {
	if len(envActive) == 0 {
		return message
	}
	return message + "\n" + fmt.Sprintf(
		"Note: %s still set in the environment and will continue to override at runtime (not cleared).",
		strings.Join(envActive, ", "),
	)
}
