package artifact

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
)

func TestProjectSearchResult(t *testing.T) {
	t.Parallel()

	result := &api.SearchResult{
		Content: api.SearchContent{
			ID:   "12345",
			Type: "page",
		},
		Title:   "Test Search Result",
		Excerpt: "This is a short excerpt",
		URL:     "/wiki/spaces/DEV/pages/12345",
		ResultGlobalContainer: api.SearchContainer{
			Title: "Development Space",
		},
		LastModified: "2024-01-15T10:30:00Z",
	}

	t.Run("agent mode includes core fields", func(t *testing.T) {
		art := ProjectSearchResult(result, artifact.Agent)

		testutil.Equal(t, "12345", art.ID)
		testutil.Equal(t, "Test Search Result", art.Title)
		testutil.Equal(t, "page", art.Type)
		testutil.Equal(t, "Development Space", art.SpaceName)
		testutil.Equal(t, "This is a short excerpt", art.Excerpt)
		testutil.Equal(t, "", art.URL)        // not set in agent mode
		testutil.Equal(t, "", art.ModifiedAt) // not set in agent mode
	})

	t.Run("full mode includes URL and modified date", func(t *testing.T) {
		art := ProjectSearchResult(result, artifact.Full)

		testutil.Equal(t, "12345", art.ID)
		testutil.Equal(t, "/wiki/spaces/DEV/pages/12345", art.URL)
		testutil.Equal(t, "2024-01-15T10:30:00Z", art.ModifiedAt)
	})

	t.Run("truncates long excerpt in agent mode", func(t *testing.T) {
		longExcerpt := strings.Repeat("a", 250)
		resultLong := &api.SearchResult{
			Content:               api.SearchContent{ID: "1", Type: "page"},
			Title:                 "Test",
			Excerpt:               longExcerpt,
			ResultGlobalContainer: api.SearchContainer{Title: "Space"},
		}

		art := ProjectSearchResult(resultLong, artifact.Agent)

		// Should be truncated to 200 runes + "..."
		testutil.Equal(t, 203, len(art.Excerpt))
		testutil.True(t, strings.HasSuffix(art.Excerpt, "..."))
	})

	t.Run("does not truncate excerpt in full mode", func(t *testing.T) {
		longExcerpt := strings.Repeat("a", 250)
		resultLong := &api.SearchResult{
			Content:               api.SearchContent{ID: "1", Type: "page"},
			Title:                 "Test",
			Excerpt:               longExcerpt,
			ResultGlobalContainer: api.SearchContainer{Title: "Space"},
		}

		art := ProjectSearchResult(resultLong, artifact.Full)

		testutil.Equal(t, 250, len(art.Excerpt))
	})

	t.Run("handles multi-byte UTF-8 in excerpt truncation", func(t *testing.T) {
		// 200 runes of multi-byte characters
		multiByteExcerpt := strings.Repeat("日", 250)
		resultMultiByte := &api.SearchResult{
			Content:               api.SearchContent{ID: "1", Type: "page"},
			Title:                 "Test",
			Excerpt:               multiByteExcerpt,
			ResultGlobalContainer: api.SearchContainer{Title: "Space"},
		}

		art := ProjectSearchResult(resultMultiByte, artifact.Agent)

		runes := []rune(art.Excerpt)
		// 200 runes + 3 for "..."
		testutil.Equal(t, 203, len(runes))
		testutil.True(t, strings.HasSuffix(art.Excerpt, "..."))
	})

	t.Run("empty excerpt is omitted", func(t *testing.T) {
		resultNoExcerpt := &api.SearchResult{
			Content:               api.SearchContent{ID: "1", Type: "page"},
			Title:                 "Test",
			Excerpt:               "",
			ResultGlobalContainer: api.SearchContainer{Title: "Space"},
		}

		art := ProjectSearchResult(resultNoExcerpt, artifact.Agent)
		testutil.Equal(t, "", art.Excerpt)
	})
}

func TestProjectSearchResults(t *testing.T) {
	t.Parallel()

	results := []api.SearchResult{
		{
			Content:               api.SearchContent{ID: "1", Type: "page"},
			Title:                 "Page 1",
			ResultGlobalContainer: api.SearchContainer{Title: "Space A"},
		},
		{
			Content:               api.SearchContent{ID: "2", Type: "blogpost"},
			Title:                 "Blog Post 1",
			ResultGlobalContainer: api.SearchContainer{Title: "Space B"},
		},
	}

	arts := ProjectSearchResults(results, artifact.Agent)

	testutil.Equal(t, 2, len(arts))
	testutil.Equal(t, "1", arts[0].ID)
	testutil.Equal(t, "page", arts[0].Type)
	testutil.Equal(t, "2", arts[1].ID)
	testutil.Equal(t, "blogpost", arts[1].Type)
}
