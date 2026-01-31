package api

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	sharederrors "github.com/open-cli-collective/atlassian-go/errors"
)

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
func (c *Client) GetIssueAttachments(issueKey string) ([]Attachment, error) {
	if issueKey == "" {
		return nil, fmt.Errorf("issue key is required")
	}

	urlStr := fmt.Sprintf("%s/issue/%s?fields=attachment", c.BaseURL, issueKey)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var result struct {
		Fields struct {
			Attachment []Attachment `json:"attachment"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse attachments: %w", err)
	}

	return result.Fields.Attachment, nil
}

// GetAttachment returns metadata for a specific attachment
func (c *Client) GetAttachment(attachmentID string) (*Attachment, error) {
	if attachmentID == "" {
		return nil, fmt.Errorf("attachment ID is required")
	}

	urlStr := fmt.Sprintf("%s/attachment/%s", c.BaseURL, attachmentID)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var attachment Attachment
	if err := json.Unmarshal(body, &attachment); err != nil {
		return nil, fmt.Errorf("failed to parse attachment: %w", err)
	}

	return &attachment, nil
}

// AddAttachment uploads a file as an attachment to an issue
func (c *Client) AddAttachment(issueKey, filePath string) ([]Attachment, error) {
	if issueKey == "" {
		return nil, fmt.Errorf("issue key is required")
	}
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
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

		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			errChan <- fmt.Errorf("failed to create form file: %w", err)
			return
		}

		if _, err := io.Copy(part, file); err != nil {
			errChan <- fmt.Errorf("failed to copy file content: %w", err)
			return
		}
		errChan <- nil
	}()

	urlStr := fmt.Sprintf("%s/issue/%s/attachments", c.BaseURL, issueKey)

	req, err := http.NewRequest(http.MethodPost, urlStr, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.GetAuthHeader())
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	// Required header for attachment uploads
	req.Header.Set("X-Atlassian-Token", "no-check")

	if c.Verbose {
		fmt.Printf("→ POST %s\n", urlStr)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Wait for the write goroutine to finish
	if writeErr := <-errChan; writeErr != nil {
		return nil, writeErr
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if c.Verbose {
		fmt.Printf("← %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if resp.StatusCode >= 400 {
		return nil, sharederrors.ParseAPIError(resp.StatusCode, respBody)
	}

	var attachments []Attachment
	if err := json.Unmarshal(respBody, &attachments); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return attachments, nil
}

// DeleteAttachment deletes an attachment by ID
func (c *Client) DeleteAttachment(attachmentID string) error {
	if attachmentID == "" {
		return fmt.Errorf("attachment ID is required")
	}

	urlStr := fmt.Sprintf("%s/attachment/%s", c.BaseURL, attachmentID)
	_, err := c.delete(urlStr)
	return err
}

// DownloadAttachment downloads an attachment to the specified output path
func (c *Client) DownloadAttachment(attachment *Attachment, outputPath string) error {
	if attachment == nil {
		return fmt.Errorf("attachment is required")
	}
	if attachment.Content == "" {
		return fmt.Errorf("attachment has no content URL")
	}

	// Create the request
	req, err := http.NewRequest(http.MethodGet, attachment.Content, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.GetAuthHeader())

	if c.Verbose {
		fmt.Printf("→ GET %s\n", attachment.Content)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if c.Verbose {
		fmt.Printf("← %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
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
	file, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Copy the content
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
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
