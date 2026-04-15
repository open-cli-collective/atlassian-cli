package present

import (
	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// FieldPresenter creates presentation models for field data.
type FieldPresenter struct{}

// PresentList creates a table view for a list of fields.
func (FieldPresenter) PresentList(fields []api.Field) *present.OutputModel {
	rows := make([]present.Row, len(fields))
	for i, f := range fields {
		custom := "no"
		if f.Custom {
			custom = "yes"
		}
		rows[i] = present.Row{
			Cells: []string{f.ID, f.Name, f.Schema.Type, custom},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "TYPE", "CUSTOM"},
				Rows:    rows,
			},
		},
	}
}

// EditableField represents a field from issue edit metadata.
type EditableField struct {
	ID       string
	Name     string
	Type     string
	Required bool
}

// PresentEditableFields creates a table view for editable fields.
func (FieldPresenter) PresentEditableFields(fields []EditableField) *present.OutputModel {
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

// FieldOption represents a field option value.
type FieldOption struct {
	ID    string
	Value string
}

// PresentFieldOptions creates a table view for field options.
func (FieldPresenter) PresentFieldOptions(options []FieldOption) *present.OutputModel {
	rows := make([]present.Row, len(options))
	for i, opt := range options {
		rows[i] = present.Row{
			Cells: []string{opt.ID, opt.Value},
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

// FieldContext represents a field context.
type FieldContext struct {
	ID               string
	Name             string
	IsGlobalContext  bool
	IsAnyIssueType   bool
}

// PresentContexts creates a table view for field contexts.
func (FieldPresenter) PresentContexts(contexts []FieldContext) *present.OutputModel {
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

// FieldContextOption represents a field context option.
type FieldContextOption struct {
	ID       string
	Value    string
	Disabled bool
}

// PresentContextOptions creates a table view for field context options.
func (FieldPresenter) PresentContextOptions(options []FieldContextOption) *present.OutputModel {
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
