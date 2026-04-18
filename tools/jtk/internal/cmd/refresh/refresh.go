// Package refresh provides the `jtk refresh` command for updating the instance cache.
package refresh

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

// Register registers the refresh command.
func Register(parent *cobra.Command, opts *root.Options) {
	var statusOnly bool

	cmd := &cobra.Command{
		Use:   "refresh [resources...]",
		Short: "Refresh the jtk instance cache",
		Long: `Refresh the jtk instance cache — the local snapshot of fields, projects,
users, issue types, statuses, priorities, resolutions, boards, and link types.

With no arguments, refreshes every cacheable resource. With resource names,
refreshes only those. With --status, reports freshness without fetching.

Requires configuration (jtk init). Some resources (boards) are unavailable
under bearer authentication and are skipped.`,
		Example: `  # Refresh everything
  jtk refresh

  # Refresh a subset
  jtk refresh fields users

  # Show freshness without fetching
  jtk refresh --status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), opts, args, statusOnly)
		},
	}

	cmd.Flags().BoolVar(&statusOnly, "status", false, "Print cache freshness; no network calls")
	parent.AddCommand(cmd)
}

// run is the command entry point. Exported for testing would require package-level
// plumbing; keeping it unexported and covered via cobra.Command.Execute in tests.
func run(ctx context.Context, opts *root.Options, names []string, statusOnly bool) error {
	if !config.IsConfigured() {
		return errors.New("configuration not found — run 'jtk init' first")
	}

	selected, err := selectEntries(names)
	if err != nil {
		return err
	}

	if statusOnly {
		return runStatus(opts, selected)
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}
	return runRefresh(ctx, opts, client, selected)
}

// selectEntries resolves the named-resources argument list (empty = all entries
// in dependency order).
func selectEntries(names []string) ([]cache.Entry, error) {
	all := cache.Entries()
	if len(names) == 0 {
		return all, nil
	}

	byName := make(map[string]cache.Entry, len(all))
	for _, e := range all {
		byName[e.Name] = e
	}

	seen := make(map[string]bool, len(names))
	selected := make([]cache.Entry, 0, len(names))
	for _, n := range names {
		if seen[n] {
			continue
		}
		e, ok := byName[n]
		if !ok {
			return nil, fmt.Errorf("unknown resource %q — valid names: %v", n, cache.Names())
		}
		selected = append(selected, e)
		seen[n] = true
	}
	return selected, nil
}

// runStatus renders the freshness table.
func runStatus(opts *root.Options, selected []cache.Entry) error {
	client, _ := opts.APIClient() // best-effort; availability check tolerates nil

	now := time.Now().UTC()
	rows := make([]present.Row, 0, len(selected))
	for _, e := range selected {
		rows = append(rows, present.Row{Cells: statusRow(e, client, now)})
	}

	model := &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"RESOURCE", "FETCHED_AT", "AGE", "TTL", "STATUS"},
				Rows:    rows,
			},
		},
	}
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

// statusRow is the per-entry row in the --status table.
func statusRow(e cache.Entry, client *api.Client, now time.Time) []string {
	if !e.IsAvailable(client) {
		return []string{e.Name, "-", "-", e.TTL, cache.StatusUnavailable.String()}
	}

	env, err := cache.ReadResource[any](e.Name)
	if errors.Is(err, cache.ErrCacheMiss) {
		return []string{e.Name, "-", "-", e.TTL, cache.StatusUninitialized.String()}
	}
	if err != nil {
		return []string{e.Name, "-", "-", e.TTL, "error"}
	}

	fetchedAt := "-"
	if !env.FetchedAt.IsZero() {
		fetchedAt = env.FetchedAt.UTC().Format(time.RFC3339)
	}
	status := cache.Classify(env.FetchedAt, e.TTL, now)
	return []string{e.Name, fetchedAt, cache.Age(env.FetchedAt, now), e.TTL, status.String()}
}

// runRefresh fetches each selected entry and renders per-resource result lines.
// Continues on error; overall exit non-zero if any resource failed.
func runRefresh(ctx context.Context, opts *root.Options, client *api.Client, selected []cache.Entry) error {
	sections := make([]present.Section, 0, len(selected))
	var firstFailure error

	for _, e := range selected {
		if !e.IsAvailable(client) {
			sections = append(sections, &present.MessageSection{
				Kind:    present.MessageInfo,
				Stream:  present.StreamStdout,
				Message: fmt.Sprintf("Skipping %s — unavailable under current auth", e.Name),
			})
			continue
		}

		prev := previousCount(e.Name)
		count, err := e.Fetch(ctx, client)
		if err != nil {
			if firstFailure == nil {
				firstFailure = err
			}
			sections = append(sections, &present.MessageSection{
				Kind:    present.MessageError,
				Stream:  present.StreamStderr,
				Message: fmt.Sprintf("Refreshing %s failed: %v", e.Name, err),
			})
			continue
		}

		sections = append(sections, &present.MessageSection{
			Kind:    present.MessageSuccess,
			Stream:  present.StreamStdout,
			Message: formatRefreshLine(e.Name, count, prev, time.Now().UTC()),
		})
	}

	model := &present.OutputModel{Sections: sections}
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return firstFailure
}

// previousCount reports the prior entry count for a resource, or -1 if unknown.
// Uses a shallow decode just to count slice/map entries — good enough for a
// delta indicator.
func previousCount(name string) int {
	env, err := cache.ReadResource[any](name)
	if err != nil {
		return -1
	}
	switch v := env.Data.(type) {
	case []any:
		return len(v)
	case map[string]any:
		total := 0
		for _, inner := range v {
			if arr, ok := inner.([]any); ok {
				total += len(arr)
			}
		}
		return total
	default:
		return -1
	}
}

// formatRefreshLine produces the per-resource success message.
// "Refreshing fields... 73 entries (was 72) — Cache updated at 2026-04-18 14:23:00"
func formatRefreshLine(name string, count, prev int, now time.Time) string {
	delta := ""
	if prev >= 0 && prev != count {
		delta = fmt.Sprintf(" (was %d)", prev)
	}
	return fmt.Sprintf("Refreshing %s... %d entries%s — Cache updated at %s",
		name, count, delta, now.Format("2006-01-02 15:04:05"))
}
