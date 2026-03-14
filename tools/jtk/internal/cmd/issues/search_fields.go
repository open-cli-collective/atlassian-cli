package issues

import (
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// resolveFields determines which fields to request from the Jira API based on
// the --fields flag, output format, and --full flag.
func resolveFields(fieldsFlag, outputFormat string, full bool) []string {
	if fieldsFlag != "" {
		return strings.Split(fieldsFlag, ",")
	}
	if outputFormat == "json" {
		return []string{"*all"}
	}
	if full {
		return api.DefaultSearchFields
	}
	return api.ListSearchFields
}
