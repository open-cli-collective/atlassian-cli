package attachment

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
)

// unusedFixtureServer serves page bodies by body-format and attachment pages
// by cursor from testdata/unused, and records each request it receives.
type unusedFixtureServer struct {
	t           *testing.T
	pageBodies  map[string]string
	attachments map[string]string

	mu       sync.Mutex
	requests []string
}

func (s *unusedFixtureServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.requests = append(s.requests, r.URL.Path+"?"+r.URL.RawQuery)
	s.mu.Unlock()

	var fixture string
	switch r.URL.Path {
	case "/api/v2/pages/12345":
		fixture = s.pageBodies[r.URL.Query().Get("body-format")]
	case "/api/v2/pages/12345/attachments":
		fixture = s.attachments[r.URL.Query().Get("cursor")]
	}
	if fixture == "" {
		s.t.Errorf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	data, err := os.ReadFile(filepath.Join("testdata", "unused", fixture)) //nolint:gosec // fixture names are test constants
	if err != nil {
		s.t.Errorf("reading fixture %s: %v", fixture, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func runUnusedList(t *testing.T, srv *unusedFixtureServer, limit int) (stdout, stderr string) {
	t.Helper()
	server := httptest.NewServer(srv)
	defer server.Close()

	rootOpts := newListTestRootOptions()
	rootOpts.Output = "plain"
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	err := runList(context.Background(), &listOptions{Options: rootOpts, pageID: "12345", limit: limit, unused: true})
	testutil.RequireNoError(t, err)
	return rootOpts.Stdout.(*bytes.Buffer).String(), rootOpts.Stderr.(*bytes.Buffer).String()
}

const unusedHeader = "ID\tTITLE\tMEDIA TYPE\tFILE SIZE\n"

func TestRunListUnused_ADFPageWithEmptyStorage(t *testing.T) {
	t.Parallel()
	srv := &unusedFixtureServer{
		t: t,
		pageBodies: map[string]string{
			"storage":          "adf_page_storage.json",
			"atlas_doc_format": "adf_page_adf.json",
		},
		attachments: map[string]string{"": "adf_attachments.json"},
	}

	stdout, stderr := runUnusedList(t, srv, 25)

	// diagram.png and spec.pdf are media nodes by file id. notes.txt is only
	// named in prose and by a media node from another page's collection.
	testutil.Equal(t, stdout, unusedHeader+"att3\tnotes.txt\ttext/plain\t512 B\n")
	testutil.Equal(t, stderr, "")
}

func TestRunListUnused_StoragePage(t *testing.T) {
	t.Parallel()
	srv := &unusedFixtureServer{
		t:           t,
		pageBodies:  map[string]string{"storage": "storage_page.json"},
		attachments: map[string]string{"": "storage_attachments.json"},
	}

	stdout, stderr := runUnusedList(t, srv, 25)

	// logo.png is referenced on another page, archive.zip only inside a code
	// block, and notes.txt only in prose.
	testutil.Equal(t, stdout, unusedHeader+
		"att3\tnotes.txt\ttext/plain\t512 B\n"+
		"att4\tlogo.png\timage/png\t1.0 KB\n"+
		"att5\tarchive.zip\tapplication/zip\t8.0 KB\n")
	testutil.Equal(t, stderr, "")
	testutil.Equal(t, srv.requests, []string{
		"/api/v2/pages/12345?body-format=storage",
		"/api/v2/pages/12345/attachments?limit=25",
	})
}

func TestRunListUnused_FilenameOnlyInProseIsUnused(t *testing.T) {
	t.Parallel()
	srv := &unusedFixtureServer{
		t:           t,
		pageBodies:  map[string]string{"storage": "prose_page.json"},
		attachments: map[string]string{"": "prose_attachments.json"},
	}

	stdout, _ := runUnusedList(t, srv, 25)

	testutil.Equal(t, stdout, unusedHeader+"att2\tnotes.txt\ttext/plain\t512 B\n")
}

func multipageServer(t *testing.T) *unusedFixtureServer {
	return &unusedFixtureServer{
		t:          t,
		pageBodies: map[string]string{"storage": "multipage_page.json"},
		attachments: map[string]string{
			"":         "multipage_attachments_1.json",
			"cursor-2": "multipage_attachments_2.json",
			"cursor-3": "multipage_attachments_3.json",
		},
	}
}

func TestRunListUnused_PagesThroughAllAttachments(t *testing.T) {
	t.Parallel()
	srv := multipageServer(t)

	stdout, stderr := runUnusedList(t, srv, 2)

	testutil.Equal(t, stdout, unusedHeader+
		"att3\tc.png\timage/png\t1.0 KB\n"+
		"att5\te.png\timage/png\t1.0 KB\n")
	testutil.Equal(t, stderr, "")
	testutil.Equal(t, srv.requests, []string{
		"/api/v2/pages/12345?body-format=storage",
		"/api/v2/pages/12345/attachments?limit=2",
		"/api/v2/pages/12345/attachments?cursor=cursor-2&limit=2",
		"/api/v2/pages/12345/attachments?cursor=cursor-3&limit=2",
	})
}

func TestRunListUnused_LimitCountsUnusedResults(t *testing.T) {
	t.Parallel()
	srv := multipageServer(t)

	stdout, stderr := runUnusedList(t, srv, 1)

	testutil.Equal(t, stdout, unusedHeader+"att3\tc.png\timage/png\t1.0 KB\n")
	testutil.Equal(t, stderr, "(showing first 1 results, use --limit to see more)\n")
}
