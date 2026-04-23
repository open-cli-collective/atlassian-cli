package present

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// FieldPresenter creates presentation models for field data.
type FieldPresenter struct{}

// PresentList creates a table view for a list of fields. Default: ID|TYPE|NAME.
// Extended adds SEARCHABLE, NAVIGABLE, ORDERABLE, CLAUSE_NAMES per #230.
func (FieldPresenter) PresentList(fields []api.Field, extended bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"ID", "NAME", "TYPE", "CUSTOM", "SEARCHABLE", "NAVIGABLE", "ORDERABLE", "CLAUSE_NAMES"}
	} else {
		headers = []string{"ID", "NAME", "TYPE", "CUSTOM"}
	}

	rows := make([]present.Row, len(fields))
	for i, f := range fields {
		custom := "no"
		if f.Custom {
			custom = "yes"
		}
		if extended {
			clauseNames := "-"
			if len(f.ClauseNames) > 0 {
				clauseNames = strings.Join(f.ClauseNames, ", ")
			}
			rows[i] = present.Row{
				Cells: []string{
					f.ID,
					f.Name,
					OrDash(f.Schema.Type),
					custom,
					BoolString(f.Searchable),
					BoolString(f.Navigable),
					BoolString(f.Orderable),
					clauseNames,
				},
			}
		} else {
			rows[i] = present.Row{
				Cells: []string{f.ID, f.Name, OrDash(f.Schema.Type), custom},
			}
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: rows},
		},
	}
}

// PresentEditableFields creates a table view for editable fields.
func (FieldPresenter) PresentEditableFields(fields []api.EditFieldMeta) *present.OutputModel {
	rows := make([]present.Row, len(fields))
	for i, f := range fields {
		required := "no"
		if f.Required {
			required = "yes"
		}
		rows[i] = present.Row{
			Cells: []string{f.ID, f.Name, f.Type, required},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "TYPE", "REQUIRED"},
				Rows:    rows,
			},
		},
	}
}

// PresentFieldOptions creates a table view for field options.
func (FieldPresenter) PresentFieldOptions(options []api.FieldOptionValue) *present.OutputModel {
	rows := make([]present.Row, len(options))
	for i, opt := range options {
		value := opt.Value
		if value == "" {
			value = opt.Name
		}
		if opt.Disabled {
			value = value + " (disabled)"
		}
		rows[i] = present.Row{
			Cells: []string{opt.ID, value},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "VALUE"},
				Rows:    rows,
			},
		},
	}
}

// PresentContexts creates a table view for field contexts.
func (FieldPresenter) PresentContexts(contexts []api.FieldContext) *present.OutputModel {
	rows := make([]present.Row, len(contexts))
	for i, ctx := range contexts {
		global := "no"
		if ctx.IsGlobalContext {
			global = "yes"
		}
		anyIssueType := "no"
		if ctx.IsAnyIssueType {
			anyIssueType = "yes"
		}
		rows[i] = present.Row{
			Cells: []string{ctx.ID, ctx.Name, global, anyIssueType},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "GLOBAL", "ANY_ISSUE_TYPE"},
				Rows:    rows,
			},
		},
	}
}

// PresentContextOptions creates a table view for field context options.
func (FieldPresenter) PresentContextOptions(options []api.FieldContextOption) *present.OutputModel {
	rows := make([]present.Row, len(options))
	for i, opt := range options {
		disabled := "no"
		if opt.Disabled {
			disabled = "yes"
		}
		rows[i] = present.Row{
			Cells: []string{opt.ID, opt.Value, disabled},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "VALUE", "DISABLED"},
				Rows:    rows,
			},
		},
	}
}

// PresentCreated creates a success message for field creation.
func (FieldPresenter) PresentCreated(id, name string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Created field %s (%s)", id, name),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentTrashed creates a success message for field trashing.
func (FieldPresenter) PresentTrashed(fieldID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Trashed field %s", fieldID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentRestored creates a success message for field restoration.
func (FieldPresenter) PresentRestored(fieldID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Restored field %s", fieldID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentEmpty creates an info message when no fields are found.
func (FieldPresenter) PresentEmpty() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No fields found",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentDeleteCancelled creates an info message for cancelled field deletion.
func (FieldPresenter) PresentDeleteCancelled() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "Deletion cancelled.",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentNoContexts creates an info message when no contexts are found.
func (FieldPresenter) PresentNoContexts(fieldID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No contexts found for field %s", fieldID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentContextCreated creates a success message for context creation.
func (FieldPresenter) PresentContextCreated(id, name string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Created context %s (%s)", id, name),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentContextDeleted creates a success message for context deletion.
func (FieldPresenter) PresentContextDeleted(contextID, fieldID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Deleted context %s from field %s", contextID, fieldID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentNoOptions creates an info message when no options are found.
func (FieldPresenter) PresentNoOptions(fieldID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No options found for field %s", fieldID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentOptionAdded creates a success message for option addition.
// If optionID is empty, only the value is shown.
func (FieldPresenter) PresentOptionAdded(optionID, value string) *present.OutputModel {
	msg := fmt.Sprintf("Added option %s", value)
	if optionID != "" {
		msg = fmt.Sprintf("Added option %s (%s)", optionID, value)
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: msg,
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentOptionUpdated creates a success message for option update.
func (FieldPresenter) PresentOptionUpdated(optionID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Updated option %s", optionID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentOptionDeleted creates a success message for option deletion.
func (FieldPresenter) PresentOptionDeleted(optionID, fieldID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Deleted option %s from field %s", optionID, fieldID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// --- Field options with header ---

// PresentOptionsNoContext creates a warning about missing issue context.
func (FieldPresenter) PresentOptionsNoContext() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageWarning,
				Message: "Could not get field options without issue context. Use --issue flag for better results.",
				Stream:  present.StreamStderr,
			},
		},
	}
}

// PresentFieldOptionsWithHeader creates a header + table for field options.
func (FieldPresenter) PresentFieldOptionsWithHeader(fieldName string, options []api.FieldOptionValue) *present.OutputModel {
	rows := make([]present.Row, len(options))
	for i, opt := range options {
		value := opt.Value
		if value == "" {
			value = opt.Name
		}
		if opt.Disabled {
			value = value + " (disabled)"
		}
		rows[i] = present.Row{
			Cells: []string{opt.ID, value},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("Allowed values for field '%s':", fieldName),
				Stream:  present.StreamStdout,
			},
			&present.TableSection{
				Headers: []string{"ID", "VALUE"},
				Rows:    rows,
			},
		},
	}
}
