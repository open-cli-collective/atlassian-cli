package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ListPagesOptions contains options for listing pages.
type ListPagesOptions struct {
	Limit      int
	Cursor     string
	Status     string // current, archived, trashed, deleted
	Sort       string // title, -title, created-date, -created-date, modified-date, -modified-date
	Title      string // Filter by title (contains)
	BodyFormat string // storage, atlas_doc_format, view
}

// GetPageOptions contains options for getting a page.
type GetPageOptions struct {
	BodyFormat string // storage, atlas_doc_format, view
}

// ListPageVersionsOptions contains options for listing page versions.
type ListPageVersionsOptions struct {
	Limit      int
	Cursor     string
	Sort       string // modified-date, -modified-date
	BodyFormat string // storage, atlas_doc_format
}

// GetPageVersionOptions contains options for getting a page version body.
type GetPageVersionOptions struct {
	BodyFormat string // storage, atlas_doc_format
}

// PageVersionLocation identifies a specific page version in a sorted versions list.
type PageVersionLocation struct {
	Version     Version
	Cursor      string
	Sort        string
	CurrentPage *Page
}

// ListPages returns a list of pages in a space.
func (c *Client) ListPages(ctx context.Context, spaceID string, opts *ListPagesOptions) (*PaginatedResponse[Page], error) {
	params := url.Values{}
	params.Set("limit", "25") // Default limit

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if opts.Sort != "" {
			params.Set("sort", opts.Sort)
		}
		if opts.Title != "" {
			params.Set("title", opts.Title)
		}
		if opts.BodyFormat != "" {
			params.Set("body-format", opts.BodyFormat)
		}
	}

	path := fmt.Sprintf("/api/v2/spaces/%s/pages?%s", spaceID, params.Encode())
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("listing pages: %w", err)
	}

	var result PaginatedResponse[Page]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing pages response: %w", err)
	}

	return &result, nil
}

// GetPage returns a single page by ID.
func (c *Client) GetPage(ctx context.Context, pageID string, opts *GetPageOptions) (*Page, error) {
	params := url.Values{}
	if opts != nil && opts.BodyFormat != "" {
		params.Set("body-format", opts.BodyFormat)
	}

	path := fmt.Sprintf("/api/v2/pages/%s", pageID)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("getting page: %w", err)
	}

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parsing page response: %w", err)
	}

	return &page, nil
}

// ListPageVersions returns versions for a page.
func (c *Client) ListPageVersions(ctx context.Context, pageID string, opts *ListPageVersionsOptions) (*PaginatedResponse[Version], error) {
	params := url.Values{}
	params.Set("limit", "25") // Default limit

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.Sort != "" {
			params.Set("sort", opts.Sort)
		}
		if opts.BodyFormat != "" {
			params.Set("body-format", opts.BodyFormat)
		}
	}

	path := fmt.Sprintf("/api/v2/pages/%s/versions?%s", pageID, params.Encode())
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("listing page versions: %w", err)
	}

	var result PaginatedResponse[Version]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing page versions response: %w", err)
	}

	return &result, nil
}

