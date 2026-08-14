package page

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

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
	// VisibleSent and VisibleStored count the characters a reader sees, so
	// the numbers reported are facts about the document rather than
	// positions in the internal fingerprint.
	VisibleSent   int
	VisibleStored int
	// AtomsChanged reports that embedded content (cards, images, mentions,
	// breaks) differs even where the visible text does not.
	AtomsChanged bool
	// SentVisible and StoredVisible are the reader-visible text, which is
	// where a reported position has to be measured for it to mean anything
	// to the operator.
	SentVisible   string
	StoredVisible string
	// AtomChanges names embedded node types whose counts moved. Attribute
	// diffs cannot cover this: hardBreak and rule carry no attributes, so
	// without it a change confined to them has nothing to report.
	AtomChanges []string

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
			TextChanged:   sentText != storedText,
			SentText:      sentText,
			StoredText:    storedText,
			VisibleSent:   len([]rune(sentText)),
			VisibleStored: len([]rune(storedText)),
			SentVisible:   sentText,
			StoredVisible: storedText,
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

	sentText, storedText := adfContent(sentDoc), adfContent(storedDoc)
	sentVisible, storedVisible := visibleText(sentDoc), visibleText(storedDoc)
	dropped, added := diffAttrProfiles(adfAttrProfile(sentDoc), adfAttrProfile(storedDoc))
	return writeDrift{
		TextChanged:   sentText != storedText,
		SentText:      sentText,
		StoredText:    storedText,
		VisibleSent:   len([]rune(sentVisible)),
		VisibleStored: len([]rune(storedVisible)),
		AtomsChanged:  sentText != storedText && sentVisible == storedVisible,
		SentVisible:   sentVisible,
		StoredVisible: storedVisible,
		AtomChanges:   diffAtomProfiles(atomProfile(sentDoc), atomProfile(storedDoc)),
		DroppedAttrs:  dropped,
		AddedAttrs:    added,
	}, nil
}

// contentAtoms are ADF leaf nodes that carry content the reader sees but
// hold no text of their own. They must count toward the content fingerprint:
// otherwise dropping an entire card, image or mention leaves the text
// identical and the loss is reported as harmless attribute normalization.
var contentAtoms = map[string]bool{
	"inlineCard":      true,
	"blockCard":       true,
	"embedCard":       true,
	"media":           true,
	"mediaInline":     true,
	"mention":         true,
	"emoji":           true,
	"status":          true,
	"date":            true,
	"extension":       true,
	"inlineExtension": true,
	"bodiedExtension": true,
	// Attribute-less atoms: dropping one moves neither the text nor the
	// attribute profile, so without naming them here the loss is invisible.
	"hardBreak": true,
	"rule":      true,
}

