package links

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

// isolateCache points cache I/O at a temp directory and overrides the
// derived instance key directly (bypassing env-var parsing) so tests can
// run under t.Parallel() without the t.Setenv race. Matches the pattern
// used by the rest of the package.
func isolateCache(t *testing.T) {
	t.Helper()
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
}

func TestNewListCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newListCmd(opts)

	testutil.Equal(t, cmd.Use, "list <issue-key>")
	testutil.Equal(t, cmd.Short, "List links on an issue")
}

func TestRunList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fields": map[string]any{
				"issuelinks": []map[string]any{
					{
						"id":   "10001",
						"type": map[string]string{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
						"outwardIssue": map[string]any{
							"key":    "PROJ-456",
							"fields": map[string]string{"summary": "Blocked issue"},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "PROJ-123", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "PROJ-456")
	testutil.Contains(t, stdout.String(), "Blocks")
	// OutwardIssue is set → current issue is the inward side → show inward direction
	testutil.Contains(t, stdout.String(), "is blocked by")
}

func TestRunList_NoLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fields": map[string]any{
				"issuelinks": []any{},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "PROJ-123", "")
	testutil.RequireNoError(t, err)
}

func TestRunList_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fields": map[string]any{
				"issuelinks": []any{
					map[string]any{
						"id":   "17844",
						"type": map[string]any{"id": "10", "name": "Blocker", "inward": "is blocked by", "outward": "blocks"},
						"outwardIssue": map[string]any{
							"key":    "PROJ-2",
							"fields": map[string]any{"summary": "Target"},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "PROJ-123", "")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, stdout.String(), "17844\n")
}

func linkServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("linkServer: expected GET, got %s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fields": map[string]any{
				"issuelinks": []any{
					map[string]any{
						"id":   "17844",
						"type": map[string]any{"id": "10100", "name": "Blocker", "inward": "is blocked by", "outward": "blocks"},
						"outwardIssue": map[string]any{
							"key":    "PROJ-2",
							"fields": map[string]any{"summary": "Target", "status": map[string]any{"name": "Open"}},
						},
					},
				},
			},
		})
	}))
}

func TestRunList_Extended(t *testing.T) {
	t.Parallel()
	server := linkServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "PROJ-123", "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "TYPE_ID")
	testutil.Contains(t, out, "STATUS")
	testutil.Contains(t, out, "10100")
	testutil.Contains(t, out, "Open")
}

func TestRunList_FieldsProjection(t *testing.T) {
	t.Parallel()
	server := linkServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "PROJ-123", "TYPE,ISSUE")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	// LINK_ID is always present (Identity pin) even though not in --fields
	testutil.Contains(t, out, "LINK_ID")
	testutil.Contains(t, out, "TYPE")
	testutil.Contains(t, out, "ISSUE")
	testutil.NotContains(t, out, "SUMMARY")
	testutil.NotContains(t, out, "DIRECTION")
}

func TestRunList_FieldsWithJSON_Error(t *testing.T) {
	t.Parallel()
	server := linkServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "json", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "PROJ-123", "TYPE")
	if !errors.Is(err, jtkpresent.ErrFieldsWithJSON) {
		t.Fatalf("expected ErrFieldsWithJSON, got %v", err)
	}
}

func TestRunCreate(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issueLinkType":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issueLinkTypes": []map[string]string{
					{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
				},
			})
		case "/rest/api/3/issueLink":
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	isolateCache(t)
	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ-123", "PROJ-456", "Blocks")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Created")

	var req api.CreateIssueLinkRequest
	err = json.Unmarshal(capturedBody, &req)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, req.Type.Name, "Blocks")
	testutil.Equal(t, req.OutwardIssue.Key, "PROJ-123")
	testutil.Equal(t, req.InwardIssue.Key, "PROJ-456")
}

