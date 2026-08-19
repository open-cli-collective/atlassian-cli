package api //nolint:revive // package name is intentional

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Export task states reported by Confluence. A task that is neither
// finished nor failed reports some other value while it runs; only these
// two decide the outcome.
const (
	ExportStateSucceeded = "SUCCEEDED"
	ExportStateFailed    = "FAILED"
)

// ErrExportTaskMissing reports that Confluence answered the export request
// without naming a task. The export cannot be polled, so there is nothing
// to wait for.
var ErrExportTaskMissing = errors.New("no export task in Confluence response")

// ErrExportFailed reports that Confluence ran the export and it did not
// succeed.
var ErrExportFailed = errors.New("export failed on the Confluence side")

// ErrExportNotPDF reports that the download returned something other than a
// PDF. A rejected credential is answered with a sign-in page carrying HTTP
// 200, so the status code alone does not establish that a document arrived.
var ErrExportNotPDF = errors.New("export download was not a PDF")

// pdfMagic opens every PDF document.
const pdfMagic = "%PDF-"

// PDFExport is a started export, identified by the task Confluence
// created for it.
type PDFExport struct {
	// TaskID addresses the task in the progress endpoint.
	TaskID string
	// V3 selects both the progress endpoint and how Result is read once
	// the task finishes. Confluence declares which generation served the
	// request; it is not a choice this client makes.
	V3 bool
}

// PDFExportProgress is one reading of an export task.
type PDFExportProgress struct {
	Progress int    `json:"progress"`
	State    string `json:"state"`
	// Result is empty until the task finishes. It is a download URL under
	// V3 and a URL yielding one otherwise.
	Result string `json:"result"`
	// EstimatedTimeRemaining is milliseconds, as Confluence reports it.
	EstimatedTimeRemaining int `json:"estimatedTimeRemaining"`
	TimeElapsed            int `json:"timeElapsed"`
}

// Done reports that the task will not progress further.
func (p *PDFExportProgress) Done() bool {
	return p.State == ExportStateSucceeded || p.Progress >= 100
}

// Failed reports that the task ended without producing a document.
func (p *PDFExportProgress) Failed() bool {
	return p.State == ExportStateFailed
}

var (
	metaTagRe     = regexp.MustCompile(`(?i)<meta\b[^>]*>`)
	metaNameRe    = regexp.MustCompile(`(?i)\bname="ajs-([a-zA-Z0-9_.-]+)"`)
	metaContentRe = regexp.MustCompile(`(?i)\bcontent="([^"]*)"`)
)

// ajsMeta extracts the ajs-* metadata Confluence embeds in the page it
// returns. Attribute order is the server's to choose, so name and content
// are read independently rather than as one fixed pattern.
func ajsMeta(html string) map[string]string {
	meta := make(map[string]string)
	for _, tag := range metaTagRe.FindAllString(html, -1) {
		name := metaNameRe.FindStringSubmatch(tag)
		if name == nil {
			continue
		}
		content := metaContentRe.FindStringSubmatch(tag)
		if content == nil {
			continue
		}
		meta[name[1]] = content[1]
	}
	return meta
}

