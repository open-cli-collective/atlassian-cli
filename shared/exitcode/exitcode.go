// Package exitcode provides standardized exit codes for Atlassian CLI tools.
//
// These exit codes follow Unix conventions where 0 indicates success
// and non-zero values indicate different types of errors.
package exitcode

// Exit codes for CLI tools
const (
	// Success indicates the command completed successfully
	Success = 0

	// GeneralError indicates an unspecified error occurred
	GeneralError = 1

	// UsageError indicates invalid command-line usage
	UsageError = 2

	// ConfigError indicates a configuration problem
	ConfigError = 3

	// AuthError indicates an authentication failure
	AuthError = 4

	// NotFoundError indicates a requested resource was not found
	NotFoundError = 5

	// PermissionError indicates insufficient permissions
	PermissionError = 6

	// RateLimitError indicates API rate limiting
	RateLimitError = 7

	// ServerError indicates a server-side error
	ServerError = 8
)
