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

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"

	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

// adfDoc builds a minimal ADF document around one paragraph's content JSON.
func adfDoc(paragraphContent string) string {
	return `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[` + paragraphContent + `]}]}`
}

func TestCompareStoredBodyADF(t *testing.T) {
	// The case this exists for: Confluence stores the text but drops the
	// __confluenceMetadata it will not accept back on a link mark.
	sentLink := adfDoc(`{"type":"text","text":"see A11","marks":[{"type":"link","attrs":{"href":"https://example.test#a11","__confluenceMetadata":{"linkType":"page"}}}]}`)
	storedLink := adfDoc(`{"type":"text","text":"see A11","marks":[{"type":"link","attrs":{"href":"https://example.test#a11"}}]}`)

	tests := []struct {
		name            string
		sent, stored    string
		wantTextChanged bool
		wantDropped     int
		wantClean       bool
	}{
		{
			name:      "identical documents",
			sent:      adfDoc(`{"type":"text","text":"hello"}`),
			stored:    adfDoc(`{"type":"text","text":"hello"}`),
			wantClean: true,
		},
		{
			name:            "text silently changed",
			sent:            adfDoc(`{"type":"text","text":"hello"}`),
			stored:          adfDoc(`{"type":"text","text":"hell"}`),
			wantTextChanged: true,
		},
		{
			name:        "server dropped a link attribute but kept the text",
			sent:        sentLink,
			stored:      storedLink,
			wantDropped: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := compareStoredBody(tc.sent, tc.stored, bodyFormatADF)
			if err != nil {
				t.Fatalf("compareStoredBody: %v", err)
			}
			if d.TextChanged != tc.wantTextChanged {
				t.Errorf("TextChanged = %v, want %v", d.TextChanged, tc.wantTextChanged)
			}
			if len(d.DroppedAttrs) != tc.wantDropped {
				t.Errorf("DroppedAttrs = %v, want %d", d.DroppedAttrs, tc.wantDropped)
			}
			if d.Clean() != tc.wantClean {
				t.Errorf("Clean() = %v, want %v", d.Clean(), tc.wantClean)
			}
		})
	}
}

// Dropping an attribute must not be reported as losing content: the two
// carry different consequences, and conflating them would either cry wolf on
// every normalized write or hide a real one.
func TestDroppedAttrIsNotTextLoss(t *testing.T) {
	sent := adfDoc(`{"type":"text","text":"same text","marks":[{"type":"link","attrs":{"href":"h","__confluenceMetadata":{"linkType":"page"}}}]}`)
	stored := adfDoc(`{"type":"text","text":"same text","marks":[{"type":"link","attrs":{"href":"h"}}]}`)
	d, err := compareStoredBody(sent, stored, bodyFormatADF)
	if err != nil {
		t.Fatal(err)
	}
	if d.TextChanged {
		t.Error("attribute normalization reported as text loss")
	}
	lines := driftReport(t, d)
	if !strings.Contains(lines, "Content is intact") {
		t.Errorf("report should say the text survived, got:\n%s", lines)
	}
	if !strings.Contains(lines, "__confluenceMetadata") {
		t.Errorf("report should name the dropped attribute, got:\n%s", lines)
	}
}

// Text order matters: two documents with the same words in different order
// are not the same document.
func TestADFTextRespectsOrder(t *testing.T) {
	a := adfDoc(`{"type":"text","text":"one "},{"type":"text","text":"two"}`)
	b := adfDoc(`{"type":"text","text":"two"},{"type":"text","text":"one "}`)
	d, err := compareStoredBody(a, b, bodyFormatADF)
	if err != nil {
		t.Fatal(err)
	}
	if !d.TextChanged {
		t.Error("reordered text reported as unchanged")
	}
}

func TestCompareStoredBodyXHTMLComparesText(t *testing.T) {
	tests := []struct {
		name         string
		sent, stored string
		wantChanged  bool
	}{
		{
			name:   "markup rewritten, text preserved",
			sent:   "<p>hello <strong>world</strong></p>",
			stored: `<p class="auto">hello <b>world</b></p>`,
		},
		{
			name:        "text actually lost",
			sent:        "<p>hello world</p>",
			stored:      "<p>hello</p>",
			wantChanged: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := compareStoredBody(tc.sent, tc.stored, bodyFormatXHTML)
			if err != nil {
				t.Fatal(err)
			}
			if d.TextChanged != tc.wantChanged {
				t.Errorf("TextChanged = %v, want %v (sent %q stored %q)", d.TextChanged, tc.wantChanged, xhtmlText(tc.sent), xhtmlText(tc.stored))
			}
		})
	}
}

