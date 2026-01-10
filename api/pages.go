package api

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
	Status     string // current, archived, draft
	Sort       string // title, -title, created-date, -created-date, modified-date, -modified-date
	Title      string // Filter by title (contains)
	BodyFormat string // storage, atlas_doc_format, view
}

// GetPageOptions contains options for getting a page.
type GetPageOptions struct {
	BodyFormat string // storage, atlas_doc_format, view
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
		return nil, err
	}

	var result PaginatedResponse[Page]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse pages response: %w", err)
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
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("failed to parse page response: %w", err)
	}

	return &page, nil
}

// CreatePage creates a new page.
func (c *Client) CreatePage(ctx context.Context, req *CreatePageRequest) (*Page, error) {
	body, err := c.Post(ctx, "/api/v2/pages", req)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("failed to parse create page response: %w", err)
	}

	return &page, nil
}

// UpdatePage updates an existing page.
func (c *Client) UpdatePage(ctx context.Context, pageID string, req *UpdatePageRequest) (*Page, error) {
	path := fmt.Sprintf("/api/v2/pages/%s", pageID)
	body, err := c.Put(ctx, path, req)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("failed to parse update page response: %w", err)
	}

	return &page, nil
}

// DeletePage deletes a page.
func (c *Client) DeletePage(ctx context.Context, pageID string) error {
	path := fmt.Sprintf("/api/v2/pages/%s", pageID)
	_, err := c.Delete(ctx, path)
	return err
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
		return nil, err
	}

	var response v1PageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse copy response: %w", err)
	}

	return response.toPage(), nil
}
