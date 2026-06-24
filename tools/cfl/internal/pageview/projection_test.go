package pageview

import (
	"fmt"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/pkg/md"
)

func TestProject_DefaultStorageMarkdown(t *testing.T) {
	t.Parallel()

	page := &api.Page{
		ID:      "12345",
		Title:   "Test Page",
		SpaceID: "98765",
		Version: &api.Version{Number: 3},
		Body: &api.Body{
			Storage: &api.BodyRepresentation{Value: "<p>Hello <strong>World</strong></p>"},
		},
	}

	expectedBody, err := md.FromConfluenceStorageWithOptions(
		"<p>Hello <strong>World</strong></p>",
		md.ConvertOptions{},
	)
	testutil.RequireNoError(t, err)

	proj := Project(page, "TEST", Options{})

	testutil.Equal(t, Projection{
		Title:       "Test Page",
		ID:          "12345",
		SpaceKey:    "TEST",
		SpaceID:     "98765",
		Version:     3,
		HasVersion:  true,
		ContentOnly: false,
		Body:        expectedBody,
	}, proj)
}

func TestProject_ContentOnlyRawStorage(t *testing.T) {
	t.Parallel()

	proj := Project(&api.Page{
		Body: &api.Body{
			Storage: &api.BodyRepresentation{Value: "<p>Raw HTML</p>"},
		},
	}, "", Options{Raw: true, ContentOnly: true})

	testutil.Equal(t, "<p>Raw HTML</p>", proj.Body)
	testutil.Equal(t, "", proj.Advisory)
	testutil.True(t, proj.ContentOnly)
}

func TestProject_ADFConversionFallback(t *testing.T) {
	t.Parallel()

	proj := Project(&api.Page{
		Body: &api.Body{
			AtlasDocFormat: &api.BodyRepresentation{Value: "{not-json"},
		},
	}, "", Options{})

	testutil.Equal(t, "{not-json", proj.Body)
	testutil.Equal(t, "(Failed to convert ADF to markdown, showing raw ADF)", proj.Advisory)
}

func TestProject_EmptyContent(t *testing.T) {
	t.Parallel()

	proj := Project(&api.Page{
		ID:    "12345",
		Title: "Empty Page",
	}, "", Options{})

	testutil.Equal(t, "(No content)", proj.Body)
	testutil.Equal(t, "", proj.Advisory)
}

func TestTruncateContent(t *testing.T) {
	t.Parallel()

	short := TruncateContent("short", Options{})
	testutil.Equal(t, "short", short)

	long := strings.Repeat("x", MaxChars+10)
	truncated := TruncateContent(long, Options{})
	testutil.Contains(t, truncated, fmt.Sprintf("... [truncated at %d chars, use --no-truncate for complete text]", MaxChars))

	full := TruncateContent(long, Options{NoTruncate: true})
	testutil.Equal(t, long, full)

	contentOnly := TruncateContent(long, Options{ContentOnly: true})
	testutil.Equal(t, long, contentOnly)
}
