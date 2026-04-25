package issues

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
)

func newMoveCmd(opts *root.Options) *cobra.Command {
	var targetProject string
	var targetType string
	var notify bool
	var wait bool

	cmd := &cobra.Command{
		Use:   "move <issue-key>...",
		Short: "Move issues to another project (Cloud only)",
		Long: `Move one or more issues to a different project and/or issue type.

This command uses the Jira Cloud bulk move API and is not available
on Jira Server or Data Center.

The operation is asynchronous - by default it waits for completion.
Use --no-wait to return immediately with the task ID.

Limitations:
- Maximum 1000 issues per request
- Subtasks must be moved with their parent or separately
- Some field values may need to be remapped manually`,
		Example: `  # --to-project accepts a key or name; --to-type accepts a type name
  jtk issues move PROJ-123 --to-project NEWPROJ
  jtk issues move PROJ-123 --to-project "Platform Development" --to-type Task

  # Move multiple issues
  jtk issues move PROJ-123 PROJ-124 PROJ-125 --to-project NEWPROJ

  # Move without waiting for completion
  jtk issues move PROJ-123 --to-project NEWPROJ --no-wait

  # Move without notifications
  jtk issues move PROJ-123 --to-project NEWPROJ --no-notify`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMove(cmd.Context(), opts, args, targetProject, targetType, notify, wait)
		},
	}

	cmd.Flags().StringVar(&targetProject, "to-project", "", "Target project key or name (required)")
	cmd.Flags().StringVar(&targetType, "to-type", "", "Target issue type name (default: same as source, resolved via cache)")
	cmd.Flags().BoolVar(&notify, "notify", true, "Send notifications for the move")
	cmd.Flags().BoolVar(&wait, "wait", true, "Wait for the move to complete")

	_ = cmd.MarkFlagRequired("to-project")

	return cmd
}

func runMove(ctx context.Context, opts *root.Options, issueKeys []string, targetProject, targetType string, notify, wait bool) error {
	ip := jtkpresent.IssuePresenter{}

	if len(issueKeys) > 1000 {
		return fmt.Errorf("cannot move more than 1000 issues at once (got %d)", len(issueKeys))
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	resolver := resolve.New(client)

	resolvedProject, err := resolver.Project(ctx, targetProject)
	if err != nil {
		return err
	}
	projectKey := resolvedProject.Key

	// The bulk-move API addresses targets as "PROJECT_KEY,ISSUE_TYPE_ID",
	// so we need a concrete numeric ID at request time. Both resolution
	// paths below MUST yield an IssueType with a non-empty ID; a
	// cold-cache synthetic (Name only) can't satisfy that contract.
	var targetIssueType *api.IssueType
	if targetType == "" {
		issue, err := client.GetIssue(ctx, issueKeys[0])
		if err != nil {
			return fmt.Errorf("getting source issue: %w", err)
		}
		if issue.Fields.IssueType == nil {
			return fmt.Errorf("source issue %s has no issue type", issueKeys[0])
		}
		sourceTypeName := issue.Fields.IssueType.Name
		// Cache-first with a targeted single-project live fallback on
		// cold cache. Using the cache avoids N+1 fetches during bulk
		// moves; the live fallback preserves the pre-cache O(1) cold-
		// start cost (one GetProjectIssueTypes call for the target
		// project) instead of pulling the entire multi-project
		// issuetypes envelope just to answer one lookup.
		match, fallback, derr := matchCachedIssueType(projectKey, sourceTypeName)
		if derr != nil && errors.Is(derr, errIssueTypesCacheUninitialized) {
			liveTypes, lerr := client.GetProjectIssueTypes(ctx, projectKey)
			if lerr != nil {
				return fmt.Errorf("%w (live fetch also failed: %w)", derr, lerr)
			}
			match, fallback, derr = matchIssueTypeInSlice(liveTypes, projectKey, sourceTypeName)
		}
		if derr != nil {
			return derr
		}
		if fallback && match != nil {
			_ = jtkpresent.Emit(opts, jtkpresent.IssuePresenter{}.PresentTypeFallbackWarning(sourceTypeName, projectKey, match.Name))
		}
		targetIssueType = match
	} else {
		resolved, err := resolver.IssueType(ctx, projectKey, targetType)
		if err != nil {
			return err
		}
		targetIssueType = &resolved
	}
	if targetIssueType.ID == "" {
		// The resolver's cold-start fallback yields a Name-only synthetic,
		// which would produce an invalid "PROJECT_KEY," mapping and an
		// opaque API rejection. Surface a clear, actionable error instead.
		return fmt.Errorf(
			"cannot resolve issue type ID for %q in project %s from cache — "+
				"run `jtk refresh issuetypes` (requires `jtk refresh projects` first if projects are stale)",
			targetIssueType.Name, projectKey)
	}

	// Progress message to stderr
	progressModel := ip.PresentMoveProgress(len(issueKeys), projectKey, targetIssueType.Name)
	progressOut := present.Render(progressModel, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stderr, progressOut.Stderr)

	// Build and execute the move request
	req := api.BuildMoveRequest(issueKeys, projectKey, targetIssueType.ID, notify)

	resp, err := client.MoveIssues(ctx, req)
	if err != nil {
		// Check if this is a Server/DC instance
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("move operation failed - this feature is only available on Jira Cloud")
		}
		return fmt.Errorf("initiating move: %w", err)
	}

	if !wait {
		model := ip.PresentMoveInitiated(resp.TaskID)
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
		return nil
	}

	// Wait for completion - progress to stderr
	waitModel := ip.PresentMoveWaiting()
	waitOut := present.Render(waitModel, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stderr, waitOut.Stderr)

	for {
		status, err := client.GetMoveTaskStatus(ctx, resp.TaskID)
		if err != nil {
			return fmt.Errorf("getting task status: %w", err)
		}

		switch status.Status {
		case "COMPLETE":
			if status.Result != nil && len(status.Result.Failed) > 0 {
				model := ip.PresentMovePartialFailure(status.Result.Successful, status.Result.Failed)
				out := present.Render(model, opts.RenderStyle())
				_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
				_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
				return fmt.Errorf("some issues failed to move")
			}
			model := ip.PresentMoved(len(issueKeys), projectKey)
			out := present.Render(model, opts.RenderStyle())
			_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
			return nil

		case "FAILED":
			return fmt.Errorf("move failed")

		case "CANCELLED":
			return fmt.Errorf("move was cancelled")

		case "ENQUEUED", "RUNNING":
			// Still in progress
			time.Sleep(1 * time.Second)

		default:
			return fmt.Errorf("unknown task status: %s", status.Status)
		}
	}
}

func newMoveStatusCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move-status <task-id>",
		Short: "Check status of a move operation",
		Long:  "Check the status of an asynchronous move operation by task ID.",
		Example: `  # Check move task status
  jtk issues move-status abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMoveStatus(cmd.Context(), opts, args[0])
		},
	}

	return cmd
}

func runMoveStatus(ctx context.Context, opts *root.Options, taskID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	status, err := client.GetMoveTaskStatus(ctx, taskID)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(status)
	}

	model := jtkpresent.IssuePresenter{}.PresentMoveStatus(status)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

// matchCachedIssueType looks up sourceTypeName in the target project's cached
// issue types and, failing that, returns the first non-subtask type. Cache-
// authoritative: no refresh, no live fallback. Used by `issues move` when
// --to-type is omitted.
//
// Only cache.ErrCacheMiss maps to the "cold cache" fallback case — any other
// read error (I/O, permission, corrupt envelope) propagates so the user sees
// the real problem instead of a misleading "cache missing" message. The
// refresh hint is left to the caller since the root cause differs between
// the cold-cache and the "populated but empty" paths.
// errIssueTypesCacheUninitialized signals that the issuetypes envelope
// doesn't exist yet, so a caller can attempt a one-shot refresh before
// giving up (matches the resolver.IssueType path for --to-type).
var errIssueTypesCacheUninitialized = errors.New("issuetypes cache not initialized — run `jtk refresh issuetypes`")

func matchCachedIssueType(projectKey, sourceTypeName string) (*api.IssueType, bool, error) {
	env, err := cache.ReadResource[map[string][]api.IssueType]("issuetypes")
	if err != nil {
		if errors.Is(err, cache.ErrCacheMiss) {
			return nil, false, errIssueTypesCacheUninitialized
		}
		return nil, false, fmt.Errorf("reading issuetypes cache: %w", err)
	}
	types, ok := env.Data[projectKey]
	if !ok || len(types) == 0 {
		return nil, false, fmt.Errorf("no cached issue types for project %s (run `jtk refresh issuetypes` or supply --to-type)", projectKey)
	}
	return matchIssueTypeInSlice(types, projectKey, sourceTypeName)
}

// matchIssueTypeInSlice picks a target issue type from an unordered list.
// Preferred: exact name match (case-insensitive) with the source type.
// Fallback: first non-subtask — returns usedFallback=true so the caller
// can emit a warning via the presenter.
func matchIssueTypeInSlice(types []api.IssueType, projectKey, sourceTypeName string) (*api.IssueType, bool, error) {
	for i := range types {
		if strings.EqualFold(types[i].Name, sourceTypeName) {
			return &types[i], false, nil
		}
	}
	for i := range types {
		if !types[i].Subtask {
			return &types[i], true, nil
		}
	}
	return nil, false, fmt.Errorf("no non-subtask issue types available for project %s", projectKey)
}