// Confluence returns a macro's parameters in an order of its own choosing. The
// guard exists to stop a write that lost content, so it must not fail a write
// that lost nothing: a reordering has to read as clean, while an edited or
// missing parameter still has to surface.
func TestCompareStoredBodyXHTMLMacroParameterOrder(t *testing.T) {
	macro := func(params ...string) string {
		return `<ac:structured-macro ac:name="code">` + strings.Join(params, "") +
			`<ac:plain-text-body><![CDATA[echo hi]]></ac:plain-text-body></ac:structured-macro>`
	}
	mode := `<ac:parameter ac:name="breakoutMode">wide</ac:parameter>`
	width := `<ac:parameter ac:name="breakoutWidth">760</ac:parameter>`
	theme := `<ac:parameter ac:name="theme">none</ac:parameter>`

	tests := []struct {
		name         string
		sent, stored string
		wantChanges  int
	}{
		{
			name:   "parameters reordered by the server",
			sent:   macro(theme, mode, width),
			stored: macro(mode, width, theme),
		},
		{
			name:        "a parameter value edited",
			sent:        macro(mode, width),
			stored:      macro(mode, `<ac:parameter ac:name="breakoutWidth">500</ac:parameter>`),
			wantChanges: 2, // the old value dropped, the new one added
		},
		{
			name:        "a parameter dropped",
			sent:        macro(mode, width),
			stored:      macro(mode),
			wantChanges: 1,
		},
		{
			// Only the whitespace inside a value moved. It travelled in a
			// whitespace-collapsed text stream before, so collapsing it here
			// keeps the comparison exactly as strict as it was.
			name:   "a parameter value respaced",
			sent:   macro(`<ac:parameter ac:name="t">a  b</ac:parameter>`),
			stored: macro(`<ac:parameter ac:name="t">a b</ac:parameter>`),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := compareStoredBody(tc.sent, tc.stored, bodyFormatXHTML)
			if err != nil {
				t.Fatal(err)
			}
			// The page text is identical in every case here, so claiming the
			// text changed would make the report assert something untrue.
			if d.TextChanged {
				t.Errorf("TextChanged = true; the page text is identical, only parameters differ")
			}
			if got := len(d.ParamsDropped) + len(d.ParamsAdded); got != tc.wantChanges {
				t.Errorf("parameter changes = %v/%v, want %d entries", d.ParamsDropped, d.ParamsAdded, tc.wantChanges)
			}
			if d.Clean() != (tc.wantChanges == 0) {
				t.Errorf("Clean() = %v, want %v", d.Clean(), tc.wantChanges == 0)
			}
		})
	}
}

// A self-closing parameter is valid storage XHTML and the sent body is the
// caller's own markup. Matching it as if it opened a pair runs the value
// capture on to the next closing tag and swallows the page content in between,
// which would hide exactly the loss this guard exists to catch.
func TestMacroParameterSelfClosingDoesNotSwallowContent(t *testing.T) {
	body := `<ac:structured-macro ac:name="m"><ac:parameter ac:name="e"/></ac:structured-macro>` +
		`<p>IMPORTANT PARAGRAPH</p>` +
		`<ac:structured-macro ac:name="code"><ac:parameter ac:name="f">v</ac:parameter></ac:structured-macro>`
	if got := xhtmlText(body); !strings.Contains(got, "IMPORTANT PARAGRAPH") {
		t.Errorf("page text was swallowed by a self-closing parameter: xhtmlText = %q", got)
	}
	// Losing that paragraph must still read as a failed write.
	d, err := compareStoredBody(body, strings.Replace(body, "<p>IMPORTANT PARAGRAPH</p>", "", 1), bodyFormatXHTML)
	if err != nil {
		t.Fatal(err)
	}
	if !d.TextChanged {
		t.Error("dropping a paragraph next to a self-closing parameter reported as unchanged")
	}
}

// Storage format carries macro bodies verbatim inside CDATA, so markup quoted
// in a code block is content and not page configuration. storageProfile
// already strips those spans; the parameter scan has to agree with it.
func TestMacroParameterProfileIgnoresQuotedMarkup(t *testing.T) {
	body := `<ac:structured-macro ac:name="code"><ac:plain-text-body>` +
		`<![CDATA[<ac:parameter ac:name="x">1</ac:parameter>]]></ac:plain-text-body></ac:structured-macro>`
	if got := macroParameterProfile(body); len(got) != 0 {
		t.Errorf("markup quoted inside CDATA read as configuration: %v", got)
	}
}

// A parameter is scoped to the macro holding it, so a value moving between two
// macros is a change rather than a reordering. Without that scope the two
// bodies share one document-wide multiset and the swap passes silently.
func TestMacroParameterProfileIsScopedPerMacro(t *testing.T) {
	macro := func(theme string) string {
		return `<ac:structured-macro ac:name="code"><ac:parameter ac:name="theme">` + theme +
			`</ac:parameter></ac:structured-macro>`
	}
	sent := macro("none") + macro("dark")
	stored := macro("dark") + macro("none")
	if dropped, added := diffMacroParameters(sent, stored); len(dropped)+len(added) == 0 {
		t.Error("a parameter swapped between two macros reported as an in-macro reordering")
	}
}

