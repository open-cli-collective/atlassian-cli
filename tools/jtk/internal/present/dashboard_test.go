package present

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestDashboardPresenter_PresentList_ColumnOrder(t *testing.T) {
	t.Parallel()
	dashboards := []api.Dashboard{
		{ID: "10001", Name: "Sprint Board", Owner: &api.User{DisplayName: "Alice"}, IsFavourite: true},
		{ID: "10002", Name: "Incidents", Owner: &api.User{DisplayName: "Bob"}},
	}
	counts := map[string]int{"10001": 4, "10002": 2}

	model := DashboardPresenter{}.PresentList(dashboards, counts)
	table := model.Sections[0].(*present.TableSection)

	wantHeaders := []string{"ID", "GADGETS", "OWNER", "FAVOURITE", "NAME"}
	for i, h := range wantHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, table.Headers[i], h)
		}
	}

	if table.Rows[0].Cells[0] != "10001" {
		t.Errorf("first cell should be ID, got %q", table.Rows[0].Cells[0])
	}
	if table.Rows[0].Cells[1] != "4" {
		t.Errorf("gadgets cell should be '4', got %q", table.Rows[0].Cells[1])
	}
	if table.Rows[0].Cells[2] != "Alice" {
		t.Errorf("owner cell should be 'Alice', got %q", table.Rows[0].Cells[2])
	}
	if table.Rows[0].Cells[3] != "yes" {
		t.Errorf("favourite cell should be 'yes', got %q", table.Rows[0].Cells[3])
	}
	if table.Rows[0].Cells[4] != "Sprint Board" {
		t.Errorf("name cell should be 'Sprint Board', got %q", table.Rows[0].Cells[4])
	}
}

func TestDashboardPresenter_PresentList_NilCounts(t *testing.T) {
	t.Parallel()
	dashboards := []api.Dashboard{
		{ID: "10001", Name: "Board"},
	}

	model := DashboardPresenter{}.PresentList(dashboards, nil)
	table := model.Sections[0].(*present.TableSection)

	if table.Rows[0].Cells[1] != "-" {
		t.Errorf("nil counts should render '-', got %q", table.Rows[0].Cells[1])
	}
}

func TestDashboardPresenter_PresentList_MissingKey(t *testing.T) {
	t.Parallel()
	dashboards := []api.Dashboard{
		{ID: "10001", Name: "Board"},
	}
	counts := map[string]int{"other": 5}

	model := DashboardPresenter{}.PresentList(dashboards, counts)
	table := model.Sections[0].(*present.TableSection)

	if table.Rows[0].Cells[1] != "-" {
		t.Errorf("missing key should render '-', got %q", table.Rows[0].Cells[1])
	}
}

func TestDashboardPresenter_PresentList_ZeroGadgets(t *testing.T) {
	t.Parallel()
	dashboards := []api.Dashboard{
		{ID: "10001", Name: "Board"},
	}
	counts := map[string]int{"10001": 0}

	model := DashboardPresenter{}.PresentList(dashboards, counts)
	table := model.Sections[0].(*present.TableSection)

	if table.Rows[0].Cells[1] != "0" {
		t.Errorf("zero gadgets should render '0', got %q", table.Rows[0].Cells[1])
	}
}

func TestDashboardPresenter_PresentListExtended_ColumnOrder(t *testing.T) {
	t.Parallel()
	dashboards := []api.Dashboard{
		{
			ID:          "10001",
			Name:        "Sprint Board",
			Owner:       &api.User{DisplayName: "Alice"},
			IsFavourite: true,
			Popularity:  5,
			SharePerm:   []api.SharePerm{{Type: "group", Group: &api.SharePermGroup{Name: "developers"}}},
		},
	}
	counts := map[string]int{"10001": 3}

	model := DashboardPresenter{}.PresentListExtended(dashboards, counts)
	table := model.Sections[0].(*present.TableSection)

	wantHeaders := []string{"ID", "GADGETS", "OWNER", "FAVOURITE", "RANK", "PERMISSIONS", "NAME"}
	for i, h := range wantHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, table.Headers[i], h)
		}
	}

	if table.Rows[0].Cells[4] != "5" {
		t.Errorf("rank cell should be '5', got %q", table.Rows[0].Cells[4])
	}
	if table.Rows[0].Cells[5] != "group:developers" {
		t.Errorf("permissions cell should be 'group:developers', got %q", table.Rows[0].Cells[5])
	}
}

func TestDashboardPresenter_PresentListExtended_NilCounts(t *testing.T) {
	t.Parallel()
	dashboards := []api.Dashboard{
		{ID: "10001", Name: "Board", Popularity: 3},
	}

	model := DashboardPresenter{}.PresentListExtended(dashboards, nil)
	table := model.Sections[0].(*present.TableSection)

	if table.Rows[0].Cells[1] != "-" {
		t.Errorf("nil counts should render '-', got %q", table.Rows[0].Cells[1])
	}
}

func TestDashboardPresenter_PresentGadgets_ColumnOrder(t *testing.T) {
	t.Parallel()
	gadgets := []api.DashboardGadget{
		{ID: 100, Title: "Burndown", ModuleID: "sprint-burndown-gadget", Position: api.DashboardGadgetPos{Row: 0, Column: 0}},
		{ID: 101, Title: "Pie Chart", ModuleID: "pie-chart-gadget", Position: api.DashboardGadgetPos{Row: 1, Column: 2}},
	}

	model := DashboardPresenter{}.PresentGadgets(gadgets)
	table := model.Sections[0].(*present.TableSection)

	wantHeaders := []string{"ID", "POSITION", "TITLE", "TYPE"}
	for i, h := range wantHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, table.Headers[i], h)
		}
	}

	if table.Rows[0].Cells[1] != "0,0" {
		t.Errorf("position 0,0 should be shown, got %q", table.Rows[0].Cells[1])
	}
	if table.Rows[1].Cells[1] != "1,2" {
		t.Errorf("position should be '1,2', got %q", table.Rows[1].Cells[1])
	}
	if table.Rows[0].Cells[3] != "sprint-burndown-gadget" {
		t.Errorf("type cell should be module ID, got %q", table.Rows[0].Cells[3])
	}
}

func TestFormatPermissions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		perms []api.SharePerm
		want  string
	}{
		{name: "empty", perms: nil, want: "private"},
		{name: "global", perms: []api.SharePerm{{Type: "global"}}, want: "global"},
		{name: "group with name", perms: []api.SharePerm{{Type: "group", Group: &api.SharePermGroup{Name: "devs"}}}, want: "group:devs"},
		{name: "group without name", perms: []api.SharePerm{{Type: "group"}}, want: "group"},
		{name: "project with key", perms: []api.SharePerm{{Type: "project", Project: &api.SharePermProject{Key: "MON"}}}, want: "project:MON"},
		{name: "project without key", perms: []api.SharePerm{{Type: "project"}}, want: "project"},
		{name: "project with empty key", perms: []api.SharePerm{{Type: "project", Project: &api.SharePermProject{Key: ""}}}, want: "project"},
		{name: "loggedin", perms: []api.SharePerm{{Type: "loggedin"}}, want: "logged-in"},
		{name: "unknown type", perms: []api.SharePerm{{Type: "projectRole"}}, want: "projectRole"},
		{
			name: "multiple",
			perms: []api.SharePerm{
				{Type: "group", Group: &api.SharePermGroup{Name: "devs"}},
				{Type: "global"},
			},
			want: "group:devs, global",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatPermissions(tt.perms)
			if got != tt.want {
				t.Errorf("formatPermissions() = %q, want %q", got, tt.want)
			}
		})
	}
}
