package page

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

// Confluence normalizes what it stores. A write can land with parts of the
// submitted document silently removed — observed with the
// __confluenceMetadata attributes Confluence strips from link marks — and
// the update response reports success either way. For an exact body format
// cfl transmits the caller's content verbatim, so any difference between
// what was sent and what came back is the server's doing and is precisely
// what the caller cannot otherwise see.
//
// Two kinds of difference carry different weight, so they are reported
// separately: losing text means the content did not land, while losing
// attributes means the document was normalized around content that did.

// writeDrift describes how a stored body differs from the body that was sent.
type writeDrift struct {
	// TextChanged reports that the document's text content differs. This is
	// a failed write, not a normalization.
	TextChanged bool
	SentText    string
	StoredText  string

	// DroppedAttrs and AddedAttrs are "nodeType.attrName" keys whose
	// occurrence counts fell or rose, in sorted order.
	DroppedAttrs []string
	AddedAttrs   []string
}

// Clean reports whether the stored body matched what was sent.
func (d writeDrift) Clean() bool {
	return !d.TextChanged && len(d.DroppedAttrs) == 0 && len(d.AddedAttrs) == 0
}

// compareStoredBody diffs a submitted body against the one Confluence stored.
// Bodies that cannot be parsed are reported as an error rather than silently
// treated as matching — an unverifiable write is not a verified one.
func compareStoredBody(sent, stored, bodyFormat string) (writeDrift, error) {
	switch bodyFormat {
	case bodyFormatADF:
		return compareADF(sent, stored)
	case bodyFormatXHTML:
		sentText, storedText := xhtmlText(sent), xhtmlText(stored)
		return writeDrift{
			TextChanged: sentText != storedText,
			SentText:    sentText,
			StoredText:  storedText,
		}, nil
	default:
		return writeDrift{}, fmt.Errorf("cannot verify %s input: it is converted before sending, so the stored body is not comparable to what was supplied", bodyFormat)
	}
}

func compareADF(sent, stored string) (writeDrift, error) {
	var sentDoc, storedDoc any
	if err := json.Unmarshal([]byte(sent), &sentDoc); err != nil {
		return writeDrift{}, fmt.Errorf("parsing submitted ADF: %w", err)
	}
	if err := json.Unmarshal([]byte(stored), &storedDoc); err != nil {
		return writeDrift{}, fmt.Errorf("parsing stored ADF: %w", err)
	}

	sentText, storedText := adfText(sentDoc), adfText(storedDoc)
	dropped, added := diffAttrProfiles(adfAttrProfile(sentDoc), adfAttrProfile(storedDoc))
	return writeDrift{
		TextChanged:  sentText != storedText,
		SentText:     sentText,
		StoredText:   storedText,
		DroppedAttrs: dropped,
		AddedAttrs:   added,
	}, nil
}

// adfText concatenates every text node in document order.
func adfText(node any) string {
	var b strings.Builder
	var walk func(any)
	walk = func(n any) {
		switch v := n.(type) {
		case map[string]any:
			if v["type"] == "text" {
				if s, ok := v["text"].(string); ok {
					b.WriteString(s)
				}
			}
			// Content order is what makes this comparable, so walk it
			// explicitly rather than ranging over the map.
			if content, ok := v["content"].([]any); ok {
				for _, c := range content {
					walk(c)
				}
			}
		case []any:
			for _, c := range v {
				walk(c)
			}
		}
	}
	walk(node)
	return b.String()
}

// adfAttrProfile counts "nodeType.attrName" occurrences, including the
// attributes carried by marks, which is where Confluence's link metadata
// lives.
func adfAttrProfile(node any) map[string]int {
	profile := map[string]int{}
	var walk func(any, string)
	walk = func(n any, parentType string) {
		switch v := n.(type) {
		case map[string]any:
			nodeType, _ := v["type"].(string)
			if nodeType == "" {
				nodeType = parentType
			}
			if attrs, ok := v["attrs"].(map[string]any); ok {
				for k := range attrs {
					profile[nodeType+".attrs."+k]++
				}
			}
			if marks, ok := v["marks"].([]any); ok {
				for _, m := range marks {
					walk(m, nodeType)
				}
			}
			if content, ok := v["content"].([]any); ok {
				for _, c := range content {
					walk(c, nodeType)
				}
			}
		case []any:
			for _, c := range v {
				walk(c, parentType)
			}
		}
	}
	walk(node, "")
	return profile
}

func diffAttrProfiles(sent, stored map[string]int) (dropped, added []string) {
	for k, n := range sent {
		if stored[k] < n {
			dropped = append(dropped, fmt.Sprintf("%s (%d→%d)", k, n, stored[k]))
		}
	}
	for k, n := range stored {
		if sent[k] < n {
			added = append(added, fmt.Sprintf("%s (%d→%d)", k, sent[k], n))
		}
	}
	sort.Strings(dropped)
	sort.Strings(added)
	return dropped, added
}

