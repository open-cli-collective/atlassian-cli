package present

import (
	"fmt"
	"strconv"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

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

// ValidateMax enforces the shared page-size contract for --max.
func ValidateMax(maxResults int) error {
	if maxResults <= 0 {
		return fmt.Errorf("--max must be greater than zero")
	}
	return nil
}

// ValidateMaxAtMost enforces the shared page-size contract and an endpoint ceiling.
func ValidateMaxAtMost(maxResults, ceiling int) error {
	if err := ValidateMax(maxResults); err != nil {
		return err
	}
	if maxResults > ceiling {
		return fmt.Errorf("--max must be %d or less", ceiling)
	}
	return nil
}

// paginationMessageSectionWithToken builds the spec-shaped continuation line
// that embeds the next-page token, per the JTK Output Specification (#230).
func paginationMessageSectionWithToken(token string) *present.MessageSection {
	return &present.MessageSection{
		Kind:    present.MessageInfo,
		Message: fmt.Sprintf("More results available (next: %s)", token),
		Stream:  present.StreamStderr,
	}
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

// AppendPaginationHintWithToken returns sections with a token-embedded
// pagination MessageSection appended when hasMore is true, otherwise returns
// sections unchanged.
func AppendPaginationHintWithToken(sections []present.Section, hasMore bool, token string) []present.Section {
	if !hasMore {
		return sections
	}
	return append(sections, paginationMessageSectionWithToken(token))
}

// EmitIDsWithPaginationToken is EmitIDs plus a token-embedded continuation
// line on stderr when hasMore is true. Shares paginationMessageSectionWithToken
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
