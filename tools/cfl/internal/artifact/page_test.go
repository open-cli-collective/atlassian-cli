package artifact

import (
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/atime"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
)

func TestProjectPage(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	page := &api.Page{
		ID:       "12345",
		Title:    "Test Page",
		SpaceID:  "SPACE123",
		ParentID: "PARENT456",
		Version: &api.Version{
			Number:    5,
			AuthorID:  "user123",
			CreatedAt: &atime.AtlassianTime{Time: createdAt},
		},
	}

	t.Run("agent mode includes core fields only", func(t *testing.T) {
		art := ProjectPage(page, "DEV", "<p>content</p>", artifact.Agent)

		testutil.Equal(t, "12345", art.ID)
		testutil.Equal(t, "Test Page", art.Title)
		testutil.Equal(t, "SPACE123", art.SpaceID)
		testutil.Equal(t, "DEV", art.SpaceKey)
		testutil.Equal(t, "PARENT456", art.ParentID)
		testutil.Equal(t, "<p>content</p>", art.Content)
		testutil.Equal(t, 0, art.Version) // not set in agent mode
		testutil.Equal(t, "", art.CreatedAt)
		testutil.Equal(t, "", art.AuthorID)
	})

	t.Run("full mode includes version and dates", func(t *testing.T) {
		art := ProjectPage(page, "DEV", "<p>content</p>", artifact.Full)

		testutil.Equal(t, "12345", art.ID)
		testutil.Equal(t, "Test Page", art.Title)
		testutil.Equal(t, 5, art.Version)
		testutil.Equal(t, "2024-01-15T10:30:00Z", art.CreatedAt) // UTC with Z offset
		testutil.Equal(t, "user123", art.AuthorID)
	})

	t.Run("handles nil version", func(t *testing.T) {
		pageNoVersion := &api.Page{
			ID:      "12345",
			Title:   "Test Page",
			SpaceID: "SPACE123",
		}

		art := ProjectPage(pageNoVersion, "DEV", "<p>content</p>", artifact.Full)
		testutil.Equal(t, 0, art.Version)
		testutil.Equal(t, "", art.CreatedAt)
	})
}

func TestProjectPageListItem(t *testing.T) {
	t.Parallel()

	page := &api.Page{
		ID:       "12345",
		Title:    "Test Page",
		SpaceID:  "SPACE123",
		Status:   "current",
		ParentID: "PARENT456",
		Version: &api.Version{
			Number: 3,
		},
	}

	t.Run("agent mode includes core metadata", func(t *testing.T) {
		art := ProjectPageListItem(page, artifact.Agent)

		testutil.Equal(t, "12345", art.ID)
		testutil.Equal(t, "Test Page", art.Title)
		testutil.Equal(t, "SPACE123", art.SpaceID)
		testutil.Equal(t, "current", art.Status)
		testutil.Equal(t, 0, art.Version)   // not set in agent mode
		testutil.Equal(t, "", art.ParentID) // not set in agent mode
	})

	t.Run("full mode includes version and parent", func(t *testing.T) {
		art := ProjectPageListItem(page, artifact.Full)

		testutil.Equal(t, "12345", art.ID)
		testutil.Equal(t, "Test Page", art.Title)
		testutil.Equal(t, 3, art.Version)
		testutil.Equal(t, "PARENT456", art.ParentID)
	})
}

func TestProjectPageListItems(t *testing.T) {
	t.Parallel()

	pages := []api.Page{
		{ID: "1", Title: "Page 1", SpaceID: "S1", Status: "current"},
		{ID: "2", Title: "Page 2", SpaceID: "S1", Status: "archived"},
	}

	arts := ProjectPageListItems(pages, artifact.Agent)

	testutil.Equal(t, 2, len(arts))
	testutil.Equal(t, "1", arts[0].ID)
	testutil.Equal(t, "Page 1", arts[0].Title)
	testutil.Equal(t, "2", arts[1].ID)
	testutil.Equal(t, "archived", arts[1].Status)
}
