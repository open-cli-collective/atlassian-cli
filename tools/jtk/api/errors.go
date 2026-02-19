package api //nolint:revive // package name is intentional

import (
	"errors"

	sharederrors "github.com/open-cli-collective/atlassian-go/errors"
)

// Jira-specific validation errors
var (
	ErrIssueKeyRequired         = errors.New("issue key is required")
	ErrProjectKeyRequired       = errors.New("project key is required")
	ErrFieldIDRequired          = errors.New("field ID is required")
	ErrAttachmentIDRequired     = errors.New("attachment ID is required")
	ErrFilePathRequired         = errors.New("file path is required")
	ErrAttachmentRequired       = errors.New("attachment is required")
	ErrAttachmentContentMissing = errors.New("attachment has no content URL")
	ErrCommentIDRequired        = errors.New("comment ID is required")
	ErrTaskIDRequired           = errors.New("task ID is required")
)

// APIError is an alias for the shared APIError type
type APIError = sharederrors.APIError //nolint:revive // preserving backward compat alias