func TestRunCreate_InwardVerbSwapsDirection(t *testing.T) {
	// A `jtk links create A B --type "is blocked by"` call must create a link
	// where B blocks A — i.e., A is blocked by B. Since the Jira API always
	// sees the link type by canonical name, correctness comes from the
	// outward/inward issue ordering we post.
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issueLinkType":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issueLinkTypes": []map[string]string{
					{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
				},
			})
		case "/rest/api/3/issueLink":
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	isolateCache(t)
	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	// User perspective: PROJ-1 IS BLOCKED BY PROJ-2.
	err = runCreate(context.Background(), opts, "PROJ-1", "PROJ-2", "is blocked by")
	testutil.RequireNoError(t, err)

	var req api.CreateIssueLinkRequest
	err = json.Unmarshal(capturedBody, &req)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, req.Type.Name, "Blocks")
	// Swapped: PROJ-2 blocks PROJ-1 → outward=PROJ-2, inward=PROJ-1.
	testutil.Equal(t, req.OutwardIssue.Key, "PROJ-2")
	testutil.Equal(t, req.InwardIssue.Key, "PROJ-1")
}

func TestRunCreate_OutwardVerbPreservesOrder(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issueLinkType":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issueLinkTypes": []map[string]string{
					{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
				},
			})
		case "/rest/api/3/issueLink":
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	isolateCache(t)
	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ-1", "PROJ-2", "blocks")
	testutil.RequireNoError(t, err)

	var req api.CreateIssueLinkRequest
	err = json.Unmarshal(capturedBody, &req)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, req.OutwardIssue.Key, "PROJ-1")
	testutil.Equal(t, req.InwardIssue.Key, "PROJ-2")
}

func TestRunCreate_SymmetricVerbNoSwap(t *testing.T) {
	// "Relates" has identical inward/outward verbs ("relates to"). A match
	// against the inward verb must NOT swap, because the outward side also
	// matches and the swap would invert the semantically-symmetric link.
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issueLinkType":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issueLinkTypes": []map[string]string{
					{"id": "2", "name": "Relates", "inward": "relates to", "outward": "relates to"},
				},
			})
		case "/rest/api/3/issueLink":
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	isolateCache(t)
	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ-1", "PROJ-2", "relates to")
	testutil.RequireNoError(t, err)

	var req api.CreateIssueLinkRequest
	err = json.Unmarshal(capturedBody, &req)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, req.OutwardIssue.Key, "PROJ-1")
	testutil.Equal(t, req.InwardIssue.Key, "PROJ-2")
}

func createServerWithRefetch(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issueLinkType":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issueLinkTypes": []map[string]string{
					{"id": "10100", "name": "Blocker", "inward": "is blocked by", "outward": "blocks"},
				},
			})
		case r.URL.Path == "/rest/api/3/issueLink" && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
		case r.URL.Path == "/rest/api/3/issue/PROJ-123":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"fields": map[string]any{
					"issuelinks": []map[string]any{
						{
							"id":   "17844",
							"type": map[string]any{"id": "10100", "name": "Blocker", "inward": "is blocked by", "outward": "blocks"},
							"inwardIssue": map[string]any{
								"key":    "PROJ-456",
								"fields": map[string]any{"summary": "Blocked issue", "status": map[string]any{"name": "Open"}},
							},
						},
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func seedLinkTypesForTest(t *testing.T) {
	t.Helper()
	isolateCache(t)
	testutil.RequireNoError(t, cache.WriteResource("linktypes", "24h", []api.IssueLinkType{
		{ID: "10100", Name: "Blocker", Inward: "is blocked by", Outward: "blocks"},
	}))
}

func TestRunCreate_CanonicalRow(t *testing.T) {
	t.Parallel()
	server := createServerWithRefetch(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	seedLinkTypesForTest(t)
	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ-123", "PROJ-456", "Blocker")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	testutil.Contains(t, lines[0], "LINK_ID")
	testutil.Contains(t, lines[0], "TYPE")
	testutil.Contains(t, lines[0], "DIRECTION")
	testutil.Contains(t, lines[0], "ISSUE")
	testutil.Contains(t, out, "17844")
	testutil.Contains(t, out, "Blocker")
	testutil.Contains(t, out, "PROJ-456")
	testutil.NotContains(t, out, "Created")
}

func TestRunCreate_IDOnly(t *testing.T) {
	t.Parallel()
	server := createServerWithRefetch(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	seedLinkTypesForTest(t)
	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ-123", "PROJ-456", "Blocker")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "17844\n")
}

func TestRunCreate_InvalidType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issueLinkTypes": []map[string]string{
				{"id": "1", "name": "Blocks"},
				{"id": "2", "name": "Relates"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	isolateCache(t)
	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "A", "B", "InvalidType")
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "InvalidType")
	testutil.Contains(t, err.Error(), "not found")
}

func TestRunDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issueLink/10001")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "10001")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted link 10001\n")
}

