package api

import (
	"errors"

	sharederrors "github.com/open-cli-collective/atlassian-go/errors"
)

// Jira-specific validation errors
var (
	ErrIssueKeyRequired   = errors.New("issue key is required")
	ErrProjectKeyRequired = errors.New("project key is required")
)

// APIError is an alias for the shared APIError type
type APIError = sharederrors.APIError
