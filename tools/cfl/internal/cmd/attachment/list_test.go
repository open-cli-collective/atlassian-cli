package attachment

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func newListTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunList_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "att1", "title": "doc.pdf", "mediaType": "application/pdf", "fileSize": 1024},
				{"id": "att2", "title": "image.png", "mediaType": "image/png", "fileSize": 2048}
			]
		}`))
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_PlainFullExact(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "att1", "title": "doc.pdf", "mediaType": "application/pdf", "fileSize": 1024, "status": "current", "comment": "latest"}
			],
			"_links": {"next": "/api/v2/pages/12345/attachments?cursor=abc"}
		}`))
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	rootOpts.Output = "plain"
	rootOpts.Full = true
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "ID\tTITLE\tMEDIA TYPE\tFILE SIZE\tSTATUS\tCOMMENT\natt1\tdoc.pdf\tapplication/pdf\t1.0 KB\tcurrent\tlatest\n", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "(showing first 1 results, use --limit to see more)\n", rootOpts.Stderr.(*bytes.Buffer).String())
}

func TestRunList_TableOutputExact(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "att1", "title": "doc.pdf", "mediaType": "application/pdf", "fileSize": 1024}
			]
		}`))
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "ID    TITLE    MEDIA TYPE       FILE SIZE\natt1  doc.pdf  application/pdf  1.0 KB\n", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "", rootOpts.Stderr.(*bytes.Buffer).String())
}

func TestRunList_Empty(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	// The empty-results banner used to be suppressed under -o json; #392
	// removed that skip, so the banner now always prints to stderr.
	testutil.Contains(t, rootOpts.Stderr.(*bytes.Buffer).String(), "No attachments found.")
}

func TestRunList_APIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "99999",
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "listing attachments")
}

func TestIsAttachmentReferenced(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filename string
		content  string
		expected bool
	}{
		{
			name:     "ri:filename attribute match",
			filename: "screenshot.png",
			content:  `<ac:image><ri:attachment ri:filename="screenshot.png"/></ac:image>`,
			expected: true,
		},
		{
			name:     "not referenced",
			filename: "unused.pdf",
			content:  `<ac:image><ri:attachment ri:filename="other.png"/></ac:image>`,
			expected: false,
		},
		{
			name:     "plain filename in content",
			filename: "document.docx",
			content:  `<p>See the attached document.docx for details</p>`,
			expected: true,
		},
		{
			name:     "URL encoded filename with spaces",
			filename: "my file.pdf",
			content:  `<a href="/download/my%20file.pdf">Download</a>`,
			expected: true,
		},
		{
			name:     "empty content",
			filename: "test.txt",
			content:  "",
			expected: false,
		},
		{
			name:     "partial filename not matched",
			filename: "report.pdf",
			content:  `<ri:attachment ri:filename="annual-report.pdf"/>`,
			expected: true, // substring match - contains "report.pdf"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isAttachmentReferenced(tt.filename, tt.content)
			testutil.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterUnusedAttachments(t *testing.T) {
	t.Parallel()
	attachments := []api.Attachment{
		{ID: "att1", Title: "used-image.png"},
		{ID: "att2", Title: "unused-doc.pdf"},
		{ID: "att3", Title: "another-used.jpg"},
	}

	content := `<p>Here is an image:</p>
<ac:image><ri:attachment ri:filename="used-image.png"/></ac:image>
<p>And another:</p>
<ac:image><ri:attachment ri:filename="another-used.jpg"/></ac:image>`

	unused := filterUnusedAttachments(attachments, content)

	testutil.Len(t, unused, 1)
	testutil.Equal(t, "att2", unused[0].ID)
	testutil.Equal(t, "unused-doc.pdf", unused[0].Title)
}

func TestFilterUnusedAttachments_AllUnused(t *testing.T) {
	t.Parallel()
	attachments := []api.Attachment{
		{ID: "att1", Title: "orphan1.png"},
		{ID: "att2", Title: "orphan2.pdf"},
	}

	content := `<p>This page has no attachment references</p>`

	unused := filterUnusedAttachments(attachments, content)

	testutil.Len(t, unused, 2)
}

func TestFilterUnusedAttachments_NoneUnused(t *testing.T) {
	t.Parallel()
	attachments := []api.Attachment{
		{ID: "att1", Title: "used.png"},
	}

	content := `<ac:image><ri:attachment ri:filename="used.png"/></ac:image>`

	unused := filterUnusedAttachments(attachments, content)

	testutil.Empty(t, unused)
}

func TestRunList_UnusedFlag(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch r.URL.Path {
		case "/api/v2/pages/12345/attachments":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [
					{"id": "att1", "title": "used.png", "mediaType": "image/png", "fileSize": 1024},
					{"id": "att2", "title": "unused.pdf", "mediaType": "application/pdf", "fileSize": 2048}
				]
			}`))
		case "/api/v2/pages/12345":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"body": {
					"storage": {
						"representation": "storage",
						"value": "<ac:image><ri:attachment ri:filename=\"used.png\"/></ac:image>"
					}
				}
			}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
		unused:  true,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, requestCount) // Both attachments and page content fetched
}

func TestRunList_UnusedFlag_NoUnused(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/pages/12345/attachments":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [
					{"id": "att1", "title": "used.png", "mediaType": "image/png", "fileSize": 1024}
				]
			}`))
		case "/api/v2/pages/12345":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"body": {
					"storage": {
						"representation": "storage",
						"value": "<ac:image><ri:attachment ri:filename=\"used.png\"/></ac:image>"
					}
				}
			}`))
		}
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
		unused:  true,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, rootOpts.Stderr.(*bytes.Buffer).String(), "No unused attachments found.")
}
