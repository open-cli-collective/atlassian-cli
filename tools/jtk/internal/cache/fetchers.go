package cache

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// warnWriter is where fetchers emit partial-success / truncation warnings.
// Swapped by tests to capture warnings without touching the process stderr.
// Callers outside tests can override via SetWarnWriter; the default mirrors
// the old behavior of writing to os.Stderr. Access is guarded by warnMu so
// tests that swap the writer while a fetcher is running don't race.
var (
	warnMu     sync.RWMutex
	warnWriter io.Writer = os.Stderr
)

func getWarnWriter() io.Writer {
	warnMu.RLock()
	defer warnMu.RUnlock()
	return warnWriter
}

// SetWarnWriter redirects fetcher warning output. Intended for tests that
// want to assert on the warning text. Returns a restore function.
func SetWarnWriter(w io.Writer) func() {
	warnMu.Lock()
	old := warnWriter
	warnWriter = w
	warnMu.Unlock()
	return func() {
		warnMu.Lock()
		warnWriter = old
		warnMu.Unlock()
	}
}

const (
	ttl24h  = "24h"
	ttl168h = "168h"
)

// init populates the default registry.
func init() {
	entries = defaultEntries()
}

// defaultEntries returns the production registry in declaration order.
// Entries() applies dependency ordering on top of this.
func defaultEntries() []Entry {
	return []Entry{
		{Name: "fields", TTL: ttl24h, Fetch: fetchFields},
		{Name: "projects", TTL: ttl24h, Fetch: fetchProjects},
		{Name: "boards", TTL: ttl24h, Available: supportsAgile, Fetch: fetchBoards},
		{Name: "sprints", TTL: ttl24h, DependsOn: []string{"boards"}, Available: supportsAgile, Fetch: fetchSprints},
		{Name: "linktypes", TTL: ttl24h, Fetch: fetchLinkTypes},
		{Name: "issuetypes", TTL: ttl24h, DependsOn: []string{"projects"}, Fetch: fetchIssueTypes},
		{Name: "statuses", TTL: ttl24h, DependsOn: []string{"projects"}, Fetch: fetchStatuses},
		{Name: "priorities", TTL: ttl168h, Fetch: fetchPriorities},
		{Name: "resolutions", TTL: ttl168h, Fetch: fetchResolutions},
		{Name: "users", TTL: ttl24h, Fetch: fetchUsers},
	}
}

// supportsAgile reports whether the client can reach the Agile REST API.
// Scoped bearer tokens lack the Agile scope, so boards (and any future
// Agile-gated resource) must be skipped in that mode.
func supportsAgile(c *api.Client) bool {
	return c != nil && c.SupportsAgile()
}

func fetchFields(ctx context.Context, c *api.Client) (int, error) {
	fields, err := c.GetFields(ctx)
	if err != nil {
		return 0, err
	}
	if err := WriteResource("fields", ttl24h, fields); err != nil {
		return 0, err
	}
	return len(fields), nil
}

func fetchProjects(ctx context.Context, c *api.Client) (int, error) {
	projects, err := c.ListProjects(ctx)
	if err != nil {
		return 0, err
	}
	if err := WriteResource("projects", ttl24h, projects); err != nil {
		return 0, err
	}
	return len(projects), nil
}

// fetchBoardsMax is a safety ceiling. A misbehaving server that never sets
// isLast=true would otherwise cause unbounded pagination.
const fetchBoardsMax = 5000

func fetchBoards(ctx context.Context, c *api.Client) (int, error) {
	const pageSize = 50
	var all []api.Board
	startAt := 0
	for startAt < fetchBoardsMax {
		resp, err := c.ListBoards(ctx, "", startAt, pageSize)
		if err != nil {
			return 0, err
		}
		all = append(all, resp.Values...)
		if resp.IsLast || len(resp.Values) == 0 {
			break
		}
		startAt += len(resp.Values)
	}
	if err := WriteResource("boards", ttl24h, all); err != nil {
		return 0, err
	}
	return len(all), nil
}

func fetchLinkTypes(_ context.Context, c *api.Client) (int, error) {
	types, err := c.GetIssueLinkTypes()
	if err != nil {
		return 0, err
	}
	if err := WriteResource("linktypes", ttl24h, types); err != nil {
		return 0, err
	}
	return len(types), nil
}

func fetchIssueTypes(ctx context.Context, c *api.Client) (int, error) {
	env, err := ReadResource[[]api.Project]("projects")
	if err != nil {
		return 0, fmt.Errorf("reading projects cache (refresh projects first): %w", err)
	}

	byProject := make(map[string][]api.IssueType, len(env.Data))
	total := 0
	for _, p := range env.Data {
		types, err := c.GetProjectIssueTypes(ctx, p.Key)
		if err != nil {
			return 0, fmt.Errorf("fetching issue types for %s: %w", p.Key, err)
		}
		byProject[p.Key] = types
		total += len(types)
	}
	if err := WriteResource("issuetypes", ttl24h, byProject); err != nil {
		return 0, err
	}
	return total, nil
}