// A parameter-only mismatch has to fail the command, not just set a field on
// a struct. Without this the fatal branch and its wording could be deleted and
// every other test would still pass.
func TestRunEditFailsWhenStoredMacroParametersDiffer(t *testing.T) {
	const sent = `<ac:structured-macro ac:name="code">` +
		`<ac:parameter ac:name="theme">none</ac:parameter>` +
		`<ac:plain-text-body><![CDATA[echo hi]]></ac:plain-text-body></ac:structured-macro>`
	srv, _ := driftServer(t, func(map[string]any) string {
		// Same page text, one parameter value changed by the server.
		return `{"storage":{"value":"` + strings.ReplaceAll(
			strings.Replace(sent, ">none<", ">dark<", 1), `"`, `\"`) + `"}}`
	})
	defer srv.Close()

	err := runEdit(context.Background(), editOptsFor(t, srv, sent, false))
	if err == nil {
		t.Fatal("expected an error when a stored macro parameter differs from what was sent")
	}
	if !strings.Contains(err.Error(), "macro parameters") {
		t.Errorf("error should name the macro parameters, got: %v", err)
	}
}

// The parameter scan and the text scan have to agree on what a parameter is.
// The profile skips CDATA because markup quoted in a macro body is content, so
// the text scan must keep it: otherwise a server edit to a documented sample is
// dropped from the text and absent from the profile, and nothing catches it.
func TestQuotedParameterMarkupStaysInComparedText(t *testing.T) {
	sample := func(inner string) string {
		return `<ac:structured-macro ac:name="code"><ac:plain-text-body><![CDATA[if (a > b) {}` + "\n" +
			inner + "\n" + `tail]]></ac:plain-text-body></ac:structured-macro>`
	}
	sent := sample(`<ac:parameter ac:name="x">DOCUMENTED SAMPLE</ac:parameter>`)
	stored := sample(`<ac:parameter ac:name="x">SERVER MANGLED IT</ac:parameter>`)

	d, err := compareStoredBody(sent, stored, bodyFormatXHTML)
	if err != nil {
		t.Fatal(err)
	}
	if d.Clean() {
		t.Errorf("a server edit inside a quoted code sample went unreported\n sent  %q\n stored %q",
			xhtmlText(sent), xhtmlText(stored))
	}
}

// Markdown is converted before it is sent, so there is nothing to compare it
// against; saying so is better than reporting a false mismatch.
func TestCompareStoredBodyRejectsMarkdown(t *testing.T) {
	_, err := compareStoredBody("# hi", "{}", bodyFormatMarkdown)
	if err == nil {
		t.Fatal("expected an error for markdown input")
	}
	if !strings.Contains(err.Error(), "converted before sending") {
		t.Errorf("error should explain why markdown is not comparable, got: %v", err)
	}
}

// An unparseable body means the write could not be verified, which must not
// be reported as a verified write.
func TestCompareStoredBodyUnparseable(t *testing.T) {
	if _, err := compareStoredBody(`{"type":"doc"}`, "not json", bodyFormatADF); err == nil {
		t.Fatal("expected an error when the stored body cannot be parsed")
	}
}

func TestDescribeDriftPointsAtTheDifference(t *testing.T) {
	d, err := compareStoredBody(
		adfDoc(`{"type":"text","text":"alpha bravo charlie"}`),
		adfDoc(`{"type":"text","text":"alpha bravo"}`),
		bodyFormatADF)
	if err != nil {
		t.Fatal(err)
	}
	lines := driftReport(t, d)
	if !strings.Contains(lines, "does not hold the content supplied") {
		t.Errorf("report should state the content did not land, got:\n%s", lines)
	}
	if !strings.Contains(lines, "offset") {
		t.Errorf("report should locate the difference, got:\n%s", lines)
	}
}

// driftReport renders a finding through the presenter that owns the wording,
// so the tests assert on what an operator actually reads.
func driftReport(t *testing.T, d writeDrift) string {
	t.Helper()
	model := cflpresent.PagePresenter{}.PresentWriteDrift(cflpresent.WriteDrift{
		BodyFormat:    bodyFormatADF,
		TextChanged:   d.TextChanged,
		VisibleSent:   d.VisibleSent,
		VisibleStored: d.VisibleStored,
		AtomsChanged:  d.AtomsChanged,
		DiffOffset:    diffOffset(d.SentVisible, d.StoredVisible),
		SentExcerpt:   readableExcerpt(d.SentVisible, diffOffset(d.SentVisible, d.StoredVisible)),
		StoredExcerpt: readableExcerpt(d.StoredVisible, diffOffset(d.SentVisible, d.StoredVisible)),
		AtomChanges:   d.AtomChanges,
		DroppedAttrs:  d.DroppedAttrs,
		AddedAttrs:    d.AddedAttrs,
	})
	var b strings.Builder
	for _, sec := range model.Sections {
		if msg, ok := sec.(*sharedpresent.MessageSection); ok {
			b.WriteString(msg.Message)
		}
	}
	return b.String()
}

