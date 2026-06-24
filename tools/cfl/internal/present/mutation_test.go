package present

import (
	"testing"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
)

func TestSpaceMutationPresenters(t *testing.T) {
	t.Parallel()

	space := &api.Space{
		Key:   "DEV",
		Name:  "Development",
		Links: api.Links{WebUI: "/spaces/DEV"},
	}

	createModel := SpacePresenter{}.PresentCreate(space, "https://example.atlassian.net/wiki")
	createSummary := requireMessageSection(t, createModel, 0)
	testutil.Equal(t, "Created space: Development", createSummary.Message)
	createFields := requireDetailSection(t, createModel, 1)
	testutil.Equal(t, []sharedpresent.Field{
		{Label: "Key", Value: "DEV"},
		{Label: "URL", Value: "https://example.atlassian.net/wiki/spaces/DEV"},
	}, createFields.Fields)

	updateModel := SpacePresenter{}.PresentUpdate(space)
	updateSummary := requireMessageSection(t, updateModel, 0)
	testutil.Equal(t, "Updated space: Development (DEV)", updateSummary.Message)

	deleteModel := SpacePresenter{}.PresentDelete(space)
	deleteSummary := requireMessageSection(t, deleteModel, 0)
	testutil.Equal(t, "Deleted space: Development (DEV)", deleteSummary.Message)
}

func TestPageMutationPresenters(t *testing.T) {
	t.Parallel()

	page := &api.Page{
		ID:      "12345",
		Title:   "Release Plan",
		SpaceID: "DEV",
		Version: &api.Version{Number: 7},
		Links:   api.Links{WebUI: "/spaces/DEV/pages/12345"},
	}

	createModel := PagePresenter{}.PresentCreate(page, "https://example.atlassian.net/wiki")
	createSummary := requireMessageSection(t, createModel, 0)
	testutil.Equal(t, "Created page: Release Plan", createSummary.Message)
	createFields := requireDetailSection(t, createModel, 1)
	testutil.Equal(t, []sharedpresent.Field{
		{Label: "ID", Value: "12345"},
		{Label: "URL", Value: "https://example.atlassian.net/wiki/spaces/DEV/pages/12345"},
	}, createFields.Fields)

	editModel := PagePresenter{}.PresentEdit(page, "https://example.atlassian.net/wiki", true)
	editWarning := requireMessageSection(t, editModel, 0)
	testutil.Equal(t, sharedpresent.StreamStderr, editWarning.Stream)
	testutil.Equal(t, "Using --legacy flag. If this page uses the cloud editor, it may switch to the legacy editor.", editWarning.Message)
	editSummary := requireMessageSection(t, editModel, 1)
	testutil.Equal(t, "Updated page: Release Plan", editSummary.Message)
	editFields := requireDetailSection(t, editModel, 2)
	testutil.Equal(t, []sharedpresent.Field{
		{Label: "ID", Value: "12345"},
		{Label: "Version", Value: "7"},
		{Label: "URL", Value: "https://example.atlassian.net/wiki/spaces/DEV/pages/12345"},
	}, editFields.Fields)

	copyModel := PagePresenter{}.PresentCopy(page)
	copySummary := requireMessageSection(t, copyModel, 0)
	testutil.Equal(t, "Copied page: Release Plan", copySummary.Message)
	copyFields := requireDetailSection(t, copyModel, 1)
	testutil.Equal(t, []sharedpresent.Field{
		{Label: "ID", Value: "12345"},
		{Label: "Space", Value: "DEV"},
		{Label: "Version", Value: "7"},
	}, copyFields.Fields)

	deleteModel := PagePresenter{}.PresentDelete(page)
	deleteSummary := requireMessageSection(t, deleteModel, 0)
	testutil.Equal(t, "Deleted page: Release Plan (ID: 12345)", deleteSummary.Message)
}

func TestAttachmentMutationPresenters(t *testing.T) {
	t.Parallel()

	attachment := &api.Attachment{ID: "att-1", Title: "spec.pdf"}

	uploadModel := AttachmentPresenter{}.PresentUpload("spec.pdf", attachment, 1536)
	uploadSummary := requireMessageSection(t, uploadModel, 0)
	testutil.Equal(t, "Uploaded: spec.pdf", uploadSummary.Message)
	uploadFields := requireDetailSection(t, uploadModel, 1)
	testutil.Equal(t, []sharedpresent.Field{
		{Label: "ID", Value: "att-1"},
		{Label: "Title", Value: "spec.pdf"},
		{Label: "Size", Value: "1.5 KB"},
	}, uploadFields.Fields)

	downloadModel := AttachmentPresenter{}.PresentDownload("downloads/spec.pdf", 2048)
	downloadSummary := requireMessageSection(t, downloadModel, 0)
	testutil.Equal(t, "Downloaded: downloads/spec.pdf", downloadSummary.Message)
	downloadFields := requireDetailSection(t, downloadModel, 1)
	testutil.Equal(t, []sharedpresent.Field{{Label: "Size", Value: "2.0 KB"}}, downloadFields.Fields)

	deleteModel := AttachmentPresenter{}.PresentDelete(attachment)
	deleteSummary := requireMessageSection(t, deleteModel, 0)
	testutil.Equal(t, "Deleted attachment: spec.pdf (ID: att-1)", deleteSummary.Message)
}

func TestPresentDeletionCancelled(t *testing.T) {
	t.Parallel()

	model := PresentDeletionCancelled()
	msg := requireMessageSection(t, model, 0)
	testutil.Equal(t, sharedpresent.StreamStderr, msg.Stream)
	testutil.Equal(t, "Deletion cancelled.", msg.Message)
}
