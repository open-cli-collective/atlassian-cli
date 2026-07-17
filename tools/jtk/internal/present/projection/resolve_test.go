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
	selected, applied, err := Resolve(context.Background(), testRegistry, "", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.False(t, applied, "projectionApplied must be false when --fields is empty")
	testutil.Equal(t, 5, len(selected)) // default mode registry (5 of 6)
	testutil.Equal(t, 0, stub.calls)
}

func TestResolve_EmptyFieldsFlag_RemainsCompact(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{}
	selected, applied, err := Resolve(context.Background(), testRegistry, "", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.False(t, applied)
	testutil.Equal(t, 5, len(selected))
}

func TestResolve_HeaderAliases_NoFetch(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{err: errors.New("should not be called")}
	selected, applied, err := Resolve(context.Background(), testRegistry, "KEY,SUMMARY,STATUS", stub.fetch, "issues list")
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
	selected, applied, err := Resolve(context.Background(), testRegistry, "summary,assignee", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.True(t, applied)
	testutil.Equal(t, 0, stub.calls)
	// Tokens resolved to the correct specs (identity KEY prepended).
	testutil.Equal(t, 3, len(selected))
	testutil.Equal(t, "KEY", selected[0].Header)
	testutil.Equal(t, "SUMMARY", selected[1].Header)
	testutil.Equal(t, "ASSIGNEE", selected[2].Header)
}

func TestResolve_HumanName_TriggersExactlyOneFetch(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "issuetype", Name: "Issue Type"},
	}}
	selected, applied, err := Resolve(context.Background(), testRegistry, "Issue Type", stub.fetch, "issues list")
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
	_, _, err := Resolve(context.Background(), testRegistry, "Issue Type,Summary,Issue Type", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 1, stub.calls)
}

func TestResolve_IdentityAlwaysPrepended(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{}
	selected, _, err := Resolve(context.Background(), testRegistry, "SUMMARY", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, len(selected))
	testutil.Equal(t, "KEY", selected[0].Header)
	testutil.Equal(t, "SUMMARY", selected[1].Header)
}

func TestResolve_IdentityNotDuplicated(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{}
	selected, _, err := Resolve(context.Background(), testRegistry, "KEY,SUMMARY", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, len(selected))
	testutil.Equal(t, "KEY", selected[0].Header)
	testutil.Equal(t, "SUMMARY", selected[1].Header)
}

func TestResolve_UserOrderPreserved_AfterIdentity(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{}
	selected, _, err := Resolve(context.Background(), testRegistry, "STATUS,SUMMARY", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 3, len(selected))
	testutil.Equal(t, "KEY", selected[0].Header)
	testutil.Equal(t, "STATUS", selected[1].Header)
	testutil.Equal(t, "SUMMARY", selected[2].Header)
}

// Mixed fast-path (header) and slow-path (human-name) tokens must land in
// the user's order, not be grouped by resolution path.
func TestResolve_UserOrder_MixedFastAndSlowPath(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "issuetype", Name: "Issue Type"},
	}}
	selected, _, err := Resolve(context.Background(), testRegistry, "STATUS,Issue Type,SUMMARY", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 4, len(selected))
	testutil.Equal(t, "KEY", selected[0].Header)
	testutil.Equal(t, "STATUS", selected[1].Header)
	testutil.Equal(t, "TYPE", selected[2].Header)
	testutil.Equal(t, "SUMMARY", selected[3].Header)
}

func TestResolve_UnknownToken_FallbackAttemptedButFails(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "somethingelse", Name: "Something Else"},
	}}
	_, _, err := Resolve(context.Background(), testRegistry, "bogus", stub.fetch, "issues list")
	var ufe *UnknownFieldError
	testutil.True(t, errors.As(err, &ufe))
	testutil.Equal(t, 1, stub.calls)
}

func TestResolve_DynamicSpec_ByHumanName(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "customfield_99999", Name: "Phantom"},
	}}
	selected, projected, err := Resolve(context.Background(), testRegistry, "Phantom", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.True(t, projected)
	// Should contain identity (KEY) + dynamic spec
	var found bool
	for _, s := range selected {
		if s.FieldID == "customfield_99999" && s.Dynamic && s.Header == "Phantom" {
			found = true
		}
	}
	testutil.True(t, found)
}

func TestResolve_DynamicSpec_ByFieldID(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "customfield_99999", Name: "Phantom"},
	}}
	selected, projected, err := Resolve(context.Background(), testRegistry, "customfield_99999", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.True(t, projected)
	var found bool
	for _, s := range selected {
		if s.FieldID == "customfield_99999" && s.Dynamic && s.Header == "Phantom" {
			found = true
		}
	}
	testutil.True(t, found)
}

func TestResolve_AmbiguousFieldName_Errors(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "customfield_10001", Name: "Duplicate"},
		{ID: "customfield_10002", Name: "Duplicate"},
	}}
	_, _, err := Resolve(context.Background(), testRegistry, "Duplicate", stub.fetch, "issues list")
	var afe *AmbiguousFieldNameError
	testutil.True(t, errors.As(err, &afe))
	testutil.Equal(t, 2, len(afe.Matches))
}

func TestResolve_DynamicSpec_HeaderCollision(t *testing.T) {
	t.Parallel()
	// "STATUS" is a registered header. A custom field also named "STATUS"
	// should get a disambiguated header.
	stub := &fetchStub{fields: []api.Field{
		{ID: "customfield_99999", Name: "STATUS"},
	}}
	selected, _, err := Resolve(context.Background(), testRegistry, "STATUS,customfield_99999", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	var dynamicHeader string
	for _, s := range selected {
		if s.Dynamic {
			dynamicHeader = s.Header
		}
	}
	testutil.Equal(t, "STATUS (customfield_99999)", dynamicHeader)
}

func TestResolve_OptionalToken_ResolvesExplicitly(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{}
	selected, applied, err := Resolve(context.Background(), testRegistry, "POINTS", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.True(t, applied)
	testutil.Equal(t, 2, len(selected)) // KEY + POINTS
}

func TestResolve_OptionalToken_ByHumanName_Resolves(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{
		{ID: "customfield_10035", Name: "Story Points"},
	}}
	selected, applied, err := Resolve(context.Background(), testRegistry, "Story Points", stub.fetch, "issues list")
	testutil.RequireNoError(t, err)
	testutil.True(t, applied)
	testutil.Equal(t, "POINTS", selected[1].Header)
}

// Multiple unknown tokens are batched into a single UnknownFieldError so
// users see all failures at once, not one-at-a-time.
func TestResolve_MultipleUnknownTokens_BatchedIntoOneError(t *testing.T) {
	t.Parallel()
	stub := &fetchStub{fields: []api.Field{}}
	_, _, err := Resolve(context.Background(), testRegistry, "bogus1,bogus2,bogus3", stub.fetch, "issues list")
	var ufe *UnknownFieldError
	testutil.True(t, errors.As(err, &ufe))
	testutil.Equal(t, 3, len(ufe.Unknown))
	testutil.Equal(t, "bogus1", ufe.Unknown[0])
	testutil.Equal(t, "bogus2", ufe.Unknown[1])
	testutil.Equal(t, "bogus3", ufe.Unknown[2])
	testutil.Contains(t, err.Error(), "unknown fields")
}

func TestResolve_FetchFieldsErrorPropagates(t *testing.T) {
	t.Parallel()
	boom := errors.New("network down")
	stub := &fetchStub{err: boom}
	_, _, err := Resolve(context.Background(), testRegistry, "Issue Type", stub.fetch, "issues list")
	testutil.True(t, errors.Is(err, boom))
}
