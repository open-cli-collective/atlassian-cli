package present

import (
	"fmt"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// Emit renders a presentation model and writes the split streams to the root
// writers using cfl's root-derived render style.
func Emit(opts *root.Options, model *sharedpresent.OutputModel) error {
	out := sharedpresent.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
