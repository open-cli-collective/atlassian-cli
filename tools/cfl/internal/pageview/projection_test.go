package pageview

import (
	"errors"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/pkg/md"
)

func TestProject_BodyFormats(t *testing.T) {
	t.Parallel()
	page := &api.Page{
		ID: "12345", Title: "Test Page", SpaceID: "98765", Version: &api.Version{Number: 3},
		Body: &api.Body{
			Storage:        &api.BodyRepresentation{Value: "<p>Hello</p>"},
			AtlasDocFormat: &api.BodyRepresentation{Value: `{"type":"doc","version":1,"content":[]}`},
		},
	}

	markdown, err := Project(page, "TEST", Options{BodyFormat: "markdown"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "Hello", markdown.Body)
	testutil.Equal(t, BodyKindMarkdown, markdown.BodyKind)
	omitted, err := Project(page, "TEST", Options{})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, markdown, omitted)

	xhtml, err := Project(page, "TEST", Options{BodyFormat: "xhtml", ContentOnly: true})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "<p>Hello</p>", xhtml.Body)
	testutil.Equal(t, BodyKindXHTML, xhtml.BodyKind)

	adf, err := Project(page, "TEST", Options{BodyFormat: "adf", ContentOnly: true})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, `{"type":"doc","version":1,"content":[]}`, adf.Body)
	testutil.Equal(t, BodyKindADF, adf.BodyKind)
}

func TestProject_MarkdownConversionFailsClosed(t *testing.T) {
	restore := OverrideConvertersForTest(func(string, md.ConvertOptions) (string, error) {
		return "", errors.New("boom")
	}, nil)
	defer restore()

	proj, err := Project(&api.Page{Body: &api.Body{
		Storage: &api.BodyRepresentation{Value: "<p>must not leak</p>"},
	}}, "", Options{BodyFormat: "markdown"})
	testutil.RequireError(t, err)
	testutil.Equal(t, Projection{}, proj)
	testutil.Contains(t, err.Error(), "--body-format xhtml")
}

func TestProject_EmptyContent(t *testing.T) {
	t.Parallel()
	proj, err := Project(&api.Page{ID: "12345", Title: "Empty Page"}, "", Options{})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "", proj.Body)
	testutil.Equal(t, BodyKindNone, proj.BodyKind)
	testutil.False(t, proj.HasContent)
}

func TestProject_ExactEmptyBodies(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		format string
		body   *api.Body
		kind   BodyKind
	}{
		{"ADF", BodyFormatADF, &api.Body{AtlasDocFormat: &api.BodyRepresentation{}}, BodyKindADF},
		{"XHTML", BodyFormatXHTML, &api.Body{Storage: &api.BodyRepresentation{}}, BodyKindXHTML},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proj, err := Project(&api.Page{Body: tt.body}, "", Options{BodyFormat: tt.format, ContentOnly: true})
			testutil.RequireNoError(t, err)
			testutil.Equal(t, "", proj.Body)
			testutil.Equal(t, tt.kind, proj.BodyKind)
			testutil.True(t, proj.HasContent)
		})
	}
}

func TestTruncateContent(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", MaxChars+10)
	truncated, wasTruncated := TruncateContent(long, Options{})
	testutil.Equal(t, strings.Repeat("x", MaxChars), truncated)
	testutil.True(t, wasTruncated)

	full, fullTruncated := TruncateContent(long, Options{ContentOnly: true})
	testutil.Equal(t, long, full)
	testutil.False(t, fullTruncated)
}
