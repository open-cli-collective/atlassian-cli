package pageview

import (
	"errors"
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
		BodyKind:    BodyKindMarkdown,
		Fallback:    FallbackNone,
		HasContent:  true,
		Truncated:   false,
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
	testutil.True(t, proj.ContentOnly)
	testutil.Equal(t, BodyKindStorageRaw, proj.BodyKind)
	testutil.Equal(t, FallbackNone, proj.Fallback)
	testutil.True(t, proj.HasContent)
}

func TestProject_ADFConversionFallback(t *testing.T) {
	t.Parallel()

	proj := Project(&api.Page{
		Body: &api.Body{
			AtlasDocFormat: &api.BodyRepresentation{Value: "{not-json"},
		},
	}, "", Options{})

	testutil.Equal(t, "{not-json", proj.Body)
	testutil.Equal(t, BodyKindADFRaw, proj.BodyKind)
	testutil.Equal(t, FallbackADFRaw, proj.Fallback)
	testutil.True(t, proj.HasContent)
}

func TestProject_StorageConversionFallback(t *testing.T) {
	restore := OverrideConvertersForTest(func(string, md.ConvertOptions) (string, error) {
		return "", errors.New("boom")
	}, nil)
	defer restore()

	proj := Project(&api.Page{
		Body: &api.Body{
			Storage: &api.BodyRepresentation{Value: "<p>Raw fallback</p>"},
		},
	}, "", Options{})

	testutil.Equal(t, "<p>Raw fallback</p>", proj.Body)
	testutil.Equal(t, BodyKindStorageRaw, proj.BodyKind)
	testutil.Equal(t, FallbackStorageRaw, proj.Fallback)
	testutil.True(t, proj.HasContent)
}

func TestProject_EmptyContent(t *testing.T) {
	t.Parallel()

	proj := Project(&api.Page{
		ID:    "12345",
		Title: "Empty Page",
	}, "", Options{})

	testutil.Equal(t, "", proj.Body)
	testutil.Equal(t, BodyKindNone, proj.BodyKind)
	testutil.Equal(t, FallbackNone, proj.Fallback)
	testutil.False(t, proj.HasContent)
}

func TestTruncateContent(t *testing.T) {
	t.Parallel()

	short, shortTruncated := TruncateContent("short", Options{})
	testutil.Equal(t, "short", short)
	testutil.False(t, shortTruncated)

	long := strings.Repeat("x", MaxChars+10)
	truncated, wasTruncated := TruncateContent(long, Options{})
	testutil.Equal(t, strings.Repeat("x", MaxChars), truncated)
	testutil.True(t, wasTruncated)

	full, fullTruncated := TruncateContent(long, Options{NoTruncate: true})
	testutil.Equal(t, long, full)
	testutil.False(t, fullTruncated)

	contentOnly, contentOnlyTruncated := TruncateContent(long, Options{ContentOnly: true})
	testutil.Equal(t, long, contentOnly)
	testutil.False(t, contentOnlyTruncated)
}
