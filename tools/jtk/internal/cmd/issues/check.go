package issues

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

// defaultWarnFields are commonly-important fields that are not enforced by Jira's
// own required-field machinery but tend to matter for workflow hygiene. Each is
// resolved against the project's field metadata at runtime — fields that don't
// exist on the issue's project (e.g., Sprint on a non-Agile project) are
// silently skipped, so no false warnings.
var defaultWarnFields = []string{
	"Summary",
	"Description",
	"Assignee",
	"Priority",
	"Labels",
	"Story Points",
	"Sprint",
	"Components",
	"Fix Version/s",
}

func newCheckCmd(opts *root.Options) *cobra.Command {
	var requireFields []string
	var warnFields []string

	cmd := &cobra.Command{
		Use:   "check <issue-key>",
		Short: "Check that an issue has values for expected fields",
		Long: `Audit an issue for populated/missing field values.

Useful as a guardrail before transitions or as a CI step. Each field can be
named by its display name (e.g. "Story Points"), its Jira field ID
(e.g. "customfield_10035"), or its property key (e.g. "assignee").

` + "`--require`" + ` fields fail the check (non-zero exit) if missing.
` + "`--warn`" + `    fields are reported but do not change the exit code.

When neither flag is provided, a curated default warn-list is used:
` + strings.Join(defaultWarnFields, ", ") + `.
Fields not present on the issue's project schema are silently skipped.`,
		Example: `  # Default warn list (Story Points, Sprint, Description, Assignee, …)
  jtk issues check PROJ-123

  # Hard-fail if Story Points or Sprint are missing
  jtk issues check PROJ-123 --require "Story Points" --require Sprint

  # Mix required and warning fields, comma-separated
  jtk issues check PROJ-123 --require "Story Points,Sprint" --warn "Description,Assignee"

  # By field ID (instance-specific) or property key
  jtk issues check PROJ-123 --require customfield_10035 --warn assignee`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(cmd.Context(), opts, args[0], requireFields, warnFields)
		},
	}

	cmd.Flags().StringSliceVar(&requireFields, "require", nil, "Field name/ID that must be populated (repeatable; comma-separated accepted)")
	cmd.Flags().StringSliceVar(&warnFields, "warn", nil, "Field name/ID to warn on if missing (repeatable; comma-separated accepted)")

	return cmd
}

// Levels and statuses used in checkResult and surfaced verbatim in the
// LEVEL / STATUS columns of the command output.
const (
	levelRequired = "REQUIRED"
	levelWarn     = "WARN"
	statusOK      = "OK"
	statusMissing = "MISSING"
)

type checkResult struct {
	requested string // user's spelling of the field name
	resolved  string // field ID after resolution (e.g. "customfield_10035")
	display   string // human-friendly name (e.g. "Story Points")
	level     string
	status    string
	value     string
}

func runCheck(ctx context.Context, opts *root.Options, issueKey string, required, warn []string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	useDefaults := len(required) == 0 && len(warn) == 0
	if useDefaults {
		warn = append([]string{}, defaultWarnFields...)
	}

	issue, err := client.GetIssue(ctx, issueKey)
	if err != nil {
		return err
	}

	fields, err := cache.GetFieldsCacheFirst(ctx, client)
	if err != nil {
		return err
	}

	populated := make(map[string]api.IssueFieldEntry)
	for _, e := range api.ExtractIssueFieldValues(issue, fields) {
		populated[e.ID] = e
	}

	results, missingRequired := buildCheckResults(required, warn, fields, populated, useDefaults)

	if err := emitCheckResults(opts, results); err != nil {
		return err
	}

	if missingRequired > 0 {
		return fmt.Errorf("%s: %d required field(s) missing", issueKey, missingRequired)
	}
	return nil
}

// buildCheckResults resolves every requested field name to a field ID and
// records OK/MISSING. When useDefaults is true, fields that don't resolve to a
// known schema entry are silently dropped (so a non-Agile project doesn't get
// "Sprint MISSING"). When the user named the fields explicitly, unresolved
// names are surfaced as MISSING — typos shouldn't pass silently.
func buildCheckResults(required, warn []string, fields []api.Field, populated map[string]api.IssueFieldEntry, useDefaults bool) ([]checkResult, int) {
	resolver := newFieldResolver(fields)
	var results []checkResult
	missingRequired := 0

	evaluateField := func(requested string, isRequired bool) {
		level := levelWarn
		if isRequired {
			level = levelRequired
		}
		id, display, ok := resolver.resolve(requested)
		if !ok {
			if useDefaults {
				return
			}
			results = append(results, checkResult{
				requested: requested,
				resolved:  "",
				display:   requested,
				level:     level,
				status:    statusMissing,
				value:     "(unknown field on this instance)",
			})
			if isRequired {
				missingRequired++
			}
			return
		}
		entry, populatedHit := populated[id]
		if populatedHit {
			results = append(results, checkResult{
				requested: requested,
				resolved:  id,
				display:   display,
				level:     level,
				status:    statusOK,
				value:     entry.Value,
			})
			return
		}
		results = append(results, checkResult{
			requested: requested,
			resolved:  id,
			display:   display,
			level:     level,
			status:    statusMissing,
			value:     "",
		})
		if isRequired {
			missingRequired++
		}
	}

	for _, f := range required {
		evaluateField(f, true)
	}
	for _, f := range warn {
		evaluateField(f, false)
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].level != results[j].level {
			return results[i].level == levelRequired
		}
		return results[i].display < results[j].display
	})

	return results, missingRequired
}

func emitCheckResults(opts *root.Options, results []checkResult) error {
	if opts.EmitIDOnly() {
		var ids []string
		for _, r := range results {
			if r.status == statusMissing {
				ids = append(ids, idForEmit(r))
			}
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	rows := make([]present.Row, len(results))
	for i, r := range results {
		rows[i] = present.Row{Cells: []string{r.display, r.level, r.status, r.value}}
	}

	model := &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"FIELD", "LEVEL", "STATUS", "VALUE"},
				Rows:    rows,
			},
		},
	}

	return jtkpresent.Emit(opts, model)
}

func idForEmit(r checkResult) string {
	if r.resolved != "" {
		return r.resolved
	}
	return r.requested
}

// fieldResolver maps user-supplied field references (display name, field ID,
// or property key like "assignee") to the canonical field ID + display name.
type fieldResolver struct {
	byID   map[string]api.Field
	byName map[string]api.Field // lower-cased name → field
}

func newFieldResolver(fields []api.Field) *fieldResolver {
	r := &fieldResolver{
		byID:   make(map[string]api.Field, len(fields)),
		byName: make(map[string]api.Field, len(fields)),
	}
	for _, f := range fields {
		r.byID[f.ID] = f
		r.byName[strings.ToLower(f.Name)] = f
	}
	return r
}

func (r *fieldResolver) resolve(input string) (id string, display string, ok bool) {
	if f, hit := r.byID[input]; hit {
		return f.ID, f.Name, true
	}
	if f, hit := r.byName[strings.ToLower(input)]; hit {
		return f.ID, f.Name, true
	}
	// Fall back to the literal input as both ID and display, so users can
	// pass an arbitrary customfield_NNNNN even if it's not in the cached
	// field metadata. The ExtractIssueFieldValues map will still answer
	// hit/miss correctly.
	if strings.HasPrefix(input, "customfield_") {
		return input, input, true
	}
	return "", "", false
}
