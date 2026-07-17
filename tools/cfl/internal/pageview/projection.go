package pageview

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/atime"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/pkg/md"
)

// MaxChars is the default body truncation threshold for page view output.
const MaxChars = 5000

// Options controls page-view body projection.
type Options struct {
	BodyFormat  string
	NoTruncate  bool
	ShowMacros  bool
	ContentOnly bool
	Full        bool
}

// BodyKind identifies the body representation selected for presentation.
type BodyKind int

const (
	BodyKindNone BodyKind = iota
	BodyKindMarkdown
	BodyKindXHTML
	BodyKindADF
)

// Projection is the presenter-facing view model for page view output.
type Projection struct {
	Title       string
	ID          string
	SpaceKey    string
	SpaceID     string
	Version     int
	HasVersion  bool
	ParentID    string
	CreatedAt   *atime.AtlassianTime
	AuthorID    string
	Full        bool
	ContentOnly bool
	Body        string
	BodyKind    BodyKind
	HasContent  bool
	Truncated   bool
}

var (
	fromStorage = md.FromConfluenceStorageWithOptions
	fromADF     = md.FromADF
)

// OverrideConvertersForTest swaps the storage/ADF conversion hooks and returns
// a restore function. It exists so command tests can force conversion errors.
func OverrideConvertersForTest(
	storage func(string, md.ConvertOptions) (string, error),
	adf func(string) (string, error),
) func() {
	prevStorage := fromStorage
	prevADF := fromADF
	if storage != nil {
		fromStorage = storage
	}
	if adf != nil {
		fromADF = adf
	}
	return func() {
		fromStorage = prevStorage
		fromADF = prevADF
	}
}

// Project builds the presenter-facing page-view projection from API data and
// command mode flags.
func Project(page *api.Page, spaceKey string, opts Options) (Projection, error) {
	proj := Projection{
		Title:       page.Title,
		ID:          page.ID,
		SpaceKey:    spaceKey,
		SpaceID:     page.SpaceID,
		ParentID:    page.ParentID,
		CreatedAt:   page.CreatedAt,
		AuthorID:    page.AuthorID,
		Full:        opts.Full,
		ContentOnly: opts.ContentOnly,
	}
	if page.Version != nil {
		proj.Version = page.Version.Number
		proj.HasVersion = true
	}

	switch opts.BodyFormat {
	case "adf":
		if page.Body != nil && page.Body.AtlasDocFormat != nil {
			proj.Body, proj.Truncated = TruncateContent(page.Body.AtlasDocFormat.Value, opts)
			proj.BodyKind = BodyKindADF
			proj.HasContent = page.Body.AtlasDocFormat.Value != ""
		}
	case "xhtml":
		if page.Body != nil && page.Body.Storage != nil {
			proj.Body, proj.Truncated = TruncateContent(page.Body.Storage.Value, opts)
			proj.BodyKind = BodyKindXHTML
			proj.HasContent = page.Body.Storage.Value != ""
		}
	case "", "markdown":
		switch {
		case hasStorageContent(page):
			body, truncated, err := projectStorageBody(page.Body.Storage.Value, opts)
			if err != nil {
				return Projection{}, err
			}
			proj.Body, proj.Truncated = body, truncated
			proj.BodyKind = BodyKindMarkdown
			proj.HasContent = true
		case hasADFContent(page):
			body, truncated, err := projectADFBody(page.Body.AtlasDocFormat.Value, opts)
			if err != nil {
				return Projection{}, err
			}
			proj.Body, proj.Truncated = body, truncated
			proj.BodyKind = BodyKindMarkdown
			proj.HasContent = true
		}
	default:
		return Projection{}, fmt.Errorf("unsupported body format %q", opts.BodyFormat)
	}

	return proj, nil
}

func projectStorageBody(content string, opts Options) (body string, truncated bool, err error) {
	markdown, err := fromStorage(content, md.ConvertOptions{ShowMacros: opts.ShowMacros})
	if err != nil {
		return "", false, fmt.Errorf("failed to convert XHTML to markdown; rerun with --body-format xhtml or --body-format adf: %w", err)
	}
	body, truncated = TruncateContent(markdown, opts)
	return body, truncated, nil
}

func projectADFBody(content string, opts Options) (body string, truncated bool, err error) {
	markdown, err := fromADF(content)
	if err != nil {
		return "", false, fmt.Errorf("failed to convert ADF to markdown; rerun with --body-format adf or --body-format xhtml: %w", err)
	}
	body, truncated = TruncateContent(markdown, opts)
	return body, truncated, nil
}

// TruncateContent truncates content if it exceeds the character limit and
// reports whether truncation occurred.
func TruncateContent(content string, opts Options) (string, bool) {
	if opts.NoTruncate || opts.ContentOnly {
		return content, false
	}
	runes := []rune(content)
	if len(runes) > MaxChars {
		return string(runes[:MaxChars]), true
	}
	return content, false
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