// StartPDFExport asks Confluence to render a page as PDF and returns the
// task it created.
//
// Confluence Cloud exposes no JSON endpoint that starts this task. The
// browser navigates to an action that both starts the export and returns a
// progress page, and the task identifier is only ever published as metadata
// inside that page, so the identifier is read from the returned HTML. The
// XSRF header is what separates an accepted request from a 403.
func (c *Client) StartPDFExport(ctx context.Context, pageID string) (*PDFExport, error) {
	path := fmt.Sprintf("/spaces/flyingpdf/pdfpageexport.action?pageId=%s", url.QueryEscape(pageID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.GetBaseURL()+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating export request: %w", err)
	}
	req.Header.Set("Authorization", c.GetAuthHeader())
	req.Header.Set("X-Atlassian-Token", "no-check")

	resp, err := c.GetHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("starting export: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading export response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("starting export: page %s not found, or not visible to this user", pageID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("starting export: status %d", resp.StatusCode)
	}

	meta := ajsMeta(string(body))
	taskID := meta["taskId"]
	if taskID == "" {
		return nil, ErrExportTaskMissing
	}

	return &PDFExport{TaskID: taskID, V3: meta["isV3"] == "true"}, nil
}

// progressPath addresses the task in whichever progress endpoint served
// the export.
func (e *PDFExport) progressPath() string {
	if e.V3 {
		return "/api/v2/pdfexporttask/progress/" + e.TaskID
	}
	return "/services/api/v1/task/" + e.TaskID + "/progress"
}

// GetPDFExportProgress reads the current state of an export task.
func (c *Client) GetPDFExportProgress(ctx context.Context, export *PDFExport) (*PDFExportProgress, error) {
	body, err := c.Get(ctx, export.progressPath())
	if err != nil {
		return nil, fmt.Errorf("getting export progress: %w", err)
	}

	var progress PDFExportProgress
	if err := json.Unmarshal(body, &progress); err != nil {
		return nil, fmt.Errorf("parsing export progress response: %w", err)
	}

	return &progress, nil
}

// OpenPDFExport opens the finished document for reading.
//
// Under V3 the result is already the document URL. Otherwise it addresses a
// resource whose body is the URL to fetch, which is why the indirection is
// resolved here rather than by the caller.
func (c *Client) OpenPDFExport(ctx context.Context, export *PDFExport, progress *PDFExportProgress) (io.ReadCloser, error) {
	if progress.Result == "" {
		return nil, errors.New("export finished without a download URL")
	}

	downloadURL := progress.Result
	if !export.V3 {
		resolved, err := c.resolveExportDownloadURL(ctx, downloadURL)
		if err != nil {
			return nil, err
		}
		downloadURL = resolved
	}

	resp, err := c.getExportResource(ctx, downloadURL)
	if err != nil {
		return nil, fmt.Errorf("downloading export: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("downloading export: status %d", resp.StatusCode)
	}

	return pdfBody(resp.Body)
}

// pdfBody returns the response body only once it starts like a PDF, so a
// sign-in page served with HTTP 200 is not written out as a document. The
// inspected bytes stay in the stream.
func pdfBody(body io.ReadCloser) (io.ReadCloser, error) {
	buffered := bufio.NewReader(body)

	magic, err := buffered.Peek(len(pdfMagic))
	if err != nil && !errors.Is(err, io.EOF) {
		_ = body.Close()
		return nil, fmt.Errorf("reading export download: %w", err)
	}
	if string(magic) != pdfMagic {
		_ = body.Close()
		return nil, ErrExportNotPDF
	}

	return readCloser{Reader: buffered, Closer: body}, nil
}

// readCloser reads from the buffered view while closing the underlying
// response body.
type readCloser struct {
	io.Reader
	io.Closer
}

// resolveExportDownloadURL follows the pre-V3 indirection, where the task
// result names a resource whose body is the document URL.
func (c *Client) resolveExportDownloadURL(ctx context.Context, resultURL string) (string, error) {
	resp, err := c.getExportResource(ctx, resultURL)
	if err != nil {
		return "", fmt.Errorf("resolving export download URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading export download URL: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolving export download URL: status %d", resp.StatusCode)
	}

	resolved := strings.TrimSpace(strings.Trim(strings.TrimSpace(string(body)), `"`))
	if resolved == "" {
		return "", errors.New("export download URL was empty")
	}

	return resolved, nil
}

// getExportResource fetches a URL named by the export task.
//
// A task names its result either as a path on the site or as an absolute
// URL on the media host, and the two differ in what they accept: the media
// URL is signed and carries its own access, whereas a path on the site
// needs the caller's credential. Sending the credential is therefore
// decided by where the URL points, which also keeps it off any host it does
// not belong to.
func (c *Client) getExportResource(ctx context.Context, rawURL string) (*http.Response, error) {
	target, onSite, err := c.resolveExportURL(rawURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download request: %w", err)
	}
	if onSite {
		req.Header.Set("Authorization", c.GetAuthHeader())
	}

	resp, err := c.GetHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// resolveExportURL expands a task result against the configured site and
// reports whether it addresses that site.
func (c *Client) resolveExportURL(rawURL string) (string, bool, error) {
	base, err := url.Parse(c.GetBaseURL())
	if err != nil {
		return "", false, fmt.Errorf("parsing base URL: %w", err)
	}
	ref, err := url.Parse(rawURL)
	if err != nil {
		return "", false, fmt.Errorf("parsing export URL: %w", err)
	}

	resolved := base.ResolveReference(ref)

	return resolved.String(), base.Host != "" && strings.EqualFold(base.Host, resolved.Host), nil
}
