package present

import (
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/atime"
	sharedpresent "github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
)

func TestSpacePresenter_PresentList(t *testing.T) {
	t.Parallel()

	model := SpacePresenter{}.PresentList([]api.Space{
		{ID: "100", Key: "DEV", Type: "global", Name: "Development", Status: "current"},
	}, true, "cursor-123", true)

	table := requireTableSection(t, model, 0)
	testutil.Equal(t, []string{"ID", "KEY", "TYPE", "STATUS", "NAME"}, table.Headers)
	testutil.Equal(t, []string{"100", "DEV", "global", "current", "Development"}, table.Rows[0].Cells)

	msg := requireMessageSection(t, model, 1)
	testutil.Equal(t, sharedpresent.MessageInfo, msg.Kind)
	testutil.Equal(t, sharedpresent.StreamStderr, msg.Stream)
	testutil.Equal(t, `Next page: cfl space list --cursor "cursor-123"`, msg.Message)
}

func TestSpacePresenter_PresentEmpty(t *testing.T) {
	t.Parallel()

	model := SpacePresenter{}.PresentEmpty()
	msg := requireMessageSection(t, model, 0)
	testutil.Equal(t, "No spaces found.", msg.Message)
	testutil.Equal(t, sharedpresent.StreamStderr, msg.Stream)
}

func TestPagePresenter_PresentList(t *testing.T) {
	t.Parallel()

	model := PagePresenter{}.PresentList([]api.Page{
		{ID: "200", Title: "Release Plan", Status: "current", ParentID: "100", Version: &api.Version{Number: 7}},
	}, true, true)

	table := requireTableSection(t, model, 0)
	testutil.Equal(t, []string{"ID", "TITLE", "STATUS", "VERSION", "PARENT ID"}, table.Headers)
	testutil.Equal(t, []string{"200", "Release Plan", "current", "v7", "100"}, table.Rows[0].Cells)

	msg := requireMessageSection(t, model, 1)
	testutil.Equal(t, "(showing first 1 results, use --limit to see more)", msg.Message)
	testutil.Equal(t, sharedpresent.StreamStderr, msg.Stream)
}

func TestPagePresenter_PresentEmpty(t *testing.T) {
	t.Parallel()

	model := PagePresenter{}.PresentEmpty("DEV")
	msg := requireMessageSection(t, model, 0)
	testutil.Equal(t, "No pages found in space DEV.", msg.Message)
}

func TestPageHistoryPresenter_PresentListAndIDs(t *testing.T) {
	t.Parallel()

	createdAt := atlassianTime(t, "2024-01-02T03:04:05Z")
	versions := []api.Version{{Number: 15, AuthorID: "user-1", CreatedAt: createdAt}}

	model := PageHistoryPresenter{}.PresentList(versions, "next-1", "12345", true)
	table := requireTableSection(t, model, 0)
	testutil.Equal(t, []string{"VERSION", "CREATED", "AUTHOR"}, table.Headers)
	testutil.Equal(t, []string{"15", "2024-01-02T03:04:05Z", "user-1"}, table.Rows[0].Cells)

	msg := requireMessageSection(t, model, 1)
	testutil.Equal(t, `Next page: cfl page history list 12345 --cursor "next-1"`, msg.Message)

	idModel := PageHistoryPresenter{}.PresentIDs(versions, "next-1", "12345", true)
	idMsg := requireMessageSection(t, idModel, 0)
	testutil.Equal(t, "15", idMsg.Message)
	testutil.Equal(t, sharedpresent.StreamStdout, idMsg.Stream)
	nextMsg := requireMessageSection(t, idModel, 1)
	testutil.Equal(t, sharedpresent.StreamStderr, nextMsg.Stream)
}

func TestSearchPresenter_PresentList(t *testing.T) {
	t.Parallel()

	model := SearchPresenter{}.PresentList([]api.SearchResult{
		{
			Content:               api.SearchContent{ID: "300", Type: "page", Title: "Deploy Guide"},
			ResultGlobalContainer: api.SearchContainer{DisplayURL: "/spaces/DEV/pages/300"},
			LastModified:          "2024-03-04",
			URL:                   "/wiki/spaces/DEV/pages/300",
		},
	}, true, 20, true)

	table := requireTableSection(t, model, 0)
	testutil.Equal(t, []string{"ID", "TYPE", "SPACE", "TITLE", "MODIFIED", "URL"}, table.Headers)
	testutil.Equal(t, []string{"300", "page", "DEV", "Deploy Guide", "2024-03-04", "/wiki/spaces/DEV/pages/300"}, table.Rows[0].Cells)

	msg := requireMessageSection(t, model, 1)
	testutil.Equal(t, "(showing 1 of 20 results, use --limit to see more)", msg.Message)
}