// A dropped content atom is content loss, not normalization: the text is
// unchanged when an entire card, image or mention disappears.
func TestContentAtomLossCountsAsContentChange(t *testing.T) {
	sent := adfDoc(`{"type":"text","text":"see "},{"type":"inlineCard","attrs":{"url":"https://example.test/x"}}`)
	stored := adfDoc(`{"type":"text","text":"see "}`)
	d, err := compareStoredBody(sent, stored, bodyFormatADF)
	if err != nil {
		t.Fatal(err)
	}
	if !d.TextChanged {
		t.Error("a dropped inlineCard was not reported as content loss")
	}
}

// Swapping one atom for another keeps every count identical, so identity has
// to be part of the fingerprint.
func TestContentAtomSubstitutionDetected(t *testing.T) {
	sent := adfDoc(`{"type":"inlineCard","attrs":{"url":"https://example.test/a"}}`)
	stored := adfDoc(`{"type":"inlineCard","attrs":{"url":"https://example.test/b"}}`)
	d, err := compareStoredBody(sent, stored, bodyFormatADF)
	if err != nil {
		t.Fatal(err)
	}
	if !d.TextChanged {
		t.Error("a substituted inlineCard was not reported as content loss")
	}
}

// driftServer serves a page whose GET body is whatever storedBody returns,
// so a test can make Confluence "store" something other than what was sent.
func driftServer(t *testing.T, storedBody func(sent map[string]any) string) (*httptest.Server, *int) {
	t.Helper()
	gets := 0
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			gets++
			w.WriteHeader(http.StatusOK)
			body := `{"storage":{"value":"<p>Old</p>"}}`
			if received != nil {
				body = storedBody(received)
			}
			_, _ = w.Write([]byte(`{"id":"12345","title":"Test","version":{"number":1},"body":` + body + `,"_links":{"webui":"/pages/12345"}}`))
		case "PUT":
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &received)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"12345","title":"Test","version":{"number":2},"_links":{"webui":"/pages/12345"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return srv, &gets
}

func editOptsFor(t *testing.T, srv *httptest.Server, content string, noVerify bool) *editOptions {
	t.Helper()
	rootOpts := newEditTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(srv.URL, "test@example.com", "token"))
	rootOpts.Stdin = strings.NewReader(content)
	return &editOptions{
		Options:    rootOpts,
		pageID:     "12345",
		file:       "-",
		bodyFormat: bodyFormatXHTML,
		noVerify:   noVerify,
	}
}

