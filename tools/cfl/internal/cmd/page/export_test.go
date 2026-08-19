package page

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

const exportPDFBody = "%PDF-1.4\nexported document"

// mockExportServer serves the three legs of an export: the action that
// starts the task, the progress reads, and the document itself. runsBefore
// controls how many reads report the task still running.
func mockExportServer(t *testing.T, runsBefore int) *httptest.Server {
	t.Helper()
	var polls int
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/pages/123456":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"123456","title":"Quarterly Handoff","spaceId":"789"}`))
		case "/spaces/flyingpdf/pdfpageexport.action":
			w.Header().Set("Content-Type", "text/html;charset=UTF-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><head>` +
				`<meta name="ajs-taskId" content="module-abc">` +
				`<meta name="ajs-isV3" content="true">` +
				`</head></html>`))
		case "/api/v2/pdfexporttask/progress/module-abc":
			w.WriteHeader(http.StatusOK)
			if polls < runsBefore {
				polls++
				_, _ = w.Write([]byte(`{"progress":0,"state":"IN_PROGRESS"}`))
				return
			}
			_, _ = w.Write([]byte(`{"progress":100,"state":"SUCCEEDED","result":"` + server.URL + `/download/export.pdf"}`))
		case "/download/export.pdf":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(exportPDFBody))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server
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
	server := mockExportServer(t, 0)
	defer server.Close()

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
	testutil.Equal(t, exportPDFBody, string(content))
}

func TestRunExport_CustomOutputFile(t *testing.T) {
	t.Parallel()
	server := mockExportServer(t, 0)
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "handoff.pdf")
	opts := newExportTestOptions(t, server)
	opts.outputFile = outputPath

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "Exported: "+outputPath+"\nSize: 26 B\n", opts.Stdout.(*bytes.Buffer).String())

	content, err := os.ReadFile(outputPath) //nolint:gosec // reading test output file
	testutil.RequireNoError(t, err)
	testutil.Equal(t, exportPDFBody, string(content))
}

// TestRunExport_ReportsProgress pins that a wait is visible. Confluence
// renders server-side, so silence for that stretch is indistinguishable
// from a hang.
func TestRunExport_ReportsProgress(t *testing.T) {
	t.Parallel()
	server := mockExportServer(t, 1)
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
	server := mockExportServer(t, 0)
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
	server := mockExportServer(t, 0)
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "handoff.pdf")
	testutil.RequireNoError(t, os.WriteFile(outputPath, []byte("existing content"), 0600))

	opts := newExportTestOptions(t, server)
	opts.outputFile = outputPath
	opts.force = true

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireNoError(t, err)

	content, _ := os.ReadFile(outputPath) //nolint:gosec // reading test output file
	testutil.Equal(t, exportPDFBody, string(content))
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/spaces/flyingpdf/pdfpageexport.action":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><head><meta name="ajs-taskId" content="module-abc">` +
				`<meta name="ajs-isV3" content="true"></head></html>`))
		default:
			// The task never finishes.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"progress":0,"state":"IN_PROGRESS"}`))
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/spaces/flyingpdf/pdfpageexport.action":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><head><meta name="ajs-taskId" content="module-abc">` +
				`<meta name="ajs-isV3" content="true"></head></html>`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"progress":40,"state":"FAILED"}`))
		}
	}))
	defer server.Close()

	opts := newExportTestOptions(t, server)
	opts.outputFile = filepath.Join(t.TempDir(), "handoff.pdf")

	err := runExport(context.Background(), "123456", opts)
	testutil.RequireError(t, err)
	testutil.ErrorContains(t, err, "export failed")
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
