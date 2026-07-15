package page

import (
	"context"

	"github.com/open-cli-collective/confluence-cli/api"
)

// getPageWithBodyFallback fetches a page with body content, falling back to
// atlas_doc_format if storage format returns empty content. This handles
// ADF-native pages where the server-side ADF→XHTML conversion may fail
// silently, returning an empty storage body even though the page has content.
func getPageWithBodyFallback(ctx context.Context, client *api.Client, pageID string) (*api.Page, error) {
	page, err := client.GetPage(ctx, pageID, &api.GetPageOptions{
		BodyFormat: "storage",
	})
	if err != nil {
		return nil, err
	}

	if hasStorageContent(page) {
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

// getPageVersionWithBodyFallback fetches a specific page version with body
// content, falling back to atlas_doc_format when storage is empty.
func getPageVersionWithBodyFallback(ctx context.Context, client *api.Client, pageID string, version int) (*api.Page, error) {
	location, err := client.LocatePageVersion(ctx, pageID, version)
	if err != nil {
		return nil, err
	}

	page, err := client.GetLocatedPageVersion(ctx, pageID, location, "storage")
	if err != nil {
		return nil, err
	}

	if hasStorageContent(page) {
		return page, nil
	}

	adfPage, err := client.GetLocatedPageVersion(ctx, pageID, location, "atlas_doc_format")
	if err == nil && adfPage.Body != nil && adfPage.Body.AtlasDocFormat != nil {
		page.Body = adfPage.Body
	}

	return page, nil
}

// hasStorageContent returns true if the page has non-empty storage format content.
func hasStorageContent(page *api.Page) bool {
	return page.Body != nil &&
		page.Body.Storage != nil &&
		page.Body.Storage.Value != ""
}

// hasADFContent returns true if the page has non-empty ADF content.
func hasADFContent(page *api.Page) bool {
	return page.Body != nil &&
		page.Body.AtlasDocFormat != nil &&
		page.Body.AtlasDocFormat.Value != ""
}
