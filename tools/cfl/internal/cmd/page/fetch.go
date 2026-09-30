package page

import (
	"context"
	"fmt"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/pageview"
)

func getPageWithBodyFormat(ctx context.Context, client *api.Client, pageID, bodyFormat string) (*api.Page, error) {
	if bodyFormat == bodyFormatMarkdown {
		return pageview.GetPageWithBodyFallback(ctx, client, pageID)
	}
	page, err := client.GetPage(ctx, pageID, &api.GetPageOptions{BodyFormat: apiBodyFormat(bodyFormat)})
	if err != nil {
		return nil, err
	}
	if !hasBodyRepresentation(page, bodyFormat) {
		return nil, fmt.Errorf("page does not provide the requested %s body representation", bodyFormat)
	}
	return page, nil
}

func getPageVersionWithBodyFormat(ctx context.Context, client *api.Client, pageID string, version int, bodyFormat string) (*api.Page, error) {
	if bodyFormat == bodyFormatMarkdown {
		return pageview.GetPageVersionWithBodyFallback(ctx, client, pageID, version)
	}
	location, err := client.LocatePageVersion(ctx, pageID, version)
	if err != nil {
		return nil, err
	}
	page, err := client.GetLocatedPageVersion(ctx, pageID, location, apiBodyFormat(bodyFormat))
	if err != nil {
		return nil, err
	}
	if !hasBodyRepresentation(page, bodyFormat) {
		return nil, fmt.Errorf("page version %d does not provide the requested %s body representation", version, bodyFormat)
	}
	return page, nil
}

func apiBodyFormat(bodyFormat string) string {
	if bodyFormat == bodyFormatADF {
		return "atlas_doc_format"
	}
	return "storage"
}

func hasBodyRepresentation(page *api.Page, bodyFormat string) bool {
	if page == nil || page.Body == nil {
		return false
	}
	if bodyFormat == bodyFormatADF {
		return page.Body.AtlasDocFormat != nil
	}
	return page.Body.Storage != nil
}

// fetchStorageBody returns a page's storage body, or an explanation of why it
// is unavailable. One place answers that question so callers cannot drift.
func fetchStorageBody(ctx context.Context, client *api.Client, pageID string) (body, reason string) {
	got, err := readStorageBody(ctx, client, pageID)
	switch {
	case err != nil:
		return "", err.Error()
	case got == "":
		return "", "the page returned no storage body"
	default:
		return got, ""
	}
}
