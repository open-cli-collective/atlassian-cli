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
// Extended: ID|TYPE|SEARCHABLE|NAVIGABLE|ORDERABLE|CLAUSE_NAMES|NAME per #230.
func (FieldPresenter) PresentList(fields []api.Field, extended bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"ID", "TYPE", "SEARCHABLE", "NAVIGABLE", "ORDERABLE", "CLAUSE_NAMES", "NAME"}
	} else {
		headers = []string{"ID", "TYPE", "NAME"}
	}

	rows := make([]present.Row, len(fields))
	for i, f := range fields {
		if extended {
			clauseNames := "-"
			if len(f.ClauseNames) > 0 {
				clauseNames = strings.Join(f.ClauseNames, ", ")
			}
			rows[i] = present.Row{
				Cells: []string{
					f.ID,
					OrDash(f.Schema.Type),
					BoolString(f.Searchable),
					BoolString(f.Navigable),
					BoolString(f.Orderable),
					clauseNames,
					f.Name,
				},
			}
		} else {
			rows[i] = present.Row{
				Cells: []string{f.ID, OrDash(f.Schema.Type), f.Name},
			}
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: rows},
		},
	}
}

// PresentIssueFields creates a table view for an issue's field values.
// Output: FIELD_ID|NAME|TYPE|VALUE per #230 spec.
func (FieldPresenter) PresentIssueFields(entries []api.IssueFieldEntry) *present.OutputModel {
	rows := make([]present.Row, len(entries))
	for i, e := range entries {
		rows[i] = present.Row{
			Cells: []string{e.ID, e.Name, OrDash(e.Type), e.Value},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"FIELD_ID", "NAME", "TYPE", "VALUE"},
				Rows:    rows,
			},
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

// PresentDeleted creates a success message for field deletion (soft-delete to trash).
func (FieldPresenter) PresentDeleted(fieldID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Deleted field %s (moved to trash — use fields restore to recover)", fieldID),
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
func (FieldPresenter) PresentOptionDeleted(optionID, contextID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Deleted option %s from context %s", optionID, contextID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// FieldShowRow represents a single row in the fields show denormalized view.
type FieldShowRow struct {
	ContextID   string `json:"context_id"`
	Context     string `json:"context"`
	Projects    string `json:"projects"`
	OptionID    string `json:"option_id"`
	OptionValue string `json:"option_value"`
}

// PresentFieldShow creates a table view for a field's denormalized context/option data.
func (FieldPresenter) PresentFieldShow(rows []FieldShowRow) *present.OutputModel {
	tableRows := make([]present.Row, len(rows))
	for i, r := range rows {
		tableRows[i] = present.Row{
			Cells: []string{r.ContextID, r.Context, r.Projects, r.OptionID, r.OptionValue},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"CONTEXT_ID", "CONTEXT", "PROJECTS", "OPTION_ID", "OPTION_VALUE"},
				Rows:    tableRows,
			},
		},
	}
}

// PresentFieldShowEmpty creates an info message when a field has no contexts.
func (FieldPresenter) PresentFieldShowEmpty(fieldID string) *present.OutputModel {
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
func (FieldPresenter) PresentFieldOptionsWithHeader(_ string, options []api.FieldOptionValue) *present.OutputModel {
	rows := make([]present.Row, len(options))
	for i, opt := range options {
		value := opt.Value
		if value == "" {
			value = opt.Name
		}
		disabled := "no"
		if opt.Disabled {
			disabled = "yes"
		}
		rows[i] = present.Row{
			Cells: []string{opt.ID, value, disabled},
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
