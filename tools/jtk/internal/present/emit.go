package present

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// paginationHint is the text appended after list output when more pages exist.
// Kept centralized so default-mode and --id mode share the same wording.
const paginationHint = "More results available (use --next-page-token to fetch next page)"

// paginationMessageSection builds the stdout-routed continuation-line section
// used by every list presenter. Centralizing this ensures the wording, kind,
// and stream stay in sync across all callers.
func paginationMessageSection() *present.MessageSection {
	return &present.MessageSection{
		Kind:    present.MessageInfo,
		Message: paginationHint,
		Stream:  present.StreamStdout,
	}
}

// AppendPaginationHint returns sections with a pagination MessageSection
// appended when hasMore is true, otherwise returns sections unchanged.
// Every model-building pagination call site funnels through this so
// wording, kind, and stream stay in sync across presenters and commands.
//
// Follows Go's standard append semantics: the returned slice may share
// its backing array with the input. Callers that pass a slice with spare
// capacity beyond its length should treat the input as consumed, or
// allocate a fresh slice before calling.
func AppendPaginationHint(sections []present.Section, hasMore bool) []present.Section {
	if !hasMore {
		return sections
	}
	return append(sections, paginationMessageSection())
}

// Emit applies jtk output policy: renders the model and writes the split
// streams to opts.Stdout / opts.Stderr. Returns nil so commands can
// `return Emit(...)` at the end of RunE.
func Emit(opts *root.Options, model *present.OutputModel) error {
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

// EmitIDs writes one identifier per line to opts.Stdout. Empty slice emits
// nothing. Matches `kubectl get -o name` / `ls -1` semantics.
func EmitIDs(opts *root.Options, ids []string) error {
	for _, id := range ids {
		_, _ = fmt.Fprintln(opts.Stdout, id)
	}
	return nil
}

// EmitIDsWithPagination is EmitIDs plus a continuation line on stdout when
// hasMore is true. The continuation line shares construction with the
// model-building presenters via paginationMessageSection() so `--id` and
// default mode can never drift on wording or stream.
func EmitIDsWithPagination(opts *root.Options, ids []string, hasMore bool) error {
	if err := EmitIDs(opts, ids); err != nil {
		return err
	}
	if hasMore {
		model := &present.OutputModel{Sections: []present.Section{paginationMessageSection()}}
		return Emit(opts, model)
	}
	return nil
}
