package page

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func init() {
	// Polling is exercised by every export test; none should wait in real
	// time between reads.
	pollInterval = time.Millisecond
}

const exportProgressPath = "/api/v2/pdfexporttask/progress/module-11111111-2222-3333-4444-555555555555"

func loadExportFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name)) //nolint:gosec // reading test fixture data
	testutil.RequireNoError(t, err)
	return data
}

// mockExportServer serves the three legs of an export: the action that
// starts the task, the progress reads, and the document itself. runsBefore
// controls how many reads report the task still running, and download, when
// set, replaces the handler for the document.
func mockExportServer(t *testing.T, runsBefore int, download http.HandlerFunc) *httptest.Server {
	t.Helper()
	page := loadExportFixture(t, "export_page.json")
	start := loadExportFixture(t, "export_start.html")
	running := loadExportFixture(t, "export_progress_running.json")
	succeeded := loadExportFixture(t, "export_progress_succeeded.json")
	document := loadExportFixture(t, "export_document.pdf")

	var mu sync.Mutex
	var polls int
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/pages/123456":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(page)
		case "/spaces/flyingpdf/pdfpageexport.action":
			w.Header().Set("Content-Type", "text/html;charset=UTF-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(start)
		case exportProgressPath:
			mu.Lock()
			stillRunning := polls < runsBefore
			if stillRunning {
				polls++
			}
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			if stillRunning {
				_, _ = w.Write(running)
				return
			}
			_, _ = w.Write(succeeded)
		case "/download/export.pdf":
			if download != nil {
				download(w, r)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(document)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// truncatedDownload announces a full document, sends only its opening bytes
// and drops the connection, as a download cut off midway does.
func truncatedDownload(t *testing.T) http.HandlerFunc {
	t.Helper()
	document := loadExportFixture(t, "export_document.pdf")
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(document)*100))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(document[:10])
		w.(http.Flusher).Flush()
		conn, _, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Errorf("hijacking connection: %v", err)
			return
		}
		_ = conn.Close()
	}
}

func newExportTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func newExportTestOptions(t *testing.T, server *httptest.Server) *exportOptions {
	t.Helper()
	rootOpts := newExportTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(server.URL, "user@example.com", "token"))
	return &exportOptions{
		Options: rootOpts,
		format:  exportFormatPDF,
		timeout: 30 * time.Second,
	}
}

func TestRunExport_Success(t *testing.T) {
	server := mockExportServer(t, 0, nil)
	defer server.Close()
	want := loadExportFixture(t, "export_document.pdf")

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	opts := newExportTestOptions(t, server)

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "Exported: Quarterly Handoff.pdf\nSize: 26 B\n", opts.Stdout.(*bytes.Buffer).String())

	content, err := os.ReadFile(filepath.Join(tmpDir, "Quarterly Handoff.pdf")) //nolint:gosec // reading test output file
	testutil.RequireNoError(t, err)
	testutil.Equal(t, string(want), string(content))
}

func TestRunExport_CustomOutputFile(t *testing.T) {
	t.Parallel()
	server := mockExportServer(t, 0, nil)
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "handoff.pdf")
	opts := newExportTestOptions(t, server)
	opts.outputFile = outputPath

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "Exported: "+outputPath+"\nSize: 26 B\n", opts.Stdout.(*bytes.Buffer).String())

	content, err := os.ReadFile(outputPath) //nolint:gosec // reading test output file
	testutil.RequireNoError(t, err)
	testutil.Equal(t, string(loadExportFixture(t, "export_document.pdf")), string(content))
}

// TestRunExport_ReportsProgress pins that a wait is visible. Confluence
// renders server-side, so silence for that stretch is indistinguishable
// from a hang.
func TestRunExport_ReportsProgress(t *testing.T) {
	t.Parallel()
	server := mockExportServer(t, 1, nil)
	defer server.Close()

	opts := newExportTestOptions(t, server)
	opts.outputFile = filepath.Join(t.TempDir(), "handoff.pdf")

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, opts.Stderr.(*bytes.Buffer).String(), "Exporting: 0% complete")
	// Progress belongs on stderr so stdout carries only the artifact.
	testutil.NotContains(t, opts.Stdout.(*bytes.Buffer).String(), "Exporting")
}

func TestRunExport_FileExists_NoForce(t *testing.T) {
	t.Parallel()
	server := mockExportServer(t, 0, nil)
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "handoff.pdf")
	testutil.RequireNoError(t, os.WriteFile(outputPath, []byte("existing content"), 0600))

	opts := newExportTestOptions(t, server)
	opts.outputFile = outputPath

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireError(t, err)
	testutil.ErrorContains(t, err, "file already exists")
	testutil.ErrorContains(t, err, "--force")

	content, _ := os.ReadFile(outputPath) //nolint:gosec // reading test fixture file
	testutil.Equal(t, "existing content", string(content))
}

func TestRunExport_FileExists_WithForce(t *testing.T) {
	t.Parallel()
	server := mockExportServer(t, 0, nil)
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "handoff.pdf")
	testutil.RequireNoError(t, os.WriteFile(outputPath, []byte("existing content"), 0600))

	opts := newExportTestOptions(t, server)
	opts.outputFile = outputPath
	opts.force = true

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireNoError(t, err)

	content, _ := os.ReadFile(outputPath) //nolint:gosec // reading test output file
	testutil.Equal(t, string(loadExportFixture(t, "export_document.pdf")), string(content))
}