func TestRunDelete_JSONOutputEmitsText(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "10001")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted link 10001\n")
	testutil.Equal(t, stderr.String(), "")
}

func TestRunTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issueLinkTypes": []map[string]string{
				{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
				{"id": "2", "name": "Relates", "inward": "relates to", "outward": "relates to"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runTypes(context.Background(), opts, "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Blocks")
	testutil.Contains(t, stdout.String(), "Relates")
}

func TestRunTypes_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issueLinkTypes": []map[string]string{
				{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
				{"id": "2", "name": "Relates", "inward": "relates to", "outward": "relates to"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runTypes(context.Background(), opts, "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "1\n2\n")
}

func TestFindCreatedLink_MatchByName(t *testing.T) {
	t.Parallel()
	links := []api.IssueLink{
		{ID: "100", Type: api.IssueLinkType{ID: "1", Name: "Blocks"}, InwardIssue: &api.LinkedIssue{Key: "PROJ-456"}},
		{ID: "200", Type: api.IssueLinkType{ID: "2", Name: "Relates"}, InwardIssue: &api.LinkedIssue{Key: "PROJ-789"}},
	}
	resolved := api.IssueLinkType{ID: "1", Name: "Blocks"}
	got := findCreatedLink(links, resolved, "PROJ-456")
	testutil.NotNil(t, got)
	testutil.Equal(t, got.ID, "100")
}

func TestFindCreatedLink_MatchByID(t *testing.T) {
	t.Parallel()
	links := []api.IssueLink{
		{ID: "100", Type: api.IssueLinkType{ID: "1", Name: "DifferentName"}, InwardIssue: &api.LinkedIssue{Key: "PROJ-456"}},
	}
	resolved := api.IssueLinkType{ID: "1", Name: "Blocks"}
	got := findCreatedLink(links, resolved, "PROJ-456")
	testutil.NotNil(t, got)
	testutil.Equal(t, got.ID, "100")
}

func TestFindCreatedLink_DirectionAware(t *testing.T) {
	t.Parallel()
	links := []api.IssueLink{
		{ID: "100", Type: api.IssueLinkType{ID: "1", Name: "Blocks"}, OutwardIssue: &api.LinkedIssue{Key: "PROJ-456"}},
	}
	resolved := api.IssueLinkType{ID: "1", Name: "Blocks"}
	got := findCreatedLink(links, resolved, "PROJ-456")
	testutil.Nil(t, got)
}

func TestFindCreatedLink_NoMatch(t *testing.T) {
	t.Parallel()
	links := []api.IssueLink{
		{ID: "100", Type: api.IssueLinkType{ID: "1", Name: "Blocks"}, InwardIssue: &api.LinkedIssue{Key: "PROJ-999"}},
	}
	resolved := api.IssueLinkType{ID: "1", Name: "Blocks"}
	got := findCreatedLink(links, resolved, "PROJ-456")
	testutil.Nil(t, got)
}

func TestFindCreatedLink_CaseInsensitive(t *testing.T) {
	t.Parallel()
	links := []api.IssueLink{
		{ID: "100", Type: api.IssueLinkType{ID: "1", Name: "blocks"}, InwardIssue: &api.LinkedIssue{Key: "proj-456"}},
	}
	resolved := api.IssueLinkType{Name: "Blocks"}
	got := findCreatedLink(links, resolved, "PROJ-456")
	testutil.NotNil(t, got)
}
