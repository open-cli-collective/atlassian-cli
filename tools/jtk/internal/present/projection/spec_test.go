package projection

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// testRegistry mirrors the shape of IssueListSpec for self-contained tests.
var testRegistry = Registry{
	{Header: "KEY", Identity: true},
	{Header: "SUMMARY", FieldID: "summary"},
	{Header: "STATUS", FieldID: "status"},
	{Header: "ASSIGNEE", FieldID: "assignee"},
	{Header: "TYPE", FieldID: "issuetype"},
	{Header: "POINTS", FieldID: "customfield_10035", Extended: true},
}

func TestRegistry_ForMode_FiltersExtended(t *testing.T) {
	t.Parallel()
	testutil.Equal(t, 5, len(testRegistry.ForMode(false)))
	testutil.Equal(t, 6, len(testRegistry.ForMode(true)))
}

func TestRegistry_Match_HeaderCaseInsensitive(t *testing.T) {
	t.Parallel()
	for _, tok := range []string{"KEY", "key", "Key"} {
		spec, ok := testRegistry.Match(tok, nil)
		testutil.True(t, ok)
		testutil.Equal(t, "KEY", spec.Header)
	}
}

func TestRegistry_Match_FieldID_NoFetchNeeded(t *testing.T) {
	t.Parallel()
	spec, ok := testRegistry.Match("summary", nil)
	testutil.True(t, ok, "expected match for field ID token")
	testutil.Equal(t, "SUMMARY", spec.Header)
}

func TestRegistry_Match_HumanName_UsesFieldsSlice(t *testing.T) {
	t.Parallel()
	fields := []api.Field{
		{ID: "issuetype", Name: "Issue Type"},
	}
	// Not a header, alias, or FieldID match; must fall back to api.Field.Name.
	_, ok := testRegistry.Match("Issue Type", nil)
	testutil.False(t, ok, "fallback should miss when fields is nil")

	spec, ok := testRegistry.Match("Issue Type", fields)
	testutil.True(t, ok, "fallback should hit when api.Field.Name matches")
	testutil.Equal(t, "TYPE", spec.Header)
}

func TestRegistry_Match_Unknown(t *testing.T) {
	t.Parallel()
	_, ok := testRegistry.Match("bogus", []api.Field{{ID: "something", Name: "Something"}})
	testutil.False(t, ok, "unknown token should not match")
}

func TestRegistry_Match_HumanNameResolvesToUnrenderedField(t *testing.T) {
	t.Parallel()
	// Name matches a Jira field whose ID is not in the registry. Match
	// returns false — Resolve handles the "exists but not rendered" case.
	fields := []api.Field{{ID: "customfield_99999", Name: "Phantom"}}
	_, ok := testRegistry.Match("Phantom", fields)
	testutil.False(t, ok)
}
