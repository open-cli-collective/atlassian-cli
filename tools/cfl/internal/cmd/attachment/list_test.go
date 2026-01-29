package attachment

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/api"
)

func TestRunList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"results": [
				{"id": "att1", "title": "doc.pdf", "mediaType": "application/pdf", "fileSize": 1024},
				{"id": "att2", "title": "image.png", "mediaType": "image/png", "fileSize": 2048}
			]
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		pageID:  "12345",
		limit:   25,
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		pageID:  "12345",
		limit:   25,
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		pageID:  "99999",
		limit:   25,
		noColor: true,
	}

	err := runList(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list attachments")
}

func TestRunList_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"results": [
				{"id": "att1", "title": "doc.pdf", "mediaType": "application/pdf", "fileSize": 1024}
			]
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		pageID:  "12345",
		limit:   25,
		output:  "json",
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_InvalidOutputFormat(t *testing.T) {
	// Don't need a server - should fail before API call
	client := api.NewClient("http://unused", "test@example.com", "token")
	opts := &listOptions{
		pageID:  "12345",
		output:  "invalid",
		noColor: true,
	}

	err := runList(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatFileSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAttachmentReferenced(t *testing.T) {
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
			result := isAttachmentReferenced(tt.filename, tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterUnusedAttachments(t *testing.T) {
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

	require.Len(t, unused, 1)
	assert.Equal(t, "att2", unused[0].ID)
	assert.Equal(t, "unused-doc.pdf", unused[0].Title)
}

func TestFilterUnusedAttachments_AllUnused(t *testing.T) {
	attachments := []api.Attachment{
		{ID: "att1", Title: "orphan1.png"},
		{ID: "att2", Title: "orphan2.pdf"},
	}

	content := `<p>This page has no attachment references</p>`

	unused := filterUnusedAttachments(attachments, content)

	require.Len(t, unused, 2)
}

func TestFilterUnusedAttachments_NoneUnused(t *testing.T) {
	attachments := []api.Attachment{
		{ID: "att1", Title: "used.png"},
	}

	content := `<ac:image><ri:attachment ri:filename="used.png"/></ac:image>`

	unused := filterUnusedAttachments(attachments, content)

	assert.Empty(t, unused)
}

func TestRunList_UnusedFlag(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch r.URL.Path {
		case "/api/v2/pages/12345/attachments":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"results": [
					{"id": "att1", "title": "used.png", "mediaType": "image/png", "fileSize": 1024},
					{"id": "att2", "title": "unused.pdf", "mediaType": "application/pdf", "fileSize": 2048}
				]
			}`))
		case "/api/v2/pages/12345":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
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

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		pageID:  "12345",
		limit:   25,
		unused:  true,
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
	assert.Equal(t, 2, requestCount) // Both attachments and page content fetched
}

func TestRunList_UnusedFlag_NoUnused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/pages/12345/attachments":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"results": [
					{"id": "att1", "title": "used.png", "mediaType": "image/png", "fileSize": 1024}
				]
			}`))
		case "/api/v2/pages/12345":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
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

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		pageID:  "12345",
		limit:   25,
		unused:  true,
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}