// LocatePageVersion finds the cursor and sort order needed to retrieve a specific page version.
func (c *Client) LocatePageVersion(ctx context.Context, pageID string, version int) (*PageVersionLocation, error) {
	if version < 1 {
		return nil, fmt.Errorf("invalid page version: %d (must be >= 1)", version)
	}

	currentPage, err := c.GetPage(ctx, pageID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting current page: %w", err)
	}

	sort := "-modified-date"
	if currentPage != nil && currentPage.Version != nil && currentPage.Version.Number > 0 {
		currentVersion := currentPage.Version.Number
		if version > currentVersion {
			return nil, fmt.Errorf("page version %d is newer than current version %d", version, currentVersion)
		}
		if version-1 < currentVersion-version {
			sort = "modified-date"
		}
	}

	cursor := ""
	for {
		result, err := c.ListPageVersions(ctx, pageID, &ListPageVersionsOptions{
			Limit:  1,
			Cursor: cursor,
			Sort:   sort,
		})
		if err != nil {
			return nil, fmt.Errorf("locating page version: %w", err)
		}
		if len(result.Results) == 0 {
			return nil, fmt.Errorf("page version %d not found", version)
		}

		row := result.Results[0]
		if row.Number == version {
			row.Page = nil
			return &PageVersionLocation{
				Version:     row,
				Cursor:      cursor,
				Sort:        sort,
				CurrentPage: currentPage,
			}, nil
		}
		if sort == "-modified-date" && row.Number < version {
			return nil, fmt.Errorf("page version %d not found", version)
		}
		if sort == "modified-date" && row.Number > version {
			return nil, fmt.Errorf("page version %d not found", version)
		}

		nextCursor := cursorFromNextLink(result.Links.Next)
		if nextCursor == "" {
			return nil, fmt.Errorf("page version %d not found", version)
		}
		cursor = nextCursor
	}
}

// GetLocatedPageVersion returns the page body for a previously located page version.
func (c *Client) GetLocatedPageVersion(
	ctx context.Context,
	pageID string,
	location *PageVersionLocation,
	bodyFormat string,
) (*Page, error) {
	if location == nil {
		return nil, fmt.Errorf("page version location is required")
	}
	if bodyFormat == "" {
		bodyFormat = "storage"
	}

	result, err := c.ListPageVersions(ctx, pageID, &ListPageVersionsOptions{
		Limit:      1,
		Cursor:     location.Cursor,
		Sort:       location.Sort,
		BodyFormat: bodyFormat,
	})
	if err != nil {
		return nil, err
	}
	if len(result.Results) == 0 {
		return nil, fmt.Errorf("page version %d not found", location.Version.Number)
	}

	row := result.Results[0]
	if row.Number != location.Version.Number {
		return nil, fmt.Errorf("expected page version %d, got %d", location.Version.Number, row.Number)
	}
	if row.Page == nil {
		return nil, fmt.Errorf("page version %d response did not include page content", row.Number)
	}

	page := row.Page
	normalizeVersionedPage(page, pageID, row, location.CurrentPage)
	return page, nil
}

// GetPageVersion returns a specific page version body.
func (c *Client) GetPageVersion(ctx context.Context, pageID string, version int, opts *GetPageVersionOptions) (*Page, error) {
	location, err := c.LocatePageVersion(ctx, pageID, version)
	if err != nil {
		return nil, err
	}

	bodyFormat := "storage"
	if opts != nil && opts.BodyFormat != "" {
		bodyFormat = opts.BodyFormat
	}
	return c.GetLocatedPageVersion(ctx, pageID, location, bodyFormat)
}

func normalizeVersionedPage(page *Page, pageID string, version Version, currentPage *Page) {
	if page.ID == "" {
		page.ID = pageID
	}
	if currentPage != nil {
		if page.Title == "" {
			page.Title = currentPage.Title
		}
		if page.SpaceID == "" {
			page.SpaceID = currentPage.SpaceID
		}
		if page.Links.WebUI == "" {
			page.Links.WebUI = currentPage.Links.WebUI
		}
	}
	version.Page = nil
	page.Version = &version
}

func cursorFromNextLink(nextLink string) string {
	if nextLink == "" {
		return ""
	}
	parsed, err := url.Parse(nextLink)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("cursor")
}

// CreatePage creates a new page.
func (c *Client) CreatePage(ctx context.Context, req *CreatePageRequest) (*Page, error) {
	body, err := c.Post(ctx, "/api/v2/pages", req)
	if err != nil {
		return nil, fmt.Errorf("creating page: %w", err)
	}

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parsing create page response: %w", err)
	}

	return &page, nil
}

