// Package api provides a Go client for the Jira REST API.
package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	sharederrors "github.com/open-cli-collective/atlassian-go/errors"
)

// quoteEscaper mirrors the escaping mime/multipart applies to Content-Disposition
// parameters, so an override filename containing a quote or backslash cannot break
// out of the header.
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// Attachment represents a Jira attachment
type Attachment struct {
	ID       FlexibleID `json:"id"`
	Filename string     `json:"filename"`
	Author   User       `json:"author"`
	Created  string     `json:"created"`
	Size     int64      `json:"size"`
	MimeType string     `json:"mimeType"`
	Content  string     `json:"content"` // URL to download the attachment
	Self     string     `json:"self"`
}

// FlexibleID handles Jira API inconsistency where IDs can be strings or numbers
type FlexibleID string

// UnmarshalJSON handles both string and number JSON values for IDs
func (f *FlexibleID) UnmarshalJSON(data []byte) error {
	// Check for null
	if string(data) == "null" {
		return fmt.Errorf("id cannot be null")
	}

	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexibleID(s)
		return nil
	}

	// Try number
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexibleID(fmt.Sprintf("%d", n))
		return nil
	}

	return fmt.Errorf("id must be string or number, got: %s", string(data))
}

// String returns the ID as a string
func (f FlexibleID) String() string {
	return string(f)
}

// GetIssueAttachments returns all attachments for an issue
func (c *Client) GetIssueAttachments(ctx context.Context, issueKey string) ([]Attachment, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s?fields=attachment", c.BaseURL, issueKey)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching issue attachments: %w", err)
	}

	var result struct {
		Fields struct {
			Attachment []Attachment `json:"attachment"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing attachments: %w", err)
	}

	return result.Fields.Attachment, nil
}

// GetAttachment returns metadata for a specific attachment
func (c *Client) GetAttachment(ctx context.Context, attachmentID string) (*Attachment, error) {
	if attachmentID == "" {
		return nil, ErrAttachmentIDRequired
	}

	urlStr := fmt.Sprintf("%s/attachment/%s", c.BaseURL, attachmentID)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching attachment: %w", err)
	}

	var attachment Attachment
	if err := json.Unmarshal(body, &attachment); err != nil {
		return nil, fmt.Errorf("parsing attachment: %w", err)
	}

	return &attachment, nil
}

// AddAttachment uploads a file as an attachment to an issue. When filename is
// non-empty it overrides the stored attachment name; otherwise the base name of
// filePath is used. The multipart part is tagged with a Content-Type derived from
// the stored name's extension so Jira treats images and PDFs as such (rendering
// thumbnails and inline previews) rather than as generic downloads.
func (c *Client) AddAttachment(ctx context.Context, issueKey, filePath, filename string) ([]Attachment, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}
	if filePath == "" {
		return nil, ErrFilePathRequired
	}

	storedName := filename
	if storedName == "" {
		storedName = filepath.Base(filePath)
	}

	file, err := os.Open(filePath) //nolint:gosec // CLI tool opens user-provided file paths
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// Write the file in a goroutine to avoid blocking
	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreatePart(attachmentPartHeader(storedName))
		if err != nil {
			errChan <- fmt.Errorf("creating form file: %w", err)
			return
		}

		if _, err := io.Copy(part, file); err != nil {
			errChan <- fmt.Errorf("copying file content: %w", err)
			return
		}
		errChan <- nil
	}()

	urlStr := fmt.Sprintf("%s/issue/%s/attachments", c.BaseURL, issueKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, pr)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.GetAuthHeader())
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	// Required header for attachment uploads
	req.Header.Set("X-Atlassian-Token", "no-check")

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "→ POST %s\n", urlStr)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("uploading attachment: %w", err)
	}
	defer resp.Body.Close()

	// Wait for the write goroutine to finish
	if writeErr := <-errChan; writeErr != nil {
		return nil, writeErr
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "← %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if resp.StatusCode >= 400 {
		return nil, sharederrors.ParseAPIError(resp.StatusCode, respBody)
	}

	var attachments []Attachment
	if err := json.Unmarshal(respBody, &attachments); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return attachments, nil
}

// attachmentPartHeader builds the multipart part header for an attachment upload,
// setting the Content-Type from the filename extension (falling back to
// application/octet-stream) so Jira records an accurate MIME type.
func attachmentPartHeader(filename string) textproto.MIMEHeader {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`, quoteEscaper.Replace(filename)))

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	h.Set("Content-Type", contentType)

	return h
}

// DeleteAttachment deletes an attachment by ID
func (c *Client) DeleteAttachment(ctx context.Context, attachmentID string) error {
	if attachmentID == "" {
		return ErrAttachmentIDRequired
	}

	urlStr := fmt.Sprintf("%s/attachment/%s", c.BaseURL, attachmentID)
	_, err := c.Delete(ctx, urlStr)
	if err != nil {
		return fmt.Errorf("deleting attachment %s: %w", attachmentID, err)
	}
	return nil
}

// DownloadAttachment downloads an attachment to the specified output path
func (c *Client) DownloadAttachment(ctx context.Context, attachment *Attachment, outputPath string) error {
	if attachment == nil {
		return ErrAttachmentRequired
	}
	if attachment.Content == "" {
		return ErrAttachmentContentMissing
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, attachment.Content, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.GetAuthHeader())

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "→ GET %s\n", attachment.Content)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading attachment: %w", err)
	}
	defer resp.Body.Close()

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "← %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return sharederrors.ParseAPIError(resp.StatusCode, body)
	}

	// Determine output file path
	outFile := outputPath
	if isDirectory(outputPath) {
		outFile = filepath.Join(outputPath, attachment.Filename)
	}

	// Create the output file
	file, err := os.Create(outFile) //nolint:gosec // CLI tool creates user-provided file paths
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer file.Close()

	// Copy the content
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// isDirectory checks if a path is a directory
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FormatFileSize returns a human-readable file size
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