func TestAttachmentPresenter_PresentListAndEmpty(t *testing.T) {
	t.Parallel()

	model := AttachmentPresenter{}.PresentList([]api.Attachment{
		{ID: "att-1", Title: "design.pdf", MediaType: "application/pdf", FileSize: 1536, Status: "current", Comment: "latest"},
	}, true, true)

	table := requireTableSection(t, model, 0)
	testutil.Equal(t, []string{"ID", "TITLE", "MEDIA TYPE", "FILE SIZE", "STATUS", "COMMENT"}, table.Headers)
	testutil.Equal(t, []string{"att-1", "design.pdf", "application/pdf", "1.5 KB", "current", "latest"}, table.Rows[0].Cells)

	msg := requireMessageSection(t, model, 1)
	testutil.Equal(t, "(showing first 1 results, use --limit to see more)", msg.Message)

	emptyModel := AttachmentPresenter{}.PresentEmpty(true)
	emptyMsg := requireMessageSection(t, emptyModel, 0)
	testutil.Equal(t, "No unused attachments found.", emptyMsg.Message)
}

func TestListPresenters_FallbackPaginationHintWithoutCursor(t *testing.T) {
	t.Parallel()

	spaceModel := SpacePresenter{}.PresentList([]api.Space{{ID: "100", Key: "DEV", Type: "global", Name: "Development"}}, false, "", true)
	spaceMsg := requireMessageSection(t, spaceModel, 1)
	testutil.Equal(t, "(showing first 1 results, use --limit to see more)", spaceMsg.Message)

	historyModel := PageHistoryPresenter{}.PresentList([]api.Version{{Number: 3}}, "", "12345", true)
	historyMsg := requireMessageSection(t, historyModel, 1)
	testutil.Equal(t, "(showing first 1 results, use --limit to see more)", historyMsg.Message)

	idModel := PageHistoryPresenter{}.PresentIDs([]api.Version{{Number: 3}}, "", "12345", true)
	idMsg := requireMessageSection(t, idModel, 1)
	testutil.Equal(t, "(showing first 1 results, use --limit to see more)", idMsg.Message)
}

func TestListPresenterHelpers(t *testing.T) {
	t.Parallel()

	testutil.Equal(t, "DEV", ExtractSpaceKey("/spaces/DEV/pages/123"))
	testutil.Equal(t, "cursor-1", ExtractCursor("/wiki/api/v2/spaces?cursor=cursor-1"))
	testutil.Equal(t, "", ExtractCursor("://bad-url"))
	testutil.Equal(t, "-", truncateText("", 5))
	testutil.Equal(t, "abc...", truncateText("abcdefghi", 6))
	testutil.Equal(t, "-", pageVersionCell(nil))
	testutil.Equal(t, "v2", pageVersionCell(&api.Version{Number: 2}))
	testutil.Equal(t, "-", formatHistoryTime(nil))
	testutil.Equal(t, "0 B", formatAttachmentFileSize(0))
	testutil.Equal(t, "1.0 KB", formatAttachmentFileSize(1024))
}

func requireTableSection(t *testing.T, model *sharedpresent.OutputModel, idx int) *sharedpresent.TableSection {
	t.Helper()

	if len(model.Sections) <= idx {
		t.Fatalf("expected section index %d, got %d sections", idx, len(model.Sections))
	}
	sec, ok := model.Sections[idx].(*sharedpresent.TableSection)
	if !ok {
		t.Fatalf("expected TableSection at %d, got %T", idx, model.Sections[idx])
	}
	return sec
}

func requireMessageSection(t *testing.T, model *sharedpresent.OutputModel, idx int) *sharedpresent.MessageSection {
	t.Helper()

	if len(model.Sections) <= idx {
		t.Fatalf("expected section index %d, got %d sections", idx, len(model.Sections))
	}
	sec, ok := model.Sections[idx].(*sharedpresent.MessageSection)
	if !ok {
		t.Fatalf("expected MessageSection at %d, got %T", idx, model.Sections[idx])
	}
	return sec
}

func atlassianTime(t *testing.T, value string) *atime.AtlassianTime {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	testutil.RequireNoError(t, err)
	at := atime.AtlassianTime{Time: parsed}
	return &at
}