// UpdatePage updates an existing page.
func (c *Client) UpdatePage(ctx context.Context, pageID string, req *UpdatePageRequest) (*Page, error) {
	path := fmt.Sprintf("/api/v2/pages/%s", pageID)
	body, err := c.Put(ctx, path, req)
	if err != nil {
		return nil, fmt.Errorf("updating page: %w", err)
	}

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parsing update page response: %w", err)
	}

	return &page, nil
}

// DeletePage deletes a page.
func (c *Client) DeletePage(ctx context.Context, pageID string) error {
	path := fmt.Sprintf("/api/v2/pages/%s", pageID)
	_, err := c.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("deleting page %s: %w", pageID, err)
	}
	return nil
}

// MovePage moves a page to be a child of the target parent page.
// Uses the v1 REST API as v2 doesn't support page moves.
func (c *Client) MovePage(ctx context.Context, pageID, targetParentID string) error {
	path := fmt.Sprintf("/rest/api/content/%s/move/append/%s", pageID, targetParentID)
	_, err := c.Put(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("moving page %s to parent %s: %w", pageID, targetParentID, err)
	}
	return nil
}

// CopyPageOptions configures page copy behavior.
type CopyPageOptions struct {
	Title              string // Required: new page title
	DestinationSpace   string // Optional: target space key (defaults to same space)
	CopyAttachments    bool   // Default: true
	CopyPermissions    bool   // Default: true
	CopyProperties     bool   // Default: true
	CopyLabels         bool   // Default: true
	CopyCustomContents bool   // Default: true
}

// copyPageRequest is the v1 API request body for copying a page.
type copyPageRequest struct {
	CopyAttachments    bool            `json:"copyAttachments"`
	CopyPermissions    bool            `json:"copyPermissions"`
	CopyProperties     bool            `json:"copyProperties"`
	CopyLabels         bool            `json:"copyLabels"`
	CopyCustomContents bool            `json:"copyCustomContents"`
	Destination        copyDestination `json:"destination"`
	PageTitle          string          `json:"pageTitle"`
}

// copyDestination specifies where to copy a page.
type copyDestination struct {
	Type  string `json:"type"`  // "space" or "parent_page"
	Value string `json:"value"` // space key or page ID
}

// v1PageResponse represents the v1 API page response structure.
type v1PageResponse struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Title  string `json:"title"`
	Space  struct {
		ID   int    `json:"id"`
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"space"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Links struct {
		WebUI string `json:"webui"`
		Self  string `json:"self"`
	} `json:"_links"`
}

// toPage converts a v1 API response to a Page.
func (r *v1PageResponse) toPage() *Page {
	return &Page{
		ID:      r.ID,
		Status:  r.Status,
		Title:   r.Title,
		SpaceID: r.Space.Key,
		Version: &Version{Number: r.Version.Number},
		Links:   Links{WebUI: r.Links.WebUI},
	}
}

// CopyPage duplicates a page with a new title.
// Uses the v1 REST API: POST /rest/api/content/{id}/copy
//
// Note: Callers must explicitly set all copy flags. If not set, they default to false (Go zero value).
// The command layer handles default-to-true semantics via --no-* flags.
func (c *Client) CopyPage(ctx context.Context, pageID string, opts *CopyPageOptions) (*Page, error) {
	if opts == nil || opts.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	req := copyPageRequest{
		CopyAttachments:    opts.CopyAttachments,
		CopyPermissions:    opts.CopyPermissions,
		CopyProperties:     opts.CopyProperties,
		CopyLabels:         opts.CopyLabels,
		CopyCustomContents: opts.CopyCustomContents,
		PageTitle:          opts.Title,
		Destination: copyDestination{
			Type:  "space",
			Value: opts.DestinationSpace,
		},
	}

	path := fmt.Sprintf("/rest/api/content/%s/copy", pageID)
	body, err := c.Post(ctx, path, req)
	if err != nil {
		return nil, fmt.Errorf("copying page: %w", err)
	}

	var response v1PageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing copy response: %w", err)
	}

	return response.toPage(), nil
}
