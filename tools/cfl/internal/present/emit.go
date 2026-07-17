package present

import (
	"errors"
	"fmt"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// ErrEmitted marks an error whose user-facing text was already written.
var ErrEmitted = errors.New("error already emitted")

type emittedError struct{ err error }

func (e emittedError) Error() string        { return e.err.Error() }
func (e emittedError) Unwrap() error        { return e.err }
func (e emittedError) Is(target error) bool { return target == ErrEmitted }

// Emit renders a presentation model and writes the split streams to the root
// writers using cfl's root-derived render style.
func Emit(opts *root.Options, model *sharedpresent.OutputModel) error {
	out := sharedpresent.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

// EmitError writes an error through the presenter stream model and marks it so
// the process entrypoint does not print it a second time.
func EmitError(opts *root.Options, err error) error {
	_ = Emit(opts, &sharedpresent.OutputModel{Sections: []sharedpresent.Section{stderrInfo(err.Error())}})
	return emittedError{err: err}
}
