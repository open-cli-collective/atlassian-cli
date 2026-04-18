package cache

import (
	"errors"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestEntries_DependencyOrdering(t *testing.T) {
	cleanup := SetEntriesForTest([]Entry{
		{Name: "issuetypes", DependsOn: []string{"projects"}},
		{Name: "projects"},
		{Name: "statuses", DependsOn: []string{"projects"}},
		{Name: "fields"},
	})
	defer cleanup()

	got := Entries()
	names := make([]string, len(got))
	for i, e := range got {
		names[i] = e.Name
	}

	// Dependents must appear after their deps. Exact order: independents first
	// (projects, fields — stable), then dependents (issuetypes, statuses — stable).
	testutil.Equal(t, names[0], "projects")
	testutil.Equal(t, names[1], "fields")
	testutil.Equal(t, names[2], "issuetypes")
	testutil.Equal(t, names[3], "statuses")
}

func TestEntries_NoDependencies(t *testing.T) {
	cleanup := SetEntriesForTest([]Entry{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
	})
	defer cleanup()

	got := Entries()
	testutil.Equal(t, len(got), 3)
	testutil.Equal(t, got[0].Name, "a")
	testutil.Equal(t, got[1].Name, "b")
	testutil.Equal(t, got[2].Name, "c")
}

func TestLookup(t *testing.T) {
	cleanup := SetEntriesForTest([]Entry{
		{Name: "fields", TTL: "24h"},
	})
	defer cleanup()

	e, err := Lookup("fields")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, e.Name, "fields")
	testutil.Equal(t, e.TTL, "24h")

	_, err = Lookup("nope")
	testutil.Error(t, err)
	if !errors.Is(err, ErrUnknownResource) {
		t.Fatalf("expected ErrUnknownResource, got %v", err)
	}
}

func TestNames(t *testing.T) {
	cleanup := SetEntriesForTest([]Entry{
		{Name: "fields"},
		{Name: "projects"},
	})
	defer cleanup()

	testutil.Equal(t, len(Names()), 2)
	testutil.Equal(t, Names()[0], "fields")
	testutil.Equal(t, Names()[1], "projects")
}

func TestIsAvailable_NilPredicate(t *testing.T) {
	e := Entry{Name: "x"}
	testutil.Equal(t, e.IsAvailable(nil), true)
}

func TestProjectDependents(t *testing.T) {
	deps := ProjectDependents()
	testutil.Equal(t, deps[0], "projects")
	testutil.Equal(t, deps[1], "issuetypes")
	testutil.Equal(t, deps[2], "statuses")
}
