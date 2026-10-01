package pageview

import (
	"context"

	"github.com/open-cli-collective/confluence-cli/api"
)

// GetPageWithBodyFallback fetches a page with body content, falling back to
// atlas_doc_format if storage format returns empty content. This handles
// ADF-native pages where the server-side ADF→XHTML conversion may fail
// silently, returning an empty storage body even though the page has content.
//
// A failed ADF request is not an error: the storage response is returned, so a
// caller that can show a page without its body still works. A caller that
// needs the body must check HasStorageContent and HasADFContent itself.
func GetPageWithBodyFallback(ctx context.Context, client *api.Client, pageID string) (*api.Page, error) {
	page, err := client.GetPage(ctx, pageID, &api.GetPageOptions{
		BodyFormat: "storage",
	})
	if err != nil {
		return nil, err
	}

	if HasStorageContent(page) {
		return page, nil
	}

	// Fallback: try atlas_doc_format for ADF-native pages.
	adfPage, err := client.GetPage(ctx, pageID, &api.GetPageOptions{
		BodyFormat: "atlas_doc_format",
	})
	if err == nil && adfPage.Body != nil && adfPage.Body.AtlasDocFormat != nil {
		page.Body = adfPage.Body
	}

	return page, nil
}

// GetPageVersionWithBodyFallback fetches a specific page version with body
// content, falling back to atlas_doc_format when storage is empty. A failed
// ADF request is tolerated the same way as in GetPageWithBodyFallback.
func GetPageVersionWithBodyFallback(ctx context.Context, client *api.Client, pageID string, version int) (*api.Page, error) {
	location, err := client.LocatePageVersion(ctx, pageID, version)
	if err != nil {
		return nil, err
	}

	page, err := client.GetLocatedPageVersion(ctx, pageID, location, "storage")
	if err != nil {
		return nil, err
	}

	if HasStorageContent(page) {
		return page, nil
	}

	adfPage, err := client.GetLocatedPageVersion(ctx, pageID, location, "atlas_doc_format")
	if err == nil && adfPage.Body != nil && adfPage.Body.AtlasDocFormat != nil {
		page.Body = adfPage.Body
	}

	return page, nil
}
