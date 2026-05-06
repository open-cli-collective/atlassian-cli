package credstore

import (
	"fmt"
	"strings"
)

// SectionsEqual reports whether two Sections describe the same
// credentials. URLs are normalized to base form so
// "https://acme.atlassian.net/wiki" and "https://acme.atlassian.net"
// don't read as different. AuthMethod "" is canonicalized to "basic"
// before comparison since legacy configs commonly omit the field while
// migrated configs write it explicitly.
func SectionsEqual(a, b Section) bool {
	return NormalizeBaseURL(a.URL) == NormalizeBaseURL(b.URL) &&
		a.Email == b.Email &&
		a.APIToken == b.APIToken &&
		canonAuthMethod(a.AuthMethod) == canonAuthMethod(b.AuthMethod) &&
		a.CloudID == b.CloudID
}

func canonAuthMethod(m string) string {
	if m == "" {
		return "basic"
	}
	return m
}

// MaskToken returns a redacted form of an API token suitable for
// display in reconciliation prompts: first 4 + "..." + last 4 for
// long tokens, "***" for short ones, "" for empty.
func MaskToken(t string) string {
	if t == "" {
		return ""
	}
	if len(t) <= 8 {
		return "***"
	}
	return t[:4] + "..." + t[len(t)-4:]
}

// FormatSection renders a Section as a four-line block suitable for
// embedding in interactive reconciliation prompts. Empty fields are
// rendered as "(unset)" so the user sees the full schema either way.
func FormatSection(label string, s Section) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  %s:\n", label)
	fmt.Fprintf(&b, "    url:    %s\n", display(NormalizeBaseURL(s.URL)))
	fmt.Fprintf(&b, "    email:  %s\n", display(s.Email))
	fmt.Fprintf(&b, "    token:  %s\n", display(MaskToken(s.APIToken)))
	method := s.AuthMethod
	if method == "" {
		method = "basic"
	}
	fmt.Fprintf(&b, "    method: %s", method)
	if s.CloudID != "" {
		fmt.Fprintf(&b, " (cloud_id: %s)", s.CloudID)
	}
	return b.String()
}

func display(s string) string {
	if s == "" {
		return "(unset)"
	}
	return s
}