func fetchStatuses(ctx context.Context, c *api.Client) (int, error) {
	env, err := ReadResource[[]api.Project]("projects")
	if err != nil {
		return 0, fmt.Errorf("reading projects cache (refresh projects first): %w", err)
	}

	byProject := make(map[string][]api.ProjectStatus, len(env.Data))
	total := 0
	for _, p := range env.Data {
		statuses, err := c.GetProjectStatuses(ctx, p.Key)
		if err != nil {
			return 0, fmt.Errorf("fetching statuses for %s: %w", p.Key, err)
		}
		byProject[p.Key] = statuses
		total += len(statuses)
	}
	if err := WriteResource("statuses", ttl24h, byProject); err != nil {
		return 0, err
	}
	return total, nil
}

func fetchPriorities(ctx context.Context, c *api.Client) (int, error) {
	priorities, err := c.ListPriorities(ctx)
	if err != nil {
		return 0, err
	}
	if err := WriteResource("priorities", ttl168h, priorities); err != nil {
		return 0, err
	}
	return len(priorities), nil
}

func fetchResolutions(ctx context.Context, c *api.Client) (int, error) {
	resolutions, err := c.ListResolutions(ctx)
	if err != nil {
		return 0, err
	}
	if err := WriteResource("resolutions", ttl168h, resolutions); err != nil {
		return 0, err
	}
	return len(resolutions), nil
}

func fetchUsers(ctx context.Context, c *api.Client) (int, error) {
	return SeedUsers(ctx, c)
}

// fetchSprintsMax is the per-board iteration ceiling. A misbehaving server that
// never returns isLast=true would otherwise spin forever for a single board.
const fetchSprintsMax = 5000

// fetchSprints assembles a board-keyed sprint map. It reads the boards cache
// (populated by fetchBoards via the DependsOn edge), then pages ListSprints
// per board.
//
// Partial-success strategy: on instances with many boards, a single
// permission-denied or stale board shouldn't permanently block the whole
// sprints refresh. Per-board errors are logged (stderr) and the board is
// skipped; the envelope still gets written with the boards that succeeded.
// If *every* board errors, the fetch fails with a combined error so the
// refresh command reports it cleanly.
//
// If the fetcher hits the iteration ceiling mid-board, a warning is logged
// and the partial sprint list for that board is retained — downstream
// resolvers can still match what was cached.
func fetchSprints(ctx context.Context, c *api.Client) (int, error) {
	env, err := ReadResource[[]api.Board]("boards")
	if err != nil {
		return 0, fmt.Errorf("reading boards cache (refresh boards first): %w", err)
	}

	const pageSize = 50
	byBoard := make(map[int][]api.Sprint, len(env.Data))
	total := 0
	var errs []string
	for _, b := range env.Data {
		var all []api.Sprint
		startAt := 0
		finishedNaturally := false
		var boardErr error
		for startAt < fetchSprintsMax {
			resp, err := c.ListSprints(ctx, b.ID, "", startAt, pageSize)
			if err != nil {
				boardErr = err
				break
			}
			all = append(all, resp.Values...)
			if resp.IsLast || len(resp.Values) == 0 {
				finishedNaturally = true
				break
			}
			startAt += len(resp.Values)
		}
		if boardErr != nil {
			fmt.Fprintf(getWarnWriter(), "warning: sprints refresh for board %d failed, skipping: %v\n", b.ID, boardErr)
			errs = append(errs, fmt.Sprintf("board %d: %v", b.ID, boardErr))
			continue
		}
		// Only warn if the ceiling is why we stopped — i.e., the loop exited
		// without the API marking the page IsLast. A board with exactly
		// fetchSprintsMax sprints where Jira returns IsLast=true on the final
		// page hits `finishedNaturally` and gets no warning.
		if !finishedNaturally && startAt >= fetchSprintsMax {
			fmt.Fprintf(getWarnWriter(), "warning: sprints for board %d reached the %d-entry ceiling — further pages skipped (cached sprints are retained)\n", b.ID, fetchSprintsMax)
		}
		byBoard[b.ID] = all
		total += len(all)
	}
	// Fail the whole fetch only when no board succeeded — otherwise the
	// partial map is more useful than nothing.
	if len(byBoard) == 0 && len(errs) > 0 {
		return 0, fmt.Errorf("fetching sprints: all boards failed (%s)", strings.Join(errs, "; "))
	}
	if err := WriteResource("sprints", ttl24h, byBoard); err != nil {
		return 0, err
	}
	return total, nil
}
