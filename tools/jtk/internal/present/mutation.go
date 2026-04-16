// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/present"
)

// MutationPresenter creates presentation models for mutation confirmations.
type MutationPresenter struct{}

// Success creates a success message that goes to stdout (it IS the result).
func (MutationPresenter) Success(format string, args ...any) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf(format, args...),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// Info creates an informational message that goes to stdout.
func (MutationPresenter) Info(format string, args ...any) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf(format, args...),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// Advisory creates a non-primary message that goes to stderr (for genuine
// diagnostics that should not mix with primary output).
func (MutationPresenter) Advisory(format string, args ...any) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf(format, args...),
				Stream:  present.StreamStderr,
			},
		},
	}
}

// Warning creates a warning message that goes to stderr.
func (MutationPresenter) Warning(format string, args ...any) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageWarning,
				Message: fmt.Sprintf(format, args...),
				Stream:  present.StreamStderr,
			},
		},
	}
}

// Error creates an error message that goes to stderr.
func (MutationPresenter) Error(format string, args ...any) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageError,
				Message: fmt.Sprintf(format, args...),
				Stream:  present.StreamStderr,
			},
		},
	}
}
