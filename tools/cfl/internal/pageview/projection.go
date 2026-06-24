package pageview

import (
	"fmt"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/pkg/md"
)

// MaxChars is the default body truncation threshold for page view output.
const MaxChars = 5000

// Options controls page-view body projection.
type Options struct {
	Raw         bool
	NoTruncate  bool
	ShowMacros  bool
	ContentOnly bool
}

// Projection is the presenter-facing view model for page view output.
type Projection struct {
	Title       string
	ID          string
	SpaceKey    string
	SpaceID     string
	Version     int
	HasVersion  bool
	ContentOnly bool
	Body        string
	Advisory    string
}

// Project builds the presenter-facing page-view projection from API data and
// command mode flags.
func Project(page *api.Page, spaceKey string, opts Options) Projection {
	proj := Projection{
		Title:       page.Title,
		ID:          page.ID,
		SpaceKey:    spaceKey,
		SpaceID:     page.SpaceID,
		ContentOnly: opts.ContentOnly,
	}
	if page.Version != nil {
		proj.Version = page.Version.Number
		proj.HasVersion = true
	}

	switch {
	case hasStorageContent(page):
		proj.Body, proj.Advisory = projectStorageBody(page.Body.Storage.Value, opts)
	case hasADFContent(page):
		proj.Body, proj.Advisory = projectADFBody(page.Body.AtlasDocFormat.Value, opts)
	default:
		proj.Body = "(No content)"
	}

	return proj
}

func projectStorageBody(content string, opts Options) (body string, advisory string) {
	if opts.Raw {
		return TruncateContent(content, opts), ""
	}

	markdown, err := md.FromConfluenceStorageWithOptions(content, md.ConvertOptions{
		ShowMacros: opts.ShowMacros,
	})
	if err != nil {
		return TruncateContent(content, opts), "(Failed to convert to markdown, showing raw HTML)"
	}
	return TruncateContent(markdown, opts), ""
}

func projectADFBody(content string, opts Options) (body string, advisory string) {
	if opts.Raw {
		return TruncateContent(content, opts), ""
	}

	markdown, err := md.FromADF(content)
	if err != nil {
		return TruncateContent(content, opts), "(Failed to convert ADF to markdown, showing raw ADF)"
	}
	return TruncateContent(markdown, opts), ""
}

// TruncateContent truncates content if it exceeds the character limit.
// Uses rune count to avoid splitting multi-byte UTF-8 characters.
// --content-only implies --no-truncate since it is intended for piping.
func TruncateContent(content string, opts Options) string {
	if opts.NoTruncate || opts.ContentOnly {
		return content
	}
	runes := []rune(content)
	if len(runes) > MaxChars {
		return string(runes[:MaxChars]) + fmt.Sprintf("\n\n... [truncated at %d chars, use --no-truncate for complete text]", MaxChars)
	}
	return content
}

func hasStorageContent(page *api.Page) bool {
	return page.Body != nil &&
		page.Body.Storage != nil &&
		page.Body.Storage.Value != ""
}

func hasADFContent(page *api.Page) bool {
	return page.Body != nil &&
		page.Body.AtlasDocFormat != nil &&
		page.Body.AtlasDocFormat.Value != ""
}