// The point of the feature: a write the server quietly altered must fail
// rather than report the version number and exit zero.
func TestRunEditFailsWhenStoredContentDiffers(t *testing.T) {
	srv, _ := driftServer(t, func(map[string]any) string {
		return `{"storage":{"value":"<p>something else entirely</p>"}}`
	})
	defer srv.Close()

	err := runEdit(context.Background(), editOptsFor(t, srv, "<p>what we sent</p>", false))
	if err == nil {
		t.Fatal("expected an error when the stored body does not match what was sent")
	}
	if !strings.Contains(err.Error(), "does not match what was sent") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --no-verify keeps the old behavior, including skipping the extra read.
func TestRunEditNoVerifySkipsReadback(t *testing.T) {
	srv, gets := driftServer(t, func(map[string]any) string {
		return `{"storage":{"value":"<p>something else entirely</p>"}}`
	})
	defer srv.Close()

	before := *gets
	if err := runEdit(context.Background(), editOptsFor(t, srv, "<p>what we sent</p>", true)); err != nil {
		t.Fatalf("--no-verify should not fail on drift: %v", err)
	}
	if after := *gets - before; after != 1 {
		t.Errorf("GET count = %d, want 1 (the pre-write fetch only)", after)
	}
}

// Normalization is not failure: the write stands and the drift is reported.
func TestRunEditToleratesNormalization(t *testing.T) {
	srv, _ := driftServer(t, func(map[string]any) string {
		// Same text, markup rewritten — what storage-format normalization
		// looks like.
		return `{"storage":{"value":"<p class=\"auto\">what we sent</p>"}}`
	})
	defer srv.Close()

	if err := runEdit(context.Background(), editOptsFor(t, srv, "<p>what we sent</p>", false)); err != nil {
		t.Fatalf("normalized markup should not fail the write: %v", err)
	}
}

// Markdown is converted before sending, so verification must not run and
// must not invent a mismatch.
func TestRunEditSkipsVerificationForMarkdown(t *testing.T) {
	srv, gets := driftServer(t, func(map[string]any) string {
		return `{"storage":{"value":"<p>totally different</p>"}}`
	})
	defer srv.Close()

	opts := editOptsFor(t, srv, "# heading", false)
	opts.bodyFormat = bodyFormatMarkdown
	before := *gets
	if err := runEdit(context.Background(), opts); err != nil {
		t.Fatalf("markdown edit should not be verified: %v", err)
	}
	if after := *gets - before; after != 1 {
		t.Errorf("GET count = %d, want 1 (no verification read)", after)
	}
}

// The fingerprint's NUL atom separators are an internal detail; an operator
// must never be shown them.
func TestExcerptHidesFingerprintSeparators(t *testing.T) {
	sent := adfDoc(`{"type":"text","text":"see "},{"type":"inlineCard","attrs":{"url":"https://example.test/x"}}`)
	stored := adfDoc(`{"type":"text","text":"see "}`)
	d, err := compareStoredBody(sent, stored, bodyFormatADF)
	if err != nil {
		t.Fatal(err)
	}
	report := driftReport(t, d)
	if strings.Contains(report, "\x00") || strings.ContainsRune(report, 0) {
		t.Errorf("report leaked the fingerprint separator:\n%s", report)
	}
	if !strings.Contains(report, "inlineCard") {
		t.Errorf("report should still name the lost atom via its attributes:\n%s", report)
	}
}

// Attribute-less atoms move neither the text nor the attribute profile, so
// they have to be named in the fingerprint or their loss is invisible.
func TestAttributelessAtomLossDetected(t *testing.T) {
	for _, atom := range []string{"hardBreak", "rule"} {
		t.Run(atom, func(t *testing.T) {
			sent := adfDoc(`{"type":"text","text":"a"},{"type":"` + atom + `"},{"type":"text","text":"b"}`)
			stored := adfDoc(`{"type":"text","text":"a"},{"type":"text","text":"b"}`)
			d, err := compareStoredBody(sent, stored, bodyFormatADF)
			if err != nil {
				t.Fatal(err)
			}
			if !d.TextChanged {
				t.Errorf("a dropped %s was not reported as content loss", atom)
			}
		})
	}
}

// createDriftServer answers a create and then serves whatever storedBody
// returns on the readback, so a test can make Confluence "store" something
// other than what was posted.
func createDriftServer(t *testing.T, storedBody func() string) (*httptest.Server, *int) {
	t.Helper()
	gets := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"results":[{"id":"123456","key":"DEV"}]}`))
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/"):
			gets++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"99999","title":"Test","version":{"number":1},"body":` + storedBody() + `}`))
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/pages"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"99999","title":"Test","version":{"number":1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return srv, &gets
}

func createOptsFor(t *testing.T, srv *httptest.Server, content string, noVerify bool) *createOptions {
	t.Helper()
	rootOpts := newCreateTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(srv.URL, "test@example.com", "token"))
	rootOpts.Stdin = strings.NewReader(content)
	return &createOptions{
		Options:    rootOpts,
		space:      "DEV",
		title:      "Test Page",
		file:       "-",
		bodyFormat: bodyFormatXHTML,
		noVerify:   noVerify,
	}
}

