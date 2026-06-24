package present

import (
	"strings"
	"testing"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func historyFixture() []api.IssueChangelogHistory {
	return []api.IssueChangelogHistory{
		{
			ID:      "10001",
			Created: "2026-06-20T15:04:05.000+0000",
			Author:  &api.User{AccountID: "acct-1", DisplayName: "Alice"},
			Items: []api.IssueChangelogItem{
				{
					Field:      "status",
					FieldType:  "jira",
					FieldID:    "status",
					From:       "1",
					FromString: "Open",
					To:         "3",
					ToString:   "Done",
				},
				{
					Field:      "summary",
					FieldType:  "jira",
					FieldID:    "summary",
					FromString: strings.Repeat("A", 120),
					ToString:   "Short summary",
				},
			},
		},
	}
}

func issueHistoryTable(t *testing.T, model *sharedpresent.OutputModel) *sharedpresent.TableSection {
	t.Helper()
	for _, section := range model.Sections {
		if table, ok := section.(*sharedpresent.TableSection); ok {
			return table
		}
	}
	t.Fatalf("no table section")
	return nil
}

func TestIssueHistoryPresenter_DefaultRows(t *testing.T) {
	t.Parallel()

	rows := FlattenIssueHistory(historyFixture())
	model := IssueHistoryPresenter{}.PresentIssueHistory(rows, false, false)
	table := issueHistoryTable(t, model)

	testutil.Equal(t, []string{"ID", "CREATED", "AUTHOR", "FIELD", "FROM", "TO"}, table.Headers)
	testutil.Equal(t, 2, len(table.Rows))
	testutil.Equal(t, []string{"10001", "2026-06-20", "Alice", "status", "Open", "Done"}, table.Rows[0].Cells)
	testutil.Contains(t, table.Rows[1].Cells[4], "...")
	if len(table.Rows[1].Cells[4]) > 80 {
		t.Fatalf("default history cell should be bounded, got %d chars", len(table.Rows[1].Cells[4]))
	}
}

func TestIssueHistoryPresenter_ExtendedRows(t *testing.T) {
	t.Parallel()

	rows := FlattenIssueHistory(historyFixture())
	model := IssueHistoryPresenter{}.PresentIssueHistory(rows, true, true)
	table := issueHistoryTable(t, model)

	testutil.Equal(t, []string{"ID", "CREATED", "AUTHOR", "ACCOUNT_ID", "FIELD", "FIELD_ID", "TYPE", "FROM_ID", "FROM", "TO_ID", "TO"}, table.Headers)
	testutil.Equal(t, []string{"10001", "2026-06-20T15:04:05.000+0000", "Alice", "acct-1", "status", "status", "jira", "1", "Open", "3", "Done"}, table.Rows[0].Cells)
	testutil.Equal(t, strings.Repeat("A", 120), table.Rows[1].Cells[8])
}

func TestIssueHistorySpec_MatchesDefaultHeaders(t *testing.T) {
	t.Parallel()

	rows := FlattenIssueHistory(historyFixture())
	model := IssueHistoryPresenter{}.PresentIssueHistory(rows, false, false)
	table := issueHistoryTable(t, model)
	spec := IssueHistorySpec.ForMode(false)

	testutil.Equal(t, len(spec), len(table.Headers))
	for i, col := range spec {
		testutil.Equal(t, col.Header, table.Headers[i])
	}
}

func TestIssueHistorySpec_MatchesExtendedHeaders(t *testing.T) {
	t.Parallel()

	rows := FlattenIssueHistory(historyFixture())
	model := IssueHistoryPresenter{}.PresentIssueHistory(rows, true, true)
	table := issueHistoryTable(t, model)
	spec := IssueHistorySpec.ForMode(true)

	testutil.Equal(t, len(spec), len(table.Headers))
	for i, col := range spec {
		testutil.Equal(t, col.Header, table.Headers[i])
	}
}

func TestIssueHistoryPresenter_Pagination(t *testing.T) {
	t.Parallel()

	rows := FlattenIssueHistory(historyFixture())
	model := IssueHistoryPresenter{}.PresentIssueHistoryWithPagination(rows, false, false, true, "2")
	out := sharedpresent.Render(model, sharedpresent.StyleAgent)
	testutil.Contains(t, out.Stdout, "More results available (next: 2)")
}
