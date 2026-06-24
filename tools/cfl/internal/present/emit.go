package present

import (
	"fmt"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// Emit renders a presentation model and writes the split streams to the root
// writers. This is intentionally narrow prework for #271: it is safe today for
// detail/message shapes whose table/plain behavior is semantically identical.
// List/search migrations that must preserve `-o plain` TSV semantics should stay
// on legacy view helpers until cfl has a presenter-backed TSV decision.
func Emit(opts *root.Options, model *sharedpresent.OutputModel) error {
	out := sharedpresent.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
