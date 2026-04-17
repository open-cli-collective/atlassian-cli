package projection

import (
	"context"
	"errors"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// fetchStub returns cannedFields on Nth call, counting invocations.
type fetchStub struct {
	calls  int
	fields []api.Field
	err    error
}

func (s *fetchStub) fetch(_ context.Context) ([]api.Field, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.fields, nil
}

func TestResolve_EmptyFieldsFlag_NoFetch_FullRegistry(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{err: errors.New("should not be called")}
	selected, applied, err := Resolve(context.Background(), testRegistry, false, "", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.False(t, applied, "projectionApplied must be false when --fields is empty")
	testutil.Equal(t, 5, len(selected)) // default mode registry (5 of 6)
	testutil.Equal(t, 0, stub.calls)
}

func TestResolve_EmptyFieldsFlag_ExtendedRegistry(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{}
	selected, applied, err := Resolve(context.Background(), testRegistry, true, "", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.False(t, applied)
	testutil.Equal(t, 6, len(selected)) // extended mode registry (all 6)
}

func TestResolve_HeaderAliases_NoFetch(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{err: errors.New("should not be called")}
	selected, applied, err := Resolve(context.Background(), testRegistry, false, "KEY,SUMMARY,STATUS", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.True(t, applied)
	testutil.Equal(t, 0, stub.calls)
	testutil.Equal(t, 3, len(selected))
	testutil.Equal(t, "KEY", selected[0].Header)
	testutil.Equal(t, "SUMMARY", selected[1].Header)
	testutil.Equal(t, "STATUS", selected[2].Header)
}

func TestResolve_FieldIDs_NoFetch(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{err: errors.New("should not be called")}
	_, _, err := Resolve(context.Background(), testRegistry, false, "summary,assignee", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 0, stub.calls)
}

func TestResolve_HumanName_TriggersExactlyOneFetch(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "issuetype", Name: "Issue Type"},
	}}
	selected, applied, err := Resolve(context.Background(), testRegistry, false, "Issue Type", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.True(t, applied)
	testutil.Equal(t, 1, stub.calls)
	testutil.Equal(t, 2, len(selected)) // KEY + TYPE
	testutil.Equal(t, "KEY", selected[0].Header)
	testutil.Equal(t, "TYPE", selected[1].Header)
}

func TestResolve_MultiToken_FetchesOnce(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "issuetype", Name: "Issue Type"},
		{ID: "assignee", Name: "Assignee"},
	}}
	_, _, err := Resolve(context.Background(), testRegistry, false, "Issue Type,Summary,Issue Type", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 1, stub.calls)
}

func TestResolve_IdentityAlwaysPrepended(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{}
	selected, _, err := Resolve(context.Background(), testRegistry, false, "SUMMARY", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, len(selected))
	testutil.Equal(t, "KEY", selected[0].Header)
	testutil.Equal(t, "SUMMARY", selected[1].Header)
}

func TestResolve_IdentityNotDuplicated(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{}
	selected, _, err := Resolve(context.Background(), testRegistry, false, "KEY,SUMMARY", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, len(selected))
	testutil.Equal(t, "KEY", selected[0].Header)
	testutil.Equal(t, "SUMMARY", selected[1].Header)
}

func TestResolve_UserOrderPreserved_AfterIdentity(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{}
	selected, _, err := Resolve(context.Background(), testRegistry, false, "STATUS,SUMMARY", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 3, len(selected))
	testutil.Equal(t, "KEY", selected[0].Header)
	testutil.Equal(t, "STATUS", selected[1].Header)
	testutil.Equal(t, "SUMMARY", selected[2].Header)
}

func TestResolve_UnknownToken_FallbackAttemptedButFails(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "somethingelse", Name: "Something Else"},
	}}
	_, _, err := Resolve(context.Background(), testRegistry, false, "bogus", stub.fetch, "issues list")
	var ufe *UnknownFieldError
	testutil.True(t, errors.As(err, &ufe))
	testutil.Equal(t, 1, stub.calls)
}

func TestResolve_UnrenderedField_ByHumanName(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "customfield_99999", Name: "Phantom"},
	}}
	_, _, err := Resolve(context.Background(), testRegistry, false, "Phantom", stub.fetch, "issues list")
	var ure *UnrenderedFieldError
	testutil.True(t, errors.As(err, &ure))
	testutil.Equal(t, "Phantom", ure.JiraName)
	testutil.Equal(t, "customfield_99999", ure.JiraID)
	testutil.Equal(t, "issues list", ure.Command)
}

func TestResolve_UnrenderedField_ByFieldID_UsesHumanNameInMessage(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "customfield_99999", Name: "Phantom"},
	}}
	_, _, err := Resolve(context.Background(), testRegistry, false, "customfield_99999", stub.fetch, "issues list")
	var ure *UnrenderedFieldError
	testutil.True(t, errors.As(err, &ure))
	testutil.Equal(t, "Phantom", ure.JiraName)
	testutil.Equal(t, "customfield_99999", ure.JiraID)
	// Error message must resolve the ID to the human-readable name.
	testutil.Contains(t, err.Error(), "Phantom")
	testutil.Contains(t, err.Error(), "customfield_99999")
}

func TestResolve_ExtendedOnlyToken_WithoutFlag_Errors(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{}
	_, _, err := Resolve(context.Background(), testRegistry, false, "POINTS", stub.fetch, "issues list")
	var eoe *ExtendedOnlyError
	testutil.True(t, errors.As(err, &eoe))
	testutil.Equal(t, "POINTS", eoe.Header)
	testutil.Equal(t, 0, stub.calls)
}

func TestResolve_ExtendedOnlyToken_WithFlag_Resolves(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{}
	selected, applied, err := Resolve(context.Background(), testRegistry, true, "POINTS", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.True(t, applied)
	testutil.Equal(t, 2, len(selected)) // KEY + POINTS
}

func TestResolve_FetchFieldsErrorPropagates(t *testing.T) {
	t.Parallel()
	boom := errors.New("network down")
	stub := &fetchStub{err: boom}
	_, _, err := Resolve(context.Background(), testRegistry, false, "Issue Type", stub.fetch, "issues list")
	testutil.True(t, errors.Is(err, boom))
}
