package issues

import (
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// resolveFields determines which fields to request from the Jira API based on
// the --fields flag, output format, and --all-fields flag.
func resolveFields(fieldsFlag, outputFormat string, allFields bool) []string {
	if fieldsFlag != "" {
		parts := strings.Split(fieldsFlag, ",")
		fields := make([]string, 0, len(parts))
		for _, p := range parts {
			if f := strings.TrimSpace(p); f != "" {
				fields = append(fields, f)
			}
		}
		if len(fields) > 0 {
			return fields
		}
		// Fall through to defaults if all tokens were empty/whitespace
	}
	if outputFormat == "json" {
		return []string{"*all"}
	}
	if allFields {
		return append([]string(nil), api.DefaultSearchFields...)
	}
	return append([]string(nil), api.ListSearchFields...)
}
