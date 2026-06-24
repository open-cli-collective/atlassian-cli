package pageview

import (
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

// BodyKind identifies the body representation selected for presentation.
type BodyKind int

const (
	BodyKindNone BodyKind = iota
	BodyKindMarkdown
	BodyKindStorageRaw
	BodyKindADFRaw
)

// FallbackKind identifies whether conversion fell back to a raw/source-faithful body.
type FallbackKind int

const (
	FallbackNone FallbackKind = iota
	FallbackStorageRaw
	FallbackADFRaw
)

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
	BodyKind    BodyKind
	Fallback    FallbackKind
	HasContent  bool
	Truncated   bool
}

var (
	fromStorage = md.FromConfluenceStorageWithOptions
	fromADF     = md.FromADF
)

// OverrideConvertersForTest swaps the storage/ADF conversion hooks and returns
// a restore function. It exists so higher-level command tests can force the
// fallback branches without depending on fragile converter internals.
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
		proj.Body, proj.BodyKind, proj.Fallback, proj.Truncated = projectStorageBody(page.Body.Storage.Value, opts)
		proj.HasContent = true
	case hasADFContent(page):
		proj.Body, proj.BodyKind, proj.Fallback, proj.Truncated = projectADFBody(page.Body.AtlasDocFormat.Value, opts)
		proj.HasContent = true
	default:
		proj.HasContent = false
	}

	return proj
}

func projectStorageBody(content string, opts Options) (body string, kind BodyKind, fallback FallbackKind, truncated bool) {
	if opts.Raw {
		body, truncated = TruncateContent(content, opts)
		return body, BodyKindStorageRaw, FallbackNone, truncated
	}

	markdown, err := fromStorage(content, md.ConvertOptions{ShowMacros: opts.ShowMacros})
	if err != nil {
		body, truncated = TruncateContent(content, opts)
		return body, BodyKindStorageRaw, FallbackStorageRaw, truncated
	}
	body, truncated = TruncateContent(markdown, opts)
	return body, BodyKindMarkdown, FallbackNone, truncated
}

func projectADFBody(content string, opts Options) (body string, kind BodyKind, fallback FallbackKind, truncated bool) {
	if opts.Raw {
		body, truncated = TruncateContent(content, opts)
		return body, BodyKindADFRaw, FallbackNone, truncated
	}

	markdown, err := fromADF(content)
	if err != nil {
		body, truncated = TruncateContent(content, opts)
		return body, BodyKindADFRaw, FallbackADFRaw, truncated
	}
	body, truncated = TruncateContent(markdown, opts)
	return body, BodyKindMarkdown, FallbackNone, truncated
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
