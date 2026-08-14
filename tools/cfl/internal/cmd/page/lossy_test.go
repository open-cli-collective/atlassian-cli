package page

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/confluence-cli/api"
)

// storageServer serves a page whose storage body is fixed from the outset,
// including on the read that happens before a write. driftServer only
// applies its body after a PUT, so it cannot exercise a guard that inspects
// the page as it currently stands.
func storageServer(t *testing.T, storage string) *httptest.Server {
	t.Helper()
	payload := `{"id":"12345","title":"Test","version":{"number":1},"body":{"storage":{"value":` +
		mustJSON(t, storage) + `}},"_links":{"webui":"/pages/12345"}}`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(payload))
		case "PUT":
			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(payload))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func mustJSON(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestFindLossyConstructs(t *testing.T) {
	tests := []struct {
		name    string
		storage string
		want    []string
	}{
		{
			name:    "plain content is safe",
			storage: `<p>hello <strong>world</strong> and <code>code</code></p>`,
		},
		{
			name:    "internal link",
			storage: `<p><ac:link ac:anchor="a11"><ac:link-body>see</ac:link-body></ac:link></p>`,
			want:    []string{"internal links"},
		},
		{
			name:    "emphasis wrapping code",
			storage: `<p><em><strong><code>KEY</code></strong></em></p>`,
			want:    []string{"emphasis on code spans"},
		},
		{
			name:    "code wrapping emphasis",
			storage: `<p><code><strong>KEY</strong></code></p>`,
			want:    []string{"emphasis on code spans"},
		},
		{
			// Verified against Confluence: a structured macro round-trips
			// through ADF as a panel node and comes back intact, so
			// refusing on it would block a great many pages over a loss
			// that does not happen.
			name:    "structured macro survives and is not refused",
			storage: `<ac:structured-macro ac:name="info"><ac:rich-text-body><p>x</p></ac:rich-text-body></ac:structured-macro>`,
		},
		{
			name:    "empty body reports nothing",
			storage: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := findLossyConstructs(tc.storage)
			if len(got) != len(tc.want) {
				t.Fatalf("findings = %v, want %v", got, tc.want)
			}
			for i, w := range tc.want {
				if got[i].Construct != w {
					t.Errorf("finding[%d] = %q, want %q", i, got[i].Construct, w)
				}
			}
		})
	}
}

// Emphasis that does not touch a code span is not a loss and must not be
// reported: a guard that fires on safe pages gets disabled.
func TestFindLossyConstructsIgnoresPlainEmphasis(t *testing.T) {
	if got := findLossyConstructs(`<p><strong><em>bold italic</em></strong></p>`); len(got) != 0 {
		t.Errorf("plain emphasis reported as lossy: %v", got)
	}
}

// The refusal has to name what would go and how to proceed; an error that
// only says no gets worked around blindly.
func TestLossyFormatErrorNamesLossAndRemedy(t *testing.T) {
	err := lossyFormatError([]lossyFinding{
		{Construct: "internal links", Detail: "anchor and page references become plain expanded URLs", Count: 8},
	}, bodyFormatADF)
	msg := err.Error()
	for _, want := range []string{"internal links (8)", "expanded URLs", "--body-format xhtml", "--allow-lossy"} {
		if !strings.Contains(msg, want) {
			t.Errorf("refusal missing %q:\n%s", want, msg)
		}
	}
}

// The guard must stop the write, not report it afterwards: once written, the
// content is gone and the caller's copy never had it.
func TestRunEditRefusesLossyAdfWrite(t *testing.T) {
	srv := storageServer(t, `<p><ac:link ac:anchor="a11"><ac:link-body>see</ac:link-body></ac:link></p>`)
	defer srv.Close()

	opts := editOptsFor(t, srv, `{"type":"doc","version":1,"content":[]}`, true)
	opts.bodyFormat = bodyFormatADF
	err := runEdit(context.Background(), opts)
	if err == nil {
		t.Fatal("expected the lossy write to be refused")
	}
	if !strings.Contains(err.Error(), "internal links") {
		t.Errorf("refusal should name the construct: %v", err)
	}
}

func TestRunEditAllowLossyProceeds(t *testing.T) {
	srv := storageServer(t, `<p><ac:link ac:anchor="a11"><ac:link-body>see</ac:link-body></ac:link></p>`)
	defer srv.Close()

	opts := editOptsFor(t, srv, `{"type":"doc","version":1,"content":[]}`, true)
	opts.bodyFormat = bodyFormatADF
	opts.allowLossy = true
	if err := runEdit(context.Background(), opts); err != nil {
		t.Fatalf("--allow-lossy should permit the write: %v", err)
	}
}

