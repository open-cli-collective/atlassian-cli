package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
)

// ListAttachmentsOptions contains options for listing attachments.
type ListAttachmentsOptions struct {
	Limit     int
	Cursor    string
	MediaType string
	Filename  string
}

// ListAttachments returns attachments for a page.
func (c *Client) ListAttachments(ctx context.Context, pageID string, opts *ListAttachmentsOptions) (*PaginatedResponse[Attachment], error) {
	params := url.Values{}
	params.Set("limit", "25")

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.MediaType != "" {
			params.Set("mediaType", opts.MediaType)
		}
		if opts.Filename != "" {
			params.Set("filename", opts.Filename)
		}
	}

	path := fmt.Sprintf("/api/v2/pages/%s/attachments?%s", pageID, params.Encode())
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result PaginatedResponse[Attachment]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse attachments response: %w", err)
	}

	return &result, nil
}

// GetAttachment returns a single attachment by ID.
func (c *Client) GetAttachment(ctx context.Context, attachmentID string) (*Attachment, error) {
	path := fmt.Sprintf("/api/v2/attachments/%s", attachmentID)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var att Attachment
	if err := json.Unmarshal(body, &att); err != nil {
		return nil, fmt.Errorf("failed to parse attachment response: %w", err)
	}

	return &att, nil
}

// DownloadAttachment downloads an attachment and returns a reader.
func (c *Client) DownloadAttachment(ctx context.Context, attachmentID string) (io.ReadCloser, error) {
	// First, get the download URL
	path := fmt.Sprintf("/api/v2/attachments/%s/download", attachmentID)

	// Create a client that doesn't follow redirects
	noRedirectClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.email, c.apiToken)

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Handle redirect
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect {
		redirectURL := resp.Header.Get("Location")
		_ = resp.Body.Close()

		// If redirect is relative, make it absolute
		if redirectURL[0] == '/' {
			redirectURL = c.baseURL + redirectURL
		}

		req, err = http.NewRequestWithContext(ctx, "GET", redirectURL, nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(c.email, c.apiToken)

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// UploadAttachment uploads a file as an attachment to a page.
// Note: This uses the v1 API as v2 doesn't support uploads yet.
func (c *Client) UploadAttachment(ctx context.Context, pageID, filename string, content io.Reader, comment string) (*Attachment, error) {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file part
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, content); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add comment if provided
	if comment != "" {
		if err := writer.WriteField("comment", comment); err != nil {
			return nil, fmt.Errorf("failed to write comment field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Use v1 API for uploads
	path := fmt.Sprintf("/rest/api/content/%s/child/attachment", pageID)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, &buf)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "nocheck") // Required for XSRF protection

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(respBody))
		}
		return nil, &errResp
	}

	// v1 API returns results in a different format
	var result struct {
		Results []Attachment `json:"results"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse upload response: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("no attachment returned from upload")
	}

	return &result.Results[0], nil
}
