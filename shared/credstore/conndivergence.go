package credstore

import (
	"sort"
	"strings"

	"github.com/open-cli-collective/atlassian-go/auth"
)

// ConnProfile is the NON-secret connection projection the §2.2 /
// MON-5328 divergence detector works over. It deliberately has NO token
// field: the detector is secret-free by construction (never
// credstore.Section, which carries APIToken), so a pure, keyring/IO-free
// helper can never leak a credential.
type ConnProfile struct {
	URL        string
	Email      string
	AuthMethod string
	CloudID    string
}

// NamedConn is one origin-labeled connection candidate fed to the
// detector. Section labels the sub-key for the fail-loud message
// (default/cfl/jtk share one file, so a path alone is ambiguous); Path
// is the file the user edits to resolve a conflict.
type NamedConn struct {
	Label   string // e.g. "shared config", "legacy cfl config"
	Section string // "default" | "cfl" | "jtk" | "" (legacy single-section files)
	Path    string // file path for the remediation hint
	Conn    ConnProfile
}

// ConnConflict names ONE diverging connection field — the field plus the
// contributing source descriptors ("<label> <section>.<field> (<path>)"),
// NEVER the values (§1.12).
type ConnConflict struct {
	Field   string
	Sources []string
}

// canonURL / canonScalar normalize so textually-different-but-equivalent
// values don't false-conflict.
func canonURL(s string) string    { return NormalizeBaseURL(strings.TrimSpace(s)) }
func canonScalar(s string) string { return strings.TrimSpace(s) }

// usableBasic reports whether a source is, on its own, a complete basic
// connection (url+email) with no explicit non-basic auth. Such a source
// materializes an IMPLICIT auth_method=basic for conflict detection, so
// an implicit-basic `default` correctly conflicts with an explicit
// bearer per-tool/legacy source instead of silently unioning to bearer.
func usableBasic(c ConnProfile) bool {
	am := strings.ToLower(canonScalar(c.AuthMethod))
	return canonURL(c.URL) != "" && canonScalar(c.Email) != "" &&
		(am == "" || am == auth.AuthMethodBasic)
}

// effectiveAuth is the auth_method a source contributes to detection:
// an explicit value as-is; else implicit "basic" only if the source is a
// usable basic connection; else "" (no opinion — a fragmentary source
// like cloud_id-only must not vote an auth method).
func effectiveAuth(c ConnProfile) string {
	if am := strings.ToLower(canonScalar(c.AuthMethod)); am != "" {
		return am
	}
	if usableBasic(c) {
		return auth.AuthMethodBasic
	}
	return ""
}

// fieldVal is the canonical value a source contributes for field, or ""
// for "no opinion" (absent → does not participate; never a competing
// value, so a partial `default` stays compatible with a fuller source).
func fieldVal(c ConnProfile, field string) string {
	switch field {
	case "url":
		return canonURL(c.URL)
	case "email":
		return canonScalar(c.Email)
	case "auth_method":
		return effectiveAuth(c)
	case "cloud_id":
		return canonScalar(c.CloudID)
	}
	return ""
}

// canonConn is the fully-resolved single-source normal form: the same
// canonicalization DetectConnDivergence applies (url/email/cloud_id
// canonical, auth_method via effectiveAuth) plus the basic-materialization
// fallback it does once the union is otherwise resolved. Comparing two
// profiles via canonConn answers "do these resolve to the same
// connection?" without false-diffing raw-vs-normalized — e.g. an
// on-disk default that omits auth_method must compare EQUAL to a
// detector `chosen` that materialized it to basic.
func canonConn(c ConnProfile) ConnProfile {
	am := effectiveAuth(c)
	if am == "" {
		am = auth.AuthMethodBasic
	}
	return ConnProfile{
		URL:        canonURL(c.URL),
		Email:      canonScalar(c.Email),
		AuthMethod: am,
		CloudID:    canonScalar(c.CloudID),
	}
}

func sourceDesc(n NamedConn, field string) string {
	key := field
	if n.Section != "" {
		key = n.Section + "." + field
	}
	if n.Path != "" {
		return n.Label + " " + key + " (" + n.Path + ")"
	}
	return n.Label + " " + key
}

// DetectConnDivergence is the pure §2.2/MON-5328 connection resolver
// over the full named candidate set. Field-wise UNION (not whole-profile
// collapse): for each of url/email/auth_method/cloud_id independently it
// collects the distinct non-empty canonical values; a field with ≥2
// distinct values is a conflict (named, no values); a field with exactly
// one agreed value is folded into chosen. An all-empty/fragmentary
// source contributes nothing it has no opinion on, so a partial default
// is compatible with a fuller legacy source. If no conflicts and the
// resolved auth_method is still empty, it materializes basic (the
// codebase-wide default). Pure: no IO, no keyring, no secret.
func DetectConnDivergence(sources []NamedConn) (chosen ConnProfile, conflicts []ConnConflict) {
	fields := []string{"url", "email", "auth_method", "cloud_id"}
	resolved := map[string]string{}
	for _, field := range fields {
		seen := map[string][]string{} // canonical value -> source descriptors
		var order []string
		for _, src := range sources {
			v := fieldVal(src.Conn, field)
			if v == "" {
				continue
			}
			if _, ok := seen[v]; !ok {
				order = append(order, v)
			}
			seen[v] = append(seen[v], sourceDesc(src, field))
		}
		switch len(order) {
		case 0:
			// no opinion anywhere
		case 1:
			resolved[field] = order[0]
		default:
			var descs []string
			for _, v := range order {
				descs = append(descs, seen[v]...)
			}
			sort.Strings(descs)
			conflicts = append(conflicts, ConnConflict{Field: field, Sources: descs})
		}
	}
	if len(conflicts) > 0 {
		return ConnProfile{}, conflicts
	}
	chosen = ConnProfile{
		URL:        resolved["url"],
		Email:      resolved["email"],
		AuthMethod: resolved["auth_method"],
		CloudID:    resolved["cloud_id"],
	}
	if chosen.AuthMethod == "" {
		chosen.AuthMethod = auth.AuthMethodBasic
	}
	return chosen, nil
}
