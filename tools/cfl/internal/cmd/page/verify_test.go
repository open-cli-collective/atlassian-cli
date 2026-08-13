package page

import (
	"strings"
	"testing"
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
	lines := strings.Join(describeDrift(d, bodyFormatADF), "\n")
	if !strings.Contains(lines, "Text content is intact") {
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
	lines := strings.Join(describeDrift(d, bodyFormatADF), "\n")
	if !strings.Contains(lines, "does not contain the content supplied") {
		t.Errorf("report should state the content did not land, got:\n%s", lines)
	}
	if !strings.Contains(lines, "offset") {
		t.Errorf("report should locate the difference, got:\n%s", lines)
	}
}
