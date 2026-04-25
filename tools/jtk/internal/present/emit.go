package present

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// ErrFieldsWithJSON is the canonical error for `--fields` + `-o json`.
// JSON artifacts are not projected; the two flags never combine. Previously
// duplicated across cmd/issues, cmd/comments, cmd/users, and cmd/projects —
// centralizing here stops the copies from drifting in wording.
var ErrFieldsWithJSON = errors.New("--fields is not supported with --output json")

// ParseStartAtToken converts a `--next-page-token` value (a decimal offset)
// to a 0-based startAt. Empty input returns 0. Non-numeric or negative
// values return an error that names the flag, so the user sees the same
// message regardless of which migrated command they invoked.
func ParseStartAtToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(token)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid --next-page-token %q: expected a non-negative decimal", token)
	}
	return n, nil
}

// paginationHint is the legacy continuation-line wording retained for
// commands that have not yet migrated to the token-embedding variant.
// New (#237+) call sites use the *WithToken helpers below.
const paginationHint = "More results available (use --next-page-token to fetch next page)"

// paginationMessageSection builds the legacy stdout-routed continuation line.
// Migrated callers use paginationMessageSectionWithToken instead.
func paginationMessageSection() *present.MessageSection {
	return &present.MessageSection{
		Kind:    present.MessageInfo,
		Message: paginationHint,
		Stream:  present.StreamStdout,
	}
}

// paginationMessageSectionWithToken builds the spec-shaped continuation line
// that embeds the next-page token, per the JTK Output Specification (#230).
// When token is empty the helper falls back to the legacy wording so callers
// don't accidentally surface an empty "next: " fragment.
func paginationMessageSectionWithToken(token string) *present.MessageSection {
	if token == "" {
		return paginationMessageSection()
	}
	return &present.MessageSection{
		Kind:    present.MessageInfo,
		Message: fmt.Sprintf("More results available (next: %s)", token),
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

// PaginationOnlyModel creates an OutputModel containing only a pagination
// hint. Used when a paginated query returns zero results for the current
// page but more pages exist.
func PaginationOnlyModel(nextToken string) *present.OutputModel {
	return &present.OutputModel{
		Sections: AppendPaginationHintWithToken(nil, true, nextToken),
	}
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

// AppendPaginationHintWithToken returns sections with a token-embedded
// pagination MessageSection appended when hasMore is true, otherwise returns
// sections unchanged. Follow-up to #230: new migrated list commands emit
// "More results available (next: <token>)" instead of the legacy wording.
// Empty token degrades to the legacy phrasing.
func AppendPaginationHintWithToken(sections []present.Section, hasMore bool, token string) []present.Section {
	if !hasMore {
		return sections
	}
	return append(sections, paginationMessageSectionWithToken(token))
}

// EmitIDsWithPaginationToken is EmitIDs plus a token-embedded continuation
// line on stdout when hasMore is true. Shares paginationMessageSectionWithToken
// with AppendPaginationHintWithToken so `--id` and default mode stay aligned.
func EmitIDsWithPaginationToken(opts *root.Options, ids []string, hasMore bool, token string) error {
	if err := EmitIDs(opts, ids); err != nil {
		return err
	}
	if hasMore {
		model := &present.OutputModel{Sections: []present.Section{paginationMessageSectionWithToken(token)}}
		return Emit(opts, model)
	}
	return nil
}