// A create writes the same verbatim body an edit does, so it must fail the
// same way when the server stores something else.
func TestRunCreateFailsWhenStoredContentDiffers(t *testing.T) {
	srv, _ := createDriftServer(t, func() string {
		return `{"storage":{"value":"<p>something else entirely</p>"}}`
	})
	defer srv.Close()

	err := runCreate(context.Background(), createOptsFor(t, srv, "<p>what we sent</p>", false))
	if err == nil {
		t.Fatal("expected an error when the created page does not hold what was sent")
	}
	if !strings.Contains(err.Error(), "does not match what was sent") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunCreateNoVerifySkipsReadback(t *testing.T) {
	srv, gets := createDriftServer(t, func() string {
		return `{"storage":{"value":"<p>something else entirely</p>"}}`
	})
	defer srv.Close()

	if err := runCreate(context.Background(), createOptsFor(t, srv, "<p>what we sent</p>", true)); err != nil {
		t.Fatalf("--no-verify should not fail on drift: %v", err)
	}
	if *gets != 0 {
		t.Errorf("readback GET count = %d, want 0", *gets)
	}
}

func TestRunCreateToleratesNormalization(t *testing.T) {
	srv, _ := createDriftServer(t, func() string {
		return `{"storage":{"value":"<p class=\"auto\">what we sent</p>"}}`
	})
	defer srv.Close()

	if err := runCreate(context.Background(), createOptsFor(t, srv, "<p>what we sent</p>", false)); err != nil {
		t.Fatalf("normalized markup should not fail the create: %v", err)
	}
}

// The numbers reported must describe the document, not the internal
// fingerprint: an atom marker must not inflate a character count, and a
// multibyte change must not report equal lengths as if nothing moved.
func TestReportedCountsDescribeTheDocument(t *testing.T) {
	t.Run("atom does not inflate the character count", func(t *testing.T) {
		sent := adfDoc(`{"type":"text","text":"hello ."},{"type":"inlineCard","attrs":{"url":"https://example.test/a-very-long-url"}}`)
		stored := adfDoc(`{"type":"text","text":"hello ."}`)
		d, err := compareStoredBody(sent, stored, bodyFormatADF)
		if err != nil {
			t.Fatal(err)
		}
		if d.VisibleSent != 7 {
			t.Errorf("VisibleSent = %d, want 7 (the characters a reader sees)", d.VisibleSent)
		}
		if !d.AtomsChanged {
			t.Error("AtomsChanged = false, but only the embedded card differs")
		}
		if r := driftReport(t, d); !strings.Contains(r, "embedded content differs") {
			t.Errorf("report should say the embedded content changed:\n%s", r)
		}
	})

	t.Run("multibyte text counts runes", func(t *testing.T) {
		sent := adfDoc(`{"type":"text","text":"日本語テキスト"}`)
		stored := adfDoc(`{"type":"text","text":"日本語"}`)
		d, err := compareStoredBody(sent, stored, bodyFormatADF)
		if err != nil {
			t.Fatal(err)
		}
		if d.VisibleSent != 7 || d.VisibleStored != 3 {
			t.Errorf("visible counts = %d/%d, want 7/3 runes", d.VisibleSent, d.VisibleStored)
		}
		if off := diffOffset(d.SentVisible, d.StoredVisible); off != 3 {
			t.Errorf("diffOffset = %d, want 3 (runes, not bytes)", off)
		}
	})
}

// The XHTML branch must measure the document too, or the report states a
// length of zero characters for a body it never counted.
func TestXHTMLDriftReportsRealCounts(t *testing.T) {
	d, err := compareStoredBody("<p>hello world</p>", "<p>hello</p>", bodyFormatXHTML)
	if err != nil {
		t.Fatal(err)
	}
	if d.VisibleSent != len("hello world") || d.VisibleStored != len("hello") {
		t.Errorf("visible counts = %d/%d, want %d/%d", d.VisibleSent, d.VisibleStored, len("hello world"), len("hello"))
	}
	report := driftReport(t, d)
	if strings.Contains(report, "0 characters") {
		t.Errorf("report claimed zero characters for a measured body:\n%s", report)
	}
}

// An atoms-only change has no position in the visible text, so none is
// claimed.
func TestAtomOnlyChangeReportsNoTextOffset(t *testing.T) {
	sent := adfDoc(`{"type":"text","text":"hello"},{"type":"inlineCard","attrs":{"url":"https://example.test/a"}}`)
	stored := adfDoc(`{"type":"text","text":"hello"},{"type":"inlineCard","attrs":{"url":"https://example.test/b"}}`)
	d, err := compareStoredBody(sent, stored, bodyFormatADF)
	if err != nil {
		t.Fatal(err)
	}
	if off := diffOffset(d.SentVisible, d.StoredVisible); off != -1 {
		t.Errorf("diffOffset = %d, want -1: the visible text is identical", off)
	}
	if r := driftReport(t, d); strings.Contains(r, "first difference at offset") {
		t.Errorf("report claimed a text position for an atoms-only change:\n%s", r)
	}
}

// hardBreak and rule carry no attributes, so a change confined to them has
// no attribute lines to identify it. The atom names have to carry that.
func TestAttributelessAtomChangeIsNamed(t *testing.T) {
	sent := adfDoc(`{"type":"text","text":"a"},{"type":"hardBreak"},{"type":"text","text":"b"}`)
	stored := adfDoc(`{"type":"text","text":"a"},{"type":"text","text":"b"}`)
	d, err := compareStoredBody(sent, stored, bodyFormatADF)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.DroppedAttrs) != 0 {
		t.Fatalf("precondition: hardBreak should contribute no attributes, got %v", d.DroppedAttrs)
	}
	report := driftReport(t, d)
	if !strings.Contains(report, "hardBreak") {
		t.Errorf("report must name the changed atom when no attributes can:\n%s", report)
	}
}

// The loss this exists to catch: a no-op ADF round trip that Confluence
// stores without the emphasis marks its storage body carried. Sent and
// stored agree, so only the storage comparison sees it.
func TestStorageLossDetectedWhenSentAndStoredAgree(t *testing.T) {
	before := `<p><em><strong><code>KEY</code></strong></em></p><p><strong>bold</strong></p>`
	after := `<p><code>KEY</code></p><p><strong>bold</strong></p>`
	lost := diffStorageLoss(storageProfile(before), storageProfile(after))
	if len(lost) != 2 {
		t.Fatalf("lost = %v, want em and strong", lost)
	}
	joined := strings.Join(lost, " ")
	if !strings.Contains(joined, "em (1→0)") || !strings.Contains(joined, "strong (2→1)") {
		t.Errorf("unexpected loss report: %v", lost)
	}
}

