package present

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// paginationHint is the text appended after list output when more pages exist.
// Kept centralized so default-mode and --id mode share the same wording.
const paginationHint = "More results available (use --next-page-token to fetch next page)"

// Emit applies jtk output policy: renders the model and writes the split
// streams to opts.Stdout / opts.Stderr. Returns nil so commands can
// `return Emit(...)` at the end of RunE.
func Emit(opts *root.Options, model *present.OutputModel) error {
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

// EmitIDs writes one identifier per line to opts.Stdout. Empty input emits
// nothing. Matches `kubectl get -o name` / `ls -1` semantics.
func EmitIDs(opts *root.Options, ids ...string) error {
	for _, id := range ids {
		_, _ = fmt.Fprintln(opts.Stdout, id)
	}
	return nil
}

// EmitIDsWithPagination is EmitIDs plus a continuation line on stdout when
// hasMore is true. The continuation line matches the default-mode pagination
// policy so `--id` and default read from the same stream.
func EmitIDsWithPagination(opts *root.Options, ids []string, hasMore bool) error {
	if err := EmitIDs(opts, ids...); err != nil {
		return err
	}
	if hasMore {
		_, _ = fmt.Fprintln(opts.Stdout, paginationHint)
	}
	return nil
}