func TestRunExport_InvalidFormat(t *testing.T) {
	t.Parallel()
	opts := &exportOptions{Options: newExportTestRootOptions(), format: "docx", timeout: time.Minute}

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireError(t, err)
	testutil.ErrorContains(t, err, `invalid export format: "docx"`)
	testutil.ErrorContains(t, err, "valid formats: pdf")
}

func TestRunExport_InvalidTimeout(t *testing.T) {
	t.Parallel()
	opts := &exportOptions{Options: newExportTestRootOptions(), format: exportFormatPDF}

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireError(t, err)
	testutil.ErrorContains(t, err, "invalid --timeout")
}

func TestRunExport_TimeoutWaitingForRender(t *testing.T) {
	t.Parallel()
	start := loadExportFixture(t, "export_start.html")
	running := loadExportFixture(t, "export_progress_running.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/spaces/flyingpdf/pdfpageexport.action":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(start)
		default:
			// The task never finishes.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(running)
		}
	}))
	defer server.Close()

	opts := newExportTestOptions(t, server)
	opts.outputFile = filepath.Join(t.TempDir(), "handoff.pdf")
	opts.timeout = 50 * time.Millisecond

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireError(t, err)
	testutil.ErrorContains(t, err, "--timeout")
}

func TestRunExport_ExportFailed(t *testing.T) {
	t.Parallel()
	start := loadExportFixture(t, "export_start.html")
	failed := loadExportFixture(t, "export_progress_failed.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/spaces/flyingpdf/pdfpageexport.action":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(start)
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(failed)
		}
	}))
	defer server.Close()

	opts := newExportTestOptions(t, server)
	opts.outputFile = filepath.Join(t.TempDir(), "handoff.pdf")

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireError(t, err)
	testutil.ErrorContains(t, err, "export failed")
}

// TestRunExport_PollsUntilDone pins that several running reads are waited
// through well inside a timeout the production interval could not meet.
func TestRunExport_PollsUntilDone(t *testing.T) {
	t.Parallel()
	server := mockExportServer(t, 5, nil)
	defer server.Close()

	opts := newExportTestOptions(t, server)
	opts.outputFile = filepath.Join(t.TempDir(), "handoff.pdf")
	opts.timeout = time.Second

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireNoError(t, err)
}

// TestRunExport_DownloadCutOff pins that a download failing midway leaves
// nothing at the destination, so a retry without --force is not refused by
// a truncated document.
func TestRunExport_DownloadCutOff(t *testing.T) {
	t.Parallel()
	server := mockExportServer(t, 0, truncatedDownload(t))
	defer server.Close()

	dir := t.TempDir()
	opts := newExportTestOptions(t, server)
	opts.outputFile = filepath.Join(dir, "handoff.pdf")

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireError(t, err)
	testutil.ErrorContains(t, err, "writing file")

	entries, err := os.ReadDir(dir)
	testutil.RequireNoError(t, err)
	testutil.Len(t, entries, 0)

	retry := mockExportServer(t, 0, nil)
	defer retry.Close()
	retryOpts := newExportTestOptions(t, retry)
	retryOpts.outputFile = opts.outputFile

	testutil.RequireNoError(t, runExport(context.Background(), "123456", retryOpts))
	content, err := os.ReadFile(opts.outputFile) //nolint:gosec // reading test output file
	testutil.RequireNoError(t, err)
	testutil.Equal(t, string(loadExportFixture(t, "export_document.pdf")), string(content))
}

// TestRunExport_DownloadCutOff_KeepsExisting pins that with --force a failed
// download does not destroy the file it would have replaced.
func TestRunExport_DownloadCutOff_KeepsExisting(t *testing.T) {
	t.Parallel()
	server := mockExportServer(t, 0, truncatedDownload(t))
	defer server.Close()

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "handoff.pdf")
	testutil.RequireNoError(t, os.WriteFile(outputPath, []byte("existing content"), 0600))

	opts := newExportTestOptions(t, server)
	opts.outputFile = outputPath
	opts.force = true

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireError(t, err)

	content, err := os.ReadFile(outputPath) //nolint:gosec // reading test fixture file
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "existing content", string(content))
	entries, err := os.ReadDir(dir)
	testutil.RequireNoError(t, err)
	testutil.Len(t, entries, 1)
}

// TestExportFilename covers titles that are free text: they carry path
// separators and characters a filename cannot hold, and must still resolve
// to a single file in the working directory.
func TestExportFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		title  string
		pageID string
		want   string
	}{
		{"plain title", "Quarterly Handoff", "123", "Quarterly Handoff.pdf"},
		{"path separators", "Runbook: DB/Restore", "123", "Runbook- DB-Restore.pdf"},
		{"path traversal", "../../../etc/passwd", "123", "..-..-..-etc-passwd.pdf"},
		{"leading slash", "/etc/passwd", "123", "-etc-passwd.pdf"},
		{"empty title", "", "123", "123.pdf"},
		{"whitespace title", "   ", "123", "123.pdf"},
		{"dot title", ".", "123", "123.pdf"},
		{"double dot title", "..", "123", "123.pdf"},
		{"reserved characters", `Report <2026> "final"?`, "123", "Report -2026- -final--.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := exportFilename(tt.title, tt.pageID)
			testutil.Equal(t, tt.want, got)
			// Whatever the title held, the result names one file here.
			testutil.Equal(t, got, filepath.Base(got))
		})
	}
}
