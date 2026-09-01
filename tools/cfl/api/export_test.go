package api //nolint:revive // package name is intentional

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

const exportTaskID = "module-11111111-2222-3333-4444-555555555555"

func TestClient_StartPDFExport(t *testing.T) {
	t.Parallel()
	fixture := loadTestData(t, "pdf_export_start.html")

	var gotAuth, gotToken, gotPath, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotToken = r.Header.Get("X-Atlassian-Token")
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("pageId")
		w.Header().Set("Content-Type", "text/html;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	export, err := client.StartPDFExport(context.Background(), "123456")

	testutil.RequireNoError(t, err)
	testutil.Equal(t, exportTaskID, export.TaskID)
	testutil.True(t, export.V3, "fixture declares isV3")
	testutil.Equal(t, "/spaces/flyingpdf/pdfpageexport.action", gotPath)
	testutil.Equal(t, "123456", gotQuery)
	testutil.NotEmpty(t, gotAuth)
	// Without the XSRF header Confluence refuses the request with 403.
	testutil.Equal(t, "no-check", gotToken)
}

func TestClient_StartPDFExport_PageNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("<html><body>not found</body></html>"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	_, err := client.StartPDFExport(context.Background(), "999999999")

	testutil.RequireError(t, err)
	testutil.ErrorContains(t, err, "999999999")
	testutil.ErrorContains(t, err, "not found")
}

func TestClient_StartPDFExport_NoTaskID(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><head><meta name="ajs-page-id" content="1"></head></html>`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	_, err := client.StartPDFExport(context.Background(), "123456")

	testutil.RequireError(t, err)
	testutil.True(t, errors.Is(err, ErrExportTaskMissing), "want ErrExportTaskMissing, got "+err.Error())
}

// TestAJSMeta_AttributeOrder pins that metadata is read regardless of the
// order the server writes the attributes in.
func TestAJSMeta_AttributeOrder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		html string
	}{
		{"name first", `<meta name="ajs-taskId" content="task-1">`},
		{"content first", `<meta content="task-1" name="ajs-taskId">`},
		{"extra attributes", `<meta id="x" name="ajs-taskId" data-a="b" content="task-1">`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testutil.Equal(t, "task-1", ajsMeta(tt.html)["taskId"])
		})
	}
}

func TestPDFExport_ProgressPath(t *testing.T) {
	t.Parallel()
	v3 := &PDFExport{TaskID: "task-1", V3: true}
	testutil.Equal(t, "/api/v2/pdfexporttask/progress/task-1", v3.progressPath())

	legacy := &PDFExport{TaskID: "task-1"}
	testutil.Equal(t, "/services/api/v1/task/task-1/progress", legacy.progressPath())
}

func TestClient_GetPDFExportProgress(t *testing.T) {
	t.Parallel()
	running := loadTestData(t, "pdf_export_progress_running.json")
	succeeded := loadTestData(t, "pdf_export_progress_succeeded.json")

	tests := []struct {
		name      string
		fixture   []byte
		wantDone  bool
		wantState string
	}{
		{"running", running, false, "IN_PROGRESS"},
		{"succeeded", succeeded, true, ExportStateSucceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				testutil.Equal(t, "/api/v2/pdfexporttask/progress/"+exportTaskID, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(tt.fixture)
			}))
			defer server.Close()

			client := NewClient(server.URL, "user@example.com", "token")
			progress, err := client.GetPDFExportProgress(context.Background(), &PDFExport{TaskID: exportTaskID, V3: true})

			testutil.RequireNoError(t, err)
			testutil.Equal(t, tt.wantState, progress.State)
			testutil.Equal(t, tt.wantDone, progress.Done())
			testutil.False(t, progress.Failed())
		})
	}
}

func TestPDFExportProgress_Failed(t *testing.T) {
	t.Parallel()
	progress := &PDFExportProgress{State: ExportStateFailed, Progress: 40}
	testutil.True(t, progress.Failed(), "FAILED state")
	testutil.False(t, progress.Done())
}

func TestClient_OpenPDFExport_SignedURLGetsNoCredential(t *testing.T) {
	t.Parallel()
	// The media host stands in for the signed URL Confluence hands back.
	var mediaAuth string
	var mediaCalled bool
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaCalled = true
		mediaAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-1.4\nbody"))
	}))
	defer media.Close()

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("the site must not be called for an absolute result URL")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer site.Close()

	client := NewClient(site.URL, "user@example.com", "token")
	reader, err := client.OpenPDFExport(
		context.Background(),
		&PDFExport{TaskID: exportTaskID, V3: true},
		&PDFExportProgress{Result: media.URL + "/file/abc/binary?token=signed"},
	)
	testutil.RequireNoError(t, err)
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "%PDF-1.4\nbody", string(content))
	testutil.True(t, mediaCalled, "media host was called")
	testutil.Equal(t, "", mediaAuth)
}

func TestClient_OpenPDFExport_SiteURLGetsCredential(t *testing.T) {
	t.Parallel()
	var gotAuth string
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/download/export/page.pdf", r.URL.Path)
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-1.4\nbody"))
	}))
	defer site.Close()

	client := NewClient(site.URL, "user@example.com", "token")
	reader, err := client.OpenPDFExport(
		context.Background(),
		&PDFExport{TaskID: exportTaskID, V3: true},
		// A result naming a path on the site rather than an absolute URL.
		&PDFExportProgress{Result: "/download/export/page.pdf"},
	)
	testutil.RequireNoError(t, err)
	defer func() { _ = reader.Close() }()

	_, err = io.ReadAll(reader)
	testutil.RequireNoError(t, err)
	testutil.NotEmpty(t, gotAuth)
}

// TestClient_OpenPDFExport_RejectsNonPDF pins the failure that a status
// code cannot catch: a refused credential is answered with a sign-in page
// carrying HTTP 200, which would otherwise be written out as the document.
func TestClient_OpenPDFExport_RejectsNonPDF(t *testing.T) {
	t.Parallel()
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body>Log in to continue</body></html>"))
	}))
	defer site.Close()

	client := NewClient(site.URL, "user@example.com", "token")
	_, err := client.OpenPDFExport(
		context.Background(),
		&PDFExport{TaskID: exportTaskID, V3: true},
		&PDFExportProgress{Result: "/download/export/page.pdf"},
	)

	testutil.RequireError(t, err)
	testutil.True(t, errors.Is(err, ErrExportNotPDF), "want ErrExportNotPDF, got "+err.Error())
}

func TestClient_OpenPDFExport_NoResult(t *testing.T) {
	t.Parallel()
	client := NewClient("https://example.atlassian.net/wiki", "user@example.com", "token")
	_, err := client.OpenPDFExport(
		context.Background(),
		&PDFExport{TaskID: exportTaskID, V3: true},
		&PDFExportProgress{Progress: 100, State: ExportStateSucceeded},
	)

	testutil.RequireError(t, err)
	testutil.ErrorContains(t, err, "download URL")
}

// TestClient_OpenPDFExport_LegacyIndirection covers the pre-V3 shape, where
// the task result addresses a resource whose body is the document URL.
func TestClient_OpenPDFExport_LegacyIndirection(t *testing.T) {
	t.Parallel()
	var documentServed bool
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/download/link":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("/download/export/page.pdf\n"))
		case "/download/export/page.pdf":
			documentServed = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("%PDF-1.4\nbody"))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer site.Close()

	client := NewClient(site.URL, "user@example.com", "token")
	reader, err := client.OpenPDFExport(
		context.Background(),
		&PDFExport{TaskID: exportTaskID},
		&PDFExportProgress{Result: "/download/link"},
	)
	testutil.RequireNoError(t, err)
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	testutil.RequireNoError(t, err)
	testutil.True(t, documentServed, "document was fetched through the indirection")
	testutil.True(t, strings.HasPrefix(string(content), pdfMagic), "content is a PDF")
}