// adfContent renders a document's content fingerprint in document order:
// text as itself, and each atom named in contentAtoms as a marker carrying
// its identifying attributes.
//
// The atom set is an allowlist, so the fingerprint is a sound signal and not
// a complete one: a differing fingerprint always means the content changed,
// while an identical one means nothing outside the allowlist moved. A node
// type ADF gains later goes unnoticed until it is added here.
func adfContent(node any) string {
	var b strings.Builder
	var walk func(any)
	walk = func(n any) {
		switch v := n.(type) {
		case map[string]any:
			nodeType, _ := v["type"].(string)
			switch {
			case nodeType == "text":
				if s, ok := v["text"].(string); ok {
					b.WriteString(s)
				}
			case contentAtoms[nodeType]:
				// Identity, not just presence: swapping one card for another
				// keeps the count identical.
				b.WriteString("\x00" + nodeType + "(" + atomIdentity(v) + ")")
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

// atomProfile counts content-bearing atoms by type.
func atomProfile(node any) map[string]int {
	profile := map[string]int{}
	var walk func(any)
	walk = func(n any) {
		switch v := n.(type) {
		case map[string]any:
			if t, _ := v["type"].(string); contentAtoms[t] {
				profile[t]++
			}
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
	return profile
}

// diffAtomProfiles names atom types whose counts moved, in sorted order.
func diffAtomProfiles(sent, stored map[string]int) []string {
	seen := map[string]bool{}
	var out []string
	for k := range sent {
		seen[k] = true
	}
	for k := range stored {
		seen[k] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if sent[k] != stored[k] {
			out = append(out, fmt.Sprintf("%s (%d→%d)", k, sent[k], stored[k]))
		}
	}
	return out
}

// visibleText is only the characters a reader sees, with no atom markers, so
// a length taken from it is a statement about the document.
func visibleText(node any) string {
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

// atomIdentity summarizes an atom's identifying attributes so a substitution
// is visible, using a stable order.
func atomIdentity(node map[string]any) string {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return ""
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, attrs[k]))
	}
	return strings.Join(parts, ",")
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

// verifyRequest carries what a post-write readback needs, so page edit and
// page create share one verification path.
type verifyRequest struct {
	opts        *root.Options
	client      *api.Client
	pageID      string
	bodyFormat  string
	sentContent string
	enabled     bool
	// storageBefore is the page's storage body read before the write. It is
	// the representation that survives an ADF round trip, so it is what
	// reveals loss the caller inherited from their own read.
	storageBefore string
}

// verificationApplies reports whether a write will be verified. Markdown is
// converted before sending, so the stored body is not comparable to what the
// caller supplied and no read is worth paying for.
func verificationApplies(bodyFormat string, hasNewContent, noVerify bool) bool {
	if !hasNewContent || noVerify {
		return false
	}
	return bodyFormat == bodyFormatADF || bodyFormat == bodyFormatXHTML
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
	lostElements := verifyStorageLoss(ctx, req)
	if drift.Clean() && len(lostElements) == 0 {
		return nil
	}

	off := diffOffset(drift.SentVisible, drift.StoredVisible)
	finding := cflpresent.WriteDrift{
		BodyFormat:    req.bodyFormat,
		TextChanged:   drift.TextChanged,
		VisibleSent:   drift.VisibleSent,
		VisibleStored: drift.VisibleStored,
		AtomsChanged:  drift.AtomsChanged,
		DiffOffset:    off,
		SentExcerpt:   readableExcerpt(drift.SentVisible, off),
		StoredExcerpt: readableExcerpt(drift.StoredVisible, off),
		AtomChanges:   drift.AtomChanges,
		DroppedAttrs:  drift.DroppedAttrs,
		AddedAttrs:    drift.AddedAttrs,
		LostElements:  lostElements,
	}
	if emitErr := cflpresent.Emit(req.opts, cflpresent.PagePresenter{}.PresentWriteDrift(finding)); emitErr != nil {
		return emitErr
	}
	if drift.TextChanged {
		return fmt.Errorf("stored page content does not match what was sent")
	}
	return nil
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

// diffOffset reports the character position where two reader-visible texts
// first differ, or -1 when they match. Measuring on the visible text is what
// makes the number a position in the document rather than in the compared
// representation.
func diffOffset(sent, stored string) int {
	a, b := []rune(sent), []rune(stored)
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	i := 0
	for i < limit && a[i] == b[i] {
		i++
	}
	if i == limit && len(a) == len(b) {
		return -1
	}
	return i
}

// readableExcerpt returns part of the reader-visible text, starting at a
// character position, for quoting back in a report.
func readableExcerpt(s string, at int) string {
	r := []rune(s)
	if at < 0 || at > len(r) {
		return ""
	}
	end := at + 40
	if end > len(r) {
		end = len(r)
	}
	return string(r[at:end])
}

// Comparing what was sent against what was stored cannot see loss the caller
// inherited from their own read. Confluence's atlas_doc_format representation
// of a page does not always carry marks its storage representation does — em
// and strong wrapping a code span in a table cell are dropped — so reading a
// page as ADF and writing it straight back destroys them, and both sides of a
// sent-versus-stored comparison carry the loss equally.
//
// The storage representation is the one that survives the round trip, so it
// is the one worth watching. Element types whose counts fall across a write
// are reported: text edits move no element counts, while losing emphasis, a
// code span, or a link is collateral a caller almost never intends.

var storageElementRE = regexp.MustCompile(`<(\w[\w-]*)[\s/>]`)

// storageProfile counts elements in a storage-format body.
func storageProfile(body string) map[string]int {
	profile := map[string]int{}
	for _, m := range storageElementRE.FindAllStringSubmatch(body, -1) {
		profile[m[1]]++
	}
	return profile
}

// diffStorageLoss names element types that became less frequent, in sorted
// order. Additions are not reported: adding content is what an edit is for.
func diffStorageLoss(before, after map[string]int) []string {
	var lost []string
	for name, n := range before {
		if after[name] < n {
			lost = append(lost, fmt.Sprintf("%s (%d→%d)", name, n, after[name]))
		}
	}
	sort.Strings(lost)
	return lost
}

// verifyStorageLoss compares the page's storage body across the write. An
// empty result means either nothing was lost or the comparison could not be
// made; it never reports loss it did not observe.
func verifyStorageLoss(ctx context.Context, req verifyRequest) []string {
	if req.storageBefore == "" {
		return nil
	}
	after, err := readStorageBody(ctx, req.client, req.pageID)
	if err != nil || after == "" {
		return nil
	}
	return diffStorageLoss(storageProfile(req.storageBefore), storageProfile(after))
}

// readStorageBody fetches a page's storage representation, or "" when it is
// unavailable. Callers treat that as "cannot compare" rather than as "no
// loss", so a failure here never manufactures a clean result.
func readStorageBody(ctx context.Context, client *api.Client, pageID string) (string, error) {
	page, err := client.GetPage(ctx, pageID, &api.GetPageOptions{BodyFormat: apiBodyFormat(bodyFormatXHTML)})
	if err != nil {
		return "", err
	}
	return bodyValue(page, bodyFormatXHTML), nil
}