// Editing text moves no element counts, so an ordinary edit stays quiet.
func TestStorageLossQuietOnTextOnlyEdit(t *testing.T) {
	before := `<p>hello <strong>world</strong></p>`
	after := `<p>goodbye <strong>world</strong></p>`
	if lost := diffStorageLoss(storageProfile(before), storageProfile(after)); len(lost) != 0 {
		t.Errorf("text-only edit reported loss: %v", lost)
	}
}

// Adding content is what an edit is for; only losses are reported.
func TestStorageLossIgnoresAdditions(t *testing.T) {
	before := `<p>hello</p>`
	after := `<p>hello</p><p><strong>new</strong></p>`
	if lost := diffStorageLoss(storageProfile(before), storageProfile(after)); len(lost) != 0 {
		t.Errorf("additions reported as loss: %v", lost)
	}
}

// A missing baseline must be reported as "cannot compare", never pass as
// silence: silence is what a clean write looks like.
func TestStorageComparisonUnavailableWithoutBaseline(t *testing.T) {
	req := verifyRequest{storageBefore: "", storageBeforeErr: "resource not found", comparePriorState: true}
	got := compareStorage(context.Background(), req, "")
	if !got.Unavailable {
		t.Error("a missing baseline was not reported as uncomparable")
	}
	if got.Reason != "resource not found" {
		t.Errorf("Reason = %q, want the captured cause", got.Reason)
	}
	if len(got.Lost) != 0 {
		t.Errorf("claimed loss without a baseline: %v", got.Lost)
	}
}

// An xhtml write already read the stored body back; reuse it rather than
// fetching the same thing twice.
func TestStorageComparisonReusesSuppliedBody(t *testing.T) {
	req := verifyRequest{storageBefore: `<p><strong>a</strong></p>`, comparePriorState: true}
	got := compareStorage(context.Background(), req, `<p>a</p>`)
	if got.Unavailable {
		t.Fatal("reported uncomparable despite a supplied body")
	}
	if len(got.Lost) != 1 || !strings.Contains(got.Lost[0], "strong") {
		t.Errorf("Lost = %v, want the dropped strong", got.Lost)
	}
}

// Storage keeps macro bodies verbatim inside CDATA; angle brackets there are
// content, and counting them would invent losses that never happened.
func TestStorageProfileIgnoresCDATAAndComments(t *testing.T) {
	body := `<p>x</p><ac:plain-text-body><![CDATA[<strong>not markup</strong>]]></ac:plain-text-body><!-- <em>also not</em> -->`
	p := storageProfile(body)
	if p["strong"] != 0 || p["em"] != 0 {
		t.Errorf("counted markup inside CDATA/comments: %v", p)
	}
	if p["p"] != 1 {
		t.Errorf("real elements not counted: %v", p)
	}
}

// The stated cause must match the format actually used.
func TestStorageLossCauseMatchesFormat(t *testing.T) {
	if !strings.Contains(storageLossCause(bodyFormatADF), "ADF") {
		t.Error("adf cause should name the ADF round trip")
	}
	if strings.Contains(storageLossCause(bodyFormatXHTML), "ADF") {
		t.Error("an xhtml write must not be blamed on the ADF round trip")
	}
}

func TestVerificationApplies(t *testing.T) {
	tests := []struct {
		format               string
		hasNewContent, noVer bool
		want                 bool
	}{
		{bodyFormatADF, true, false, true},
		{bodyFormatXHTML, true, false, true},
		{bodyFormatMarkdown, true, false, false},
		{bodyFormatADF, true, true, false},
		{bodyFormatADF, false, false, false},
	}
	for _, tc := range tests {
		if got := verificationApplies(tc.format, tc.hasNewContent, tc.noVer); got != tc.want {
			t.Errorf("verificationApplies(%q,%v,%v) = %v, want %v", tc.format, tc.hasNewContent, tc.noVer, got, tc.want)
		}
	}
}

// The dropped-before-added ordering is a claim about rendered output, so it is
// asserted against rendered output. An edit has to read old value then new.
func TestPresentedMacroParametersReadOldThenNew(t *testing.T) {
	render := func(d cflpresent.WriteDrift) string {
		var b strings.Builder
		for _, sec := range (cflpresent.PagePresenter{}).PresentWriteDrift(d).Sections {
			if msg, ok := sec.(*sharedpresent.MessageSection); ok {
				b.WriteString(msg.Message)
			}
		}
		return b.String()
	}
	drift := cflpresent.WriteDrift{
		BodyFormat:    bodyFormatXHTML,
		ParamsDropped: []string{"macro 1 parameter breakoutWidth=760 (1→0)"},
		ParamsAdded:   []string{"macro 1 parameter breakoutWidth=500 (0→1)"},
	}

	out := render(drift)
	if !strings.Contains(out, "macro parameters that were sent") {
		t.Errorf("report should say the parameters did not survive:\n%s", out)
	}
	if !strings.Contains(out, "Page text is intact") {
		t.Errorf("report should say the text survived, not that it changed:\n%s", out)
	}
	dropped, added := strings.Index(out, "- macro 1 parameter breakoutWidth=760"), strings.Index(out, "+ macro 1 parameter breakoutWidth=500")
	if dropped < 0 || added < 0 {
		t.Fatalf("report missing one of the parameter lines:\n%s", out)
	}
	if dropped > added {
		t.Errorf("an edit reads backwards: the new value is printed above the one it replaced:\n%s", out)
	}

	// The same ordering has to hold in the text-loss branch, which renders
	// through a different code path.
	drift.TextChanged = true
	drift.DiffOffset = -1
	out = render(drift)
	dropped, added = strings.Index(out, "macro parameters dropped:"), strings.Index(out, "macro parameters added:")
	if dropped < 0 || added < 0 {
		t.Fatalf("text-loss branch missing a parameter header:\n%s", out)
	}
	if dropped > added {
		t.Errorf("text-loss branch prints added before dropped:\n%s", out)
	}
}

