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
