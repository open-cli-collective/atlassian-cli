// Package config provides configuration utilities for Atlassian CLI tools.
package config

import "os"

// GetEnvWithFallback returns the value of the primary environment variable.
// If the primary variable is empty or not set, it returns the value of the
// fallback environment variable instead.
//
// This is useful for implementing environment variable precedence patterns like:
//
//	CFL_URL → ATLASSIAN_URL → config file
//	JIRA_URL → ATLASSIAN_URL → config file
//
// Example:
//
//	url := GetEnvWithFallback("CFL_URL", "ATLASSIAN_URL")
func GetEnvWithFallback(primary, fallback string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	return os.Getenv(fallback)
}

// GetEnvWithDefault returns the value of the environment variable,
// or the default value if the variable is empty or not set.
func GetEnvWithDefault(name, defaultValue string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return defaultValue
}