func TestPresentedStorageLossExplainsTheCause(t *testing.T) {
	model := cflpresent.PagePresenter{}.PresentWriteDrift(cflpresent.WriteDrift{
		BodyFormat:   bodyFormatADF,
		LostElements: []string{"em (3→1)", "strong (3→1)"},
		LossCause:    storageLossCause(bodyFormatADF),
	})
	var b strings.Builder
	for _, sec := range model.Sections {
		if msg, ok := sec.(*sharedpresent.MessageSection); ok {
			b.WriteString(msg.Message)
		}
	}
	out := b.String()
	for _, want := range []string{"lost formatting", "em (3→1)", "writing it back"} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q:\n%s", want, out)
		}
	}
}

// A comparison that could not run must say so; silence is what a clean write
// looks like, and the two must never be confused.
func TestPresentedUncomparableStorageIsStated(t *testing.T) {
	model := cflpresent.PagePresenter{}.PresentWriteDrift(cflpresent.WriteDrift{
		BodyFormat:          bodyFormatADF,
		StorageUncomparable: true,
		StorageReason:       "resource not found",
	})
	var b strings.Builder
	for _, sec := range model.Sections {
		if msg, ok := sec.(*sharedpresent.MessageSection); ok {
			b.WriteString(msg.Message)
		}
	}
	out := b.String()
	for _, want := range []string{"Could not compare", "would not have been noticed", "resource not found"} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q:\n%s", want, out)
		}
	}
}

// A create has no prior state. That is not a failed comparison, and saying
// so on every clean create would train the operator to ignore the warning.
func TestStorageComparisonSilentWhenThereIsNoPriorState(t *testing.T) {
	req := verifyRequest{storageBefore: "", comparePriorState: false}
	got := compareStorage(context.Background(), req, "")
	if got.Unavailable {
		t.Error("a create was reported as a failed comparison")
	}
	if len(got.Lost) != 0 {
		t.Errorf("Lost = %v, want none", got.Lost)
	}
}

// Storage carries namespaced elements; a lost macro must not be invisible.
func TestStorageProfileCountsNamespacedElements(t *testing.T) {
	body := `<p>x</p><ac:structured-macro ac:name="info"><ac:rich-text-body><p>y</p></ac:rich-text-body></ac:structured-macro><ri:attachment ri:filename="a.png"/>`
	p := storageProfile(body)
	for _, want := range []string{"ac:structured-macro", "ac:rich-text-body", "ri:attachment"} {
		if p[want] != 1 {
			t.Errorf("%s not counted: %v", want, p)
		}
	}
	if p["p"] != 2 {
		t.Errorf("plain elements miscounted: %v", p)
	}
}

// The loss that motivated namespacing: a macro removed by a write.
func TestStorageLossDetectsRemovedMacro(t *testing.T) {
	before := `<p>x</p><ac:structured-macro ac:name="info"><p>y</p></ac:structured-macro>`
	after := `<p>x</p>`
	lost := strings.Join(diffStorageLoss(storageProfile(before), storageProfile(after)), " ")
	if !strings.Contains(lost, "ac:structured-macro") {
		t.Errorf("a removed macro was not reported: %s", lost)
	}
}

// The baseline comes from the page already fetched, and the xhtml readback
// is reused for the storage comparison. Both were review findings; without a
// count pinned on the verified path, either could be reintroduced silently.
func TestRunEditVerifiedPathMakesNoRedundantReads(t *testing.T) {
	srv, gets := driftServer(t, func(map[string]any) string {
		return `{"storage":{"value":"<p>what we sent</p>"}}`
	})
	defer srv.Close()

	if err := runEdit(context.Background(), editOptsFor(t, srv, "<p>what we sent</p>", false)); err != nil {
		t.Fatalf("verified edit: %v", err)
	}
	// One read before the write, one to read the result back. A third would
	// mean the baseline or the storage comparison re-fetched what the
	// command already had.
	if *gets != 2 {
		t.Errorf("GET count = %d, want 2 (pre-write fetch + post-write readback)", *gets)
	}
}