// xhtml carries these constructs, so it is never refused.
func TestRunEditDoesNotRefuseXhtml(t *testing.T) {
	srv := storageServer(t, `<p><ac:link ac:anchor="a11"><ac:link-body>see</ac:link-body></ac:link></p>`)
	defer srv.Close()

	opts := editOptsFor(t, srv, `<p><ac:link ac:anchor="a11"><ac:link-body>see</ac:link-body></ac:link></p>`, true)
	if err := runEdit(context.Background(), opts); err != nil {
		t.Fatalf("xhtml write should not be refused: %v", err)
	}
}

// Angle brackets inside CDATA are content, not markup; matching there would
// refuse a write over nothing.
func TestFindLossyConstructsIgnoresCDATA(t *testing.T) {
	body := `<p>ok</p><ac:plain-text-body><![CDATA[<ac:link ac:anchor="x"/>]]></ac:plain-text-body>`
	if got := findLossyConstructs(body); len(got) != 0 {
		t.Errorf("matched inside CDATA: %v", got)
	}
}

// A guard that cannot read the page must refuse, not wave the write through:
// failing open is the behaviour it exists to prevent.
func TestRunEditRefusesWhenBaselineUnreadable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// A page whose body carries no storage representation.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"12345","title":"T","version":{"number":1},"body":{},"_links":{"webui":"/p"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"12345","title":"T","version":{"number":2},"_links":{"webui":"/p"}}`))
	}))
	defer srv.Close()

	opts := editOptsFor(t, srv, `{"type":"doc","version":1,"content":[]}`, true)
	opts.bodyFormat = bodyFormatADF
	err := runEdit(context.Background(), opts)
	if err == nil {
		t.Fatal("expected a refusal when the page's current content cannot be read")
	}
	if !strings.Contains(err.Error(), "could not be read") {
		t.Errorf("refusal should say the check could not run: %v", err)
	}
}

// Emphasis and code are not always adjacent; a shape-matching regex missed
// the case with markup in between, which is the common one in real pages.
func TestCountEmphasisedCodeToleratesInterveningMarkup(t *testing.T) {
	tests := []struct {
		name   string
		markup string
		want   int
	}{
		{"adjacent", `<strong><code>K</code></strong>`, 1},
		{"code outside", `<code><em>K</em></code>`, 1},
		{"markup in between", `<strong>label <a href="x">link</a> <code>K</code></strong>`, 1},
		{"two in one run", `<strong><code>A</code> and <code>B</code></strong>`, 2},
		{"siblings, not nested", `<strong>bold</strong> <code>K</code>`, 0},
		{"emphasis alone", `<strong><em>bold italic</em></strong>`, 0},
		{"code alone", `<p><code>K</code></p>`, 0},
		{"closed before code", `<strong>bold</strong><p><code>K</code></p>`, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := countEmphasisedCode(tc.markup); got != tc.want {
				t.Errorf("countEmphasisedCode(%q) = %d, want %d", tc.markup, got, tc.want)
			}
		})
	}
}

// A title-only edit on a page with a storage body resends that storage
// losslessly, whatever --body-format was asked for. Refusing it would be a
// false positive, and a guard that fires on safe edits gets overridden by
// reflex.
func TestRunEditAllowsTitleOnlyEditOfStoragePage(t *testing.T) {
	srv := storageServer(t, `<p><ac:link ac:anchor="a11"><ac:link-body>see</ac:link-body></ac:link></p>`)
	defer srv.Close()

	rootOpts := newEditTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(srv.URL, "test@example.com", "token"))
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options:    rootOpts,
		pageID:     "12345",
		title:      "New Title",
		bodyFormat: bodyFormatADF,
	}
	if err := runEdit(context.Background(), opts); err != nil {
		t.Fatalf("a title-only edit resends storage and must not be refused: %v", err)
	}
}

// --no-verify silences the readback. Consenting to skip the report is not
// consenting to cause the damage.
func TestNoVerifyDoesNotDisableTheGuard(t *testing.T) {
	srv := storageServer(t, `<p><ac:link ac:anchor="a11"><ac:link-body>see</ac:link-body></ac:link></p>`)
	defer srv.Close()

	opts := editOptsFor(t, srv, `{"type":"doc","version":1,"content":[]}`, true)
	opts.bodyFormat = bodyFormatADF
	opts.noVerify = true
	err := runEdit(context.Background(), opts)
	if err == nil {
		t.Fatal("--no-verify must not permit a lossy write")
	}
	if !strings.Contains(err.Error(), "internal links") {
		t.Errorf("unexpected error: %v", err)
	}
}
