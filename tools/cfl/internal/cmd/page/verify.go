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

	// ParamsDropped and ParamsAdded name macro parameters whose occurrence
	// counts fell or rose, in sorted order. Their order within a macro is
	// the server's to choose, so only an added, dropped or edited parameter
	// appears here. They are kept apart from TextChanged because the page
	// text is intact: saying otherwise would make the report assert
	// something untrue.
	ParamsDropped []string
	ParamsAdded   []string
}

// Clean reports whether the stored body matched what was sent.
func (d writeDrift) Clean() bool {
	return !d.TextChanged && len(d.DroppedAttrs) == 0 && len(d.AddedAttrs) == 0 &&
		len(d.ParamsDropped) == 0 && len(d.ParamsAdded) == 0
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
		paramsDropped, paramsAdded := diffMacroParameters(sent, stored)
		return writeDrift{
			TextChanged:   sentText != storedText,
			SentText:      sentText,
			StoredText:    storedText,
			VisibleSent:   len([]rune(sentText)),
			VisibleStored: len([]rune(storedText)),
			SentVisible:   sentText,
			StoredVisible: storedText,
			ParamsDropped: paramsDropped,
			ParamsAdded:   paramsAdded,
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

// blankInert masks comment and CDATA spans with spaces, preserving every
// offset. Storage format carries macro bodies verbatim inside CDATA, so markup
// quoted there is content rather than page configuration.
//
// Both parameter scans match against this one masked view and then index back
// into the original, so they cannot disagree about where a parameter is: a
// single notion of what counts as markup serves the text comparison and the
// parameter profile alike.
func blankInert(s string) string {
	spans := storageInertRE.FindAllStringIndex(s, -1)
	if len(spans) == 0 {
		return s
	}
	b := []byte(s)
	for _, span := range spans {
		for i := span[0]; i < span[1]; i++ {
			b[i] = ' '
		}
	}
	return string(b)
}

// stripMacroParameters removes parameter elements from a storage body, leaving
// comment and CDATA spans as the content they are. Matching happens on the
// masked view while the text is cut from the original, so a parameter quoted
// inside a macro body stays in the compared text and a real one does not.
func stripMacroParameters(s string) string {
	matches := acParameterPairRE.FindAllStringIndex(blankInert(s), -1)
	if len(matches) == 0 {
		return s
	}
	var b strings.Builder
	last := 0
	for _, m := range matches {
		b.WriteString(s[last:m[0]])
		last = m[1]
	}
	b.WriteString(s[last:])
	return b.String()
}

// acParameterPairRE matches a macro parameter written as an open/close pair,
// capturing its attributes and its value. The `[^/]` before the closing angle
// bracket keeps it from starting on a self-closing tag, which has no value and
// no closing tag: matching one would run the value capture on to the *next*
// parameter's closing tag and swallow every element in between, including page
// content. (?s) so a value containing newlines is still one match.
var acParameterPairRE = regexp.MustCompile(`(?s)<ac:parameter\b([^>]*[^/])?>(.*?)</ac:parameter>`)

// acParameterEmptyRE matches a self-closing parameter, which carries a name
// but no value.
var acParameterEmptyRE = regexp.MustCompile(`<ac:parameter\b([^>]*?)/>`)

// acMacroOpenRE marks where each macro begins, so a parameter can be attributed
// to the macro that contains it.
var acMacroOpenRE = regexp.MustCompile(`<ac:structured-macro\b`)

// acParameterNameRe pulls a parameter's name out of its attributes.
var acParameterNameRe = regexp.MustCompile(`ac:name\s*=\s*"([^"]*)"`)

// macroParameterProfile counts a body's macro parameters. Confluence returns a
// macro's parameters in an order of its own choosing, so they are compared as a
// multiset rather than a sequence: a reordering within one macro is not a
// change, while a parameter that is dropped, added or edited still is.
//
// Each key is scoped to the macro that holds the parameter, identified by how
// many macros open before it. Without that scope a parameter moving between two
// macros carrying the same name and value would read as a reordering, and the
// swap this guard exists to catch would pass.
//
// Comment and CDATA spans are masked first by blankInert, the same view
// stripMacroParameters matches against, so the text comparison and this profile
// agree on which parameters are real and which are quoted content.
func macroParameterProfile(s string) map[string]int {
	s = blankInert(s)
	macroOpens := acMacroOpenRE.FindAllStringIndex(s, -1)
	// macroAt reports how many macros have opened before position i, which
	// identifies the macro a parameter sits in.
	macroAt := func(i int) int {
		n := 0
		for _, m := range macroOpens {
			if m[0] < i {
				n++
			}
		}
		return n
	}

	profile := map[string]int{}
	record := func(start int, attrs, value string) {
		name := ""
		if n := acParameterNameRe.FindStringSubmatch(attrs); n != nil {
			name = n[1]
		}
		// Collapse whitespace exactly as the surrounding text comparison
		// does, so a value that only had its spacing normalized is not
		// reported as an edit.
		value = strings.Join(strings.Fields(value), " ")
		profile[fmt.Sprintf("macro %d parameter %s=%s", macroAt(start), name, value)]++
	}
	for _, m := range acParameterPairRE.FindAllStringSubmatchIndex(s, -1) {
		attrs := ""
		if m[2] >= 0 {
			attrs = s[m[2]:m[3]]
		}
		record(m[0], attrs, s[m[4]:m[5]])
	}
	for _, m := range acParameterEmptyRE.FindAllStringSubmatchIndex(s, -1) {
		record(m[0], s[m[2]:m[3]], "")
	}
	return profile
}

// diffMacroParameters names the macro parameters that did not survive the
// write as they were sent, dropped and added kept apart so the presenter owns
// how they are marked. Two empty results mean every parameter came back,
// whatever order the server chose to return them in.
func diffMacroParameters(sent, stored string) (dropped, added []string) {
	return diffAttrProfiles(macroParameterProfile(sent), macroParameterProfile(stored))
}

// xhtmlText strips tags so storage-format bodies compare on their text.
// Confluence rewrites storage markup freely, so element-level equality would
// report drift on every write.
func xhtmlText(s string) string {
	// Macro parameter values are configuration, not text a reader sees, and
	// Confluence reorders them within a macro. Left in the text stream, a
	// reordering reads as rewritten content and fails an otherwise clean
	// write; macroParameterProfile compares them order-independently instead.
	s = stripMacroParameters(s)
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
	// storageBeforeErr explains a missing baseline so the operator learns
	// the check could not run, rather than seeing the silence of a clean
	// write.
	storageBeforeErr string
	// comparePriorState is false when there is no prior state to compare
	// against, as on a create. That is not a failed comparison and must not
	// be reported as one.
	comparePriorState bool
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
	// An xhtml write already read the storage body back; reuse it.
	reuse := ""
	if req.bodyFormat == bodyFormatXHTML {
		reuse = storedContent
	}
	storage := compareStorage(ctx, req, reuse)
	if drift.Clean() && len(storage.Lost) == 0 && !storage.Unavailable {
		return nil
	}

	off := diffOffset(drift.SentVisible, drift.StoredVisible)
	finding := cflpresent.WriteDrift{
		BodyFormat:          req.bodyFormat,
		TextChanged:         drift.TextChanged,
		VisibleSent:         drift.VisibleSent,
		VisibleStored:       drift.VisibleStored,
		AtomsChanged:        drift.AtomsChanged,
		DiffOffset:          off,
		SentExcerpt:         readableExcerpt(drift.SentVisible, off),
		StoredExcerpt:       readableExcerpt(drift.StoredVisible, off),
		AtomChanges:         drift.AtomChanges,
		ParamsDropped:       drift.ParamsDropped,
		ParamsAdded:         drift.ParamsAdded,
		DroppedAttrs:        drift.DroppedAttrs,
		AddedAttrs:          drift.AddedAttrs,
		LostElements:        storage.Lost,
		StorageUncomparable: storage.Unavailable,
		StorageReason:       storage.Reason,
		LossCause:           storageLossCause(req.bodyFormat),
	}
	if emitErr := cflpresent.Emit(req.opts, cflpresent.PagePresenter{}.PresentWriteDrift(finding)); emitErr != nil {
		return emitErr
	}
	if drift.TextChanged {
		return fmt.Errorf("stored page content does not match what was sent")
	}
	// A parameter the server did not store as sent is a failed write too. It
	// was fatal before macro parameters were excluded from the text compare,
	// and nothing about excluding them makes it safe to pass.
	if len(drift.ParamsDropped) > 0 || len(drift.ParamsAdded) > 0 {
		return fmt.Errorf("stored page does not hold the macro parameters that were sent")
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

// Storage format is XHTML with namespaced elements: macros arrive as
// <ac:structured-macro>, attachment references as <ri:attachment>. The
// namespace prefix is part of the name, so it has to be matched or a lost
// macro is a loss the profile cannot see.
var storageElementRE = regexp.MustCompile(`<([A-Za-z][\w-]*(?::[\w-]+)?)[\s/>]`)

var storageInertRE = regexp.MustCompile(`(?s)<!--.*?-->|<!\[CDATA\[.*?\]\]>`)

// storageProfile counts elements in a storage-format body. Comment and CDATA
// spans are removed first: storage format carries macro bodies verbatim
// inside CDATA, and angle brackets there are content, not markup.
func storageProfile(body string) map[string]int {
	profile := map[string]int{}
	for _, m := range storageElementRE.FindAllStringSubmatch(storageInertRE.ReplaceAllString(body, ""), -1) {
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

// storageLossCause names why formatting can vanish for the format in use.
// An ADF round trip drops marks Confluence's storage body carries; a storage
// write loses only what the caller's own pipeline removed.
func storageLossCause(bodyFormat string) string {
	if bodyFormat == bodyFormatADF {
		return "Reading a page as ADF and writing it back does not always preserve marks its storage form carries."
	}
	return "The submitted storage body did not carry these elements."
}

// storageComparison reports what the storage bodies showed, and says so when
// they could not be compared at all. A silent nil would be indistinguishable
// from a clean write, which is the failure this check exists to remove.
type storageComparison struct {
	Lost        []string
	Unavailable bool
	Reason      string
}

// compareStorage diffs the page's storage body across the write. storedAfter
// may be supplied by a caller that already read it, so an xhtml write does
// not pay for the same fetch twice.
func compareStorage(ctx context.Context, req verifyRequest, storedAfter string) storageComparison {
	if !req.comparePriorState {
		return storageComparison{}
	}
	if req.storageBefore == "" {
		return storageComparison{Unavailable: true, Reason: req.storageBeforeErr}
	}
	after := storedAfter
	if after == "" {
		body, err := readStorageBody(ctx, req.client, req.pageID)
		if err != nil {
			return storageComparison{Unavailable: true, Reason: err.Error()}
		}
		after = body
	}
	if after == "" {
		return storageComparison{Unavailable: true, Reason: "the page returned no storage body"}
	}
	return storageComparison{Lost: diffStorageLoss(storageProfile(req.storageBefore), storageProfile(after))}
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
