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

// BoolString returns "yes" or "no" for a boolean value.
func BoolString(b bool) string {
	if b {
		return "yes"
	}
	return "no"
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

// FormatSize formats a size in bytes as a human-readable string.
func FormatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatTime extracts the date portion from an ISO 8601 formatted timestamp string.
// Jira returns ISO 8601 format, this just shows the date part.
func FormatTime(t string) string {
	// Jira returns ISO 8601 format, just show date
	if len(t) >= 10 {
		return t[:10]
	}
	return t
}
