// Package present provides presenters that map domain types to presentation models.
package present

import (
	"github.com/open-cli-collective/atlassian-go/present"
)

// ConfigPresenter creates presentation models for config commands.
type ConfigPresenter struct{}

// ConfigEntry represents a single configuration entry.
type ConfigEntry struct {
	Key    string
	Value  string
	Source string
}

// PresentConfig creates a table presentation model for configuration entries.
func (ConfigPresenter) PresentConfig(entries []ConfigEntry) *present.OutputModel {
	rows := make([]present.Row, len(entries))
	for i, e := range entries {
		rows[i] = present.Row{
			Cells: []string{e.Key, e.Value, e.Source},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"KEY", "VALUE", "SOURCE"},
				Rows:    rows,
			},
		},
	}
}
