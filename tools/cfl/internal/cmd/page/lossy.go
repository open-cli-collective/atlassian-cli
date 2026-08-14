package page

import (
	"fmt"
	"regexp"
	"strings"
)

// Confluence's ADF export does not carry everything its storage
// representation does, so reading a page as ADF and writing it back destroys
// content the caller never touched. Two losses are established by
// experiment:
//
//   - Any mark paired with `code` is dropped. Sending code+strong stores
//     correctly and reads back as code alone, while storage returns
//     <code><strong>…</strong></code> intact. strong+em survives, so this is
//     specific to code.
//   - Internal link elements are not represented. <ac:link> anchors come back
//     as plain links, and writing that back turns a short anchor reference
//     into an expanded URL.
//
// A caller cannot see either of these: the ADF they hold is already missing
// the marks, so comparing what they sent against what was stored agrees. The
// only moment the loss is preventable is before the write, by looking at the
// page as it exists now.

// lossyConstruct is something the ADF representation cannot carry.
type lossyConstruct struct {
	name    string
	pattern *regexp.Regexp
	// detail explains the consequence in the operator's terms.
	detail string
}

var lossyConstructs = []lossyConstruct{
	{
		name:    "internal links",
		pattern: regexp.MustCompile(`<ac:link[\s>]`),
		detail:  "anchor and page references become plain expanded URLs",
	},
}

var emphasisTagRE = regexp.MustCompile(`</?(strong|em|code)>`)

// countEmphasisedCode reports how many inline code spans sit inside bold or
// italic, or vice versa. Nesting is what matters and the two need not be
// adjacent, so this tracks open tags rather than matching a shape: a regex
// would either miss intervening markup or, with RE2's lack of lookahead,
// match across unrelated elements.
func countEmphasisedCode(markup string) int {
	var depth struct{ emphasis, code int }
	found := 0
	for _, m := range emphasisTagRE.FindAllStringSubmatch(markup, -1) {
		closing := strings.HasPrefix(m[0], "</")
		switch m[1] {
		case "code":
			if closing {
				depth.code--
			} else {
				if depth.emphasis > 0 {
					found++
				}
				depth.code++
			}
		default:
			if closing {
				depth.emphasis--
			} else {
				if depth.code > 0 {
					found++
				}
				depth.emphasis++
			}
		}
	}
	return found
}

// lossyFinding names one construct a write would destroy.
type lossyFinding struct {
	Construct string
	Detail    string
	Count     int
}

// findLossyConstructs reports what an ADF write would destroy in the page as
// it currently stands. It inspects the existing storage body rather than the
// caller's content, because the caller's ADF has already lost these things —
// that is the whole problem.
//
// Comment and CDATA spans are removed first, for the same reason
// storageProfile removes them: storage carries macro bodies verbatim inside
// CDATA, where angle brackets are content rather than markup, and a match
// there would refuse a write over nothing.
func findLossyConstructs(storageBody string) []lossyFinding {
	if strings.TrimSpace(storageBody) == "" {
		return nil
	}
	markup := storageInertRE.ReplaceAllString(storageBody, "")
	var found []lossyFinding
	for _, c := range lossyConstructs {
		if n := len(c.pattern.FindAllString(markup, -1)); n > 0 {
			found = append(found, lossyFinding{Construct: c.name, Detail: c.detail, Count: n})
		}
	}
	if n := countEmphasisedCode(markup); n > 0 {
		found = append(found, lossyFinding{
			Construct: "emphasis on code spans",
			Detail:    "bold or italic wrapping inline code is dropped",
			Count:     n,
		})
	}
	return found
}

// lossyFormatError refuses a write that would silently degrade the page, and
// says how to proceed deliberately. The escape hatch is a flag rather than a
// format choice so that overwriting known-lossy content is visible in shell
// history instead of implied.
func lossyFormatError(findings []lossyFinding, bodyFormat string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "refusing to write this page as %s: the %s representation cannot carry content it currently has, and writing would remove it\n", bodyFormat, bodyFormat)
	for _, f := range findings {
		fmt.Fprintf(&b, "  - %s (%d) — %s\n", f.Construct, f.Count, f.Detail)
	}
	b.WriteString("\nUse --body-format xhtml to edit this page without losing them.\n")
	b.WriteString("Pass --allow-lossy to write it as " + bodyFormat + " anyway.")
	return fmt.Errorf("%s", b.String())
}

// orUnknown keeps a refusal honest when the cause could not be captured.
func orUnknown(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return "cause unknown"
	}
	return reason
}
