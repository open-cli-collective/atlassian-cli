package config

import "os"

// ValuesWithSources holds all config values with their source information.
// This is a projection helper that inspects env vars and config file to determine
// where each value came from. Used by commands to pass resolved values to presenters.
type ValuesWithSources struct {
	URL            string
	URLSource      string
	Email          string
	EmailSource    string
	APIToken       string // Unmasked - caller should mask before passing to presenter
	TokenSource    string
	DefaultProject string
	ProjectSource  string
	AuthMethod     string
	AuthMethodSrc  string
	CloudID        string
	CloudIDSrc     string
	Path           string
}

// GetValuesWithSources returns all config values with their source information.
func GetValuesWithSources() ValuesWithSources {
	url, urlSrc := GetURLWithSource()
	email, emailSrc := GetEmailWithSource()
	token, tokenSrc := GetAPITokenWithSource()
	project, projectSrc := GetDefaultProjectWithSource()
	authMethod, authMethodSrc := GetAuthMethodWithSource()
	cloudID, cloudIDSrc := GetCloudIDWithSource()

	return ValuesWithSources{
		URL:            url,
		URLSource:      urlSrc,
		Email:          email,
		EmailSource:    emailSrc,
		APIToken:       token,
		TokenSource:    tokenSrc,
		DefaultProject: project,
		ProjectSource:  projectSrc,
		AuthMethod:     authMethod,
		AuthMethodSrc:  authMethodSrc,
		CloudID:        cloudID,
		CloudIDSrc:     cloudIDSrc,
		Path:           Path(),
	}
}

// GetURLWithSource returns the URL and its source.
// Precedence: JIRA_URL → ATLASSIAN_URL → config url → JIRA_DOMAIN (legacy) → config domain (legacy)
func GetURLWithSource() (value, source string) {
	if os.Getenv("JIRA_URL") != "" {
		return GetURL(), "env (JIRA_URL)"
	}
	if os.Getenv("ATLASSIAN_URL") != "" {
		return GetURL(), "env (ATLASSIAN_URL)"
	}
	cfg, err := Load()
	if err != nil {
		return "", "-"
	}
	if cfg.URL != "" {
		return GetURL(), "config"
	}
	// Check legacy domain sources
	if os.Getenv("JIRA_DOMAIN") != "" {
		return GetURL(), "env (JIRA_DOMAIN, deprecated)"
	}
	if cfg.Domain != "" {
		return GetURL(), "config (domain, deprecated)"
	}
	return "", "-"
}

// GetEmailWithSource returns the email and its source.
// Precedence: JIRA_EMAIL → ATLASSIAN_EMAIL → config email
func GetEmailWithSource() (value, source string) {
	if os.Getenv("JIRA_EMAIL") != "" {
		return GetEmail(), "env (JIRA_EMAIL)"
	}
	if os.Getenv("ATLASSIAN_EMAIL") != "" {
		return GetEmail(), "env (ATLASSIAN_EMAIL)"
	}
	cfg, err := Load()
	if err != nil {
		return "", "-"
	}
	if cfg.Email != "" {
		return cfg.Email, "config"
	}
	return "", "-"
}

// GetAPITokenWithSource returns the API token and its source.
// Precedence: JIRA_API_TOKEN → ATLASSIAN_API_TOKEN → config api_token
func GetAPITokenWithSource() (value, source string) {
	if os.Getenv("JIRA_API_TOKEN") != "" {
		return GetAPIToken(), "env (JIRA_API_TOKEN)"
	}
	if os.Getenv("ATLASSIAN_API_TOKEN") != "" {
		return GetAPIToken(), "env (ATLASSIAN_API_TOKEN)"
	}
	cfg, err := Load()
	if err != nil {
		return "", "-"
	}
	if cfg.APIToken != "" {
		return cfg.APIToken, "config"
	}
	return "", "-"
}

// GetDefaultProjectWithSource returns the default project and its source.
// Precedence: JIRA_DEFAULT_PROJECT → config default_project
func GetDefaultProjectWithSource() (value, source string) {
	if os.Getenv("JIRA_DEFAULT_PROJECT") != "" {
		return GetDefaultProject(), "env (JIRA_DEFAULT_PROJECT)"
	}
	cfg, err := Load()
	if err != nil {
		return "", "-"
	}
	if cfg.DefaultProject != "" {
		return cfg.DefaultProject, "config"
	}
	return "", "-"
}