// xhtmlText strips tags so storage-format bodies compare on their text.
// Confluence rewrites storage markup freely, so element-level equality would
// report drift on every write.
func xhtmlText(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '<':
			depth++
		case r == '>':
			if depth > 0 {
				depth--
			}
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// describeDrift renders drift for an operator: what was lost, and whether it
// cost them content or only normalization.
func describeDrift(d writeDrift, bodyFormat string) []string {
	var lines []string
	if d.TextChanged {
		lines = append(lines,
			fmt.Sprintf("Stored %s body does not match what was sent: text content differs (%d chars sent, %d stored).",
				bodyFormat, len(d.SentText), len(d.StoredText)),
			"The page was updated, but it does not contain the content supplied. Re-read the page before assuming the change landed.",
		)
		if diff := firstTextDifference(d.SentText, d.StoredText); diff != "" {
			lines = append(lines, "  first difference: "+diff)
		}
		return lines
	}
	if len(d.DroppedAttrs) > 0 {
		lines = append(lines, fmt.Sprintf("Confluence normalized the stored %s body. Text content is intact; these attributes were dropped:", bodyFormat))
		for _, a := range d.DroppedAttrs {
			lines = append(lines, "  - "+a)
		}
	}
	if len(d.AddedAttrs) > 0 {
		lines = append(lines, "Confluence added attributes that were not sent:")
		for _, a := range d.AddedAttrs {
			lines = append(lines, "  + "+a)
		}
	}
	return lines
}

// firstTextDifference locates where two texts diverge so the report points at
// the change rather than restating both documents.
func firstTextDifference(sent, stored string) string {
	limit := len(sent)
	if len(stored) < limit {
		limit = len(stored)
	}
	i := 0
	for i < limit && sent[i] == stored[i] {
		i++
	}
	if i == limit && len(sent) == len(stored) {
		return ""
	}
	return fmt.Sprintf("at offset %d — sent %q, stored %q", i, excerpt(sent, i), excerpt(stored, i))
}

func excerpt(s string, at int) string {
	if at > len(s) {
		return ""
	}
	end := at + 40
	if end > len(s) {
		end = len(s)
	}
	return s[at:end]
}

// verifyRequest carries what a post-write readback needs, so edit and create
// share one verification path.
type verifyRequest struct {
	opts        *root.Options
	client      *api.Client
	pageID      string
	bodyFormat  string
	sentContent string
	enabled     bool
}

// verifyStoredBody re-reads a page after a write and reports what Confluence
// actually stored. It only runs for exact body formats: markdown input is
// converted before sending, so the stored body is not comparable to it.
//
// Losing text is an error — the caller's content did not land. Normalization
// is reported and tolerated, because it is the server's prerogative and the
// content survived it.
func verifyStoredBody(ctx context.Context, req verifyRequest) error {
	if !req.enabled || req.sentContent == "" {
		return nil
	}
	if req.bodyFormat != bodyFormatADF && req.bodyFormat != bodyFormatXHTML {
		return nil
	}

	stored, err := getPageWithBodyFormat(ctx, req.client, req.pageID, req.bodyFormat)
	if err != nil {
		return fmt.Errorf("verifying stored page: %w — the write may have succeeded; re-read the page to confirm", err)
	}
	storedContent := bodyValue(stored, req.bodyFormat)
	if storedContent == "" {
		return fmt.Errorf("verifying stored page: no %s body returned — re-read the page to confirm the write", req.bodyFormat)
	}

	drift, err := compareStoredBody(req.sentContent, storedContent, req.bodyFormat)
	if err != nil {
		return fmt.Errorf("verifying stored page: %w", err)
	}
	if drift.Clean() {
		return nil
	}

	lines := describeDrift(drift, req.bodyFormat)
	if emitErr := cflpresent.Emit(req.opts, stderrLines(lines)); emitErr != nil {
		return emitErr
	}
	if drift.TextChanged {
		return fmt.Errorf("stored page content does not match what was sent")
	}
	return nil
}

func stderrLines(lines []string) *sharedpresent.OutputModel {
	return &sharedpresent.OutputModel{Sections: []sharedpresent.Section{
		&sharedpresent.MessageSection{
			Kind:    sharedpresent.MessageWarning,
			Message: strings.Join(lines, "\n"),
			Stream:  sharedpresent.StreamStderr,
		},
	}}
}

// bodyValue returns the page body in the requested representation.
func bodyValue(page *api.Page, bodyFormat string) string {
	if page == nil || page.Body == nil {
		return ""
	}
	switch bodyFormat {
	case bodyFormatADF:
		if page.Body.AtlasDocFormat != nil {
			return page.Body.AtlasDocFormat.Value
		}
	case bodyFormatXHTML:
		if page.Body.Storage != nil {
			return page.Body.Storage.Value
		}
	}
	return ""
}
