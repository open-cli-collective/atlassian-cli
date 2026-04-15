package present

import (
	"fmt"
	"time"
)

// FormatDate formats a time.Time as a short date string.
// Returns empty string for nil or zero time.
func FormatDate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// FormatDateTime formats a time.Time with date and time.
// Returns empty string for nil or zero time.
func FormatDateTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

// TruncateText truncates text to maxLen characters, adding "..." if truncated.
func TruncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// BoolString returns "Yes" or "No" for a boolean value.
func BoolString(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// OrDash returns the string or "-" if empty.
func OrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// FormatAssignee returns the assignee name or "Unassigned" if empty.
func FormatAssignee(name string) string {
	if name == "" {
		return "Unassigned"
	}
	return name
}

// FormatInt formats an integer as a string.
func FormatInt(n int) string {
	return fmt.Sprintf("%d", n)
}
