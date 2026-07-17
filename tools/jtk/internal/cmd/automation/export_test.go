package automation

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunExportFormatsJSONDeterministically(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		compact bool
		want    string
	}{
		{"pretty input defaults to pretty", "{\n    \"name\": \"rule\",\n    \"enabled\": true\n}", false, "{\n  \"name\": \"rule\",\n  \"enabled\": true\n}\n"},
		{"compact input defaults to pretty", `{"name":"rule","enabled":true}`, false, "{\n  \"name\": \"rule\",\n  \"enabled\": true\n}\n"},
		{"compact normalizes whitespace", " { \"name\" : \"rule\", \"enabled\" : true } ", true, `{"name":"rule","enabled":true}` + "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := automationExportServer(tt.raw)
			defer server.Close()

			opts, stdout := exportTestOptions(t, server)
			testutil.RequireNoError(t, runExport(context.Background(), opts, "rule-1", tt.compact))
			if stdout.String() != tt.want {
				t.Fatalf("output mismatch:\n got %q\nwant %q", stdout.String(), tt.want)
			}
		})
	}
}

func TestRunExportMalformedJSONLeavesStdoutEmpty(t *testing.T) {
	t.Parallel()
	for _, compact := range []bool{false, true} {
		server := automationExportServer(`{"broken":`)
		opts, stdout := exportTestOptions(t, server)
		err := runExport(context.Background(), opts, "rule-1", compact)
		server.Close()

		if err == nil {
			t.Fatalf("compact=%v: expected malformed JSON error", compact)
		}
		if stdout.Len() != 0 {
			t.Fatalf("compact=%v: stdout must stay empty, got %q", compact, stdout.String())
		}
	}
}

func automationExportServer(raw string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}
		_, _ = w.Write([]byte(raw))
	}))
}

func exportTestOptions(t *testing.T, server *httptest.Server) (*root.Options, *bytes.Buffer) {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	stdout := &bytes.Buffer{}
	opts := &root.Options{Stdout: stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)
	return opts, stdout
}
