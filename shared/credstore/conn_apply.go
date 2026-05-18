package credstore

import (
	"errors"
	"sort"
	"strings"
)

// This file holds the tool-agnostic apply-layer glue for the §2.2
// (MON-5328) single-source connection model: building the named
// candidate set for DetectConnDivergence and rendering the fail-loud
// error. It lived duplicated verbatim in both tools' reconcile.go;
// centralized here (next to the detector) so a fix lands once. Pure and
// secret-free — no token field, no IO, no keyring.

func legacyConn(l *LegacyCreds) ConnProfile {
	return ConnProfile{URL: l.URL, Email: l.Email, AuthMethod: l.AuthMethod, CloudID: l.CloudID}
}

// hasConn reports whether a profile carries any connection field.
func hasConn(c ConnProfile) bool {
	return c.URL != "" || c.Email != "" || c.AuthMethod != "" || c.CloudID != ""
}

// effectiveConn merges a pre-MON-5328 per-tool section over default
// (the old per-field-merge semantics) so the detector compares what the
// tool actually USED to resolve.
func effectiveConn(def, sec SharedLegacyConn) ConnProfile {
	pick := func(o, d string) string {
		if o != "" {
			return o
		}
		return d
	}
	return ConnProfile{
		URL:        pick(sec.URL, def.URL),
		Email:      pick(sec.Email, def.Email),
		AuthMethod: pick(sec.AuthMethod, def.AuthMethod),
		CloudID:    pick(sec.CloudID, def.CloudID),
	}
}

// sharedConnHasField reports whether a raw pre-MON-5328 per-tool section
// set ANY connection field of its own. A per-tool section with none is
// "no opinion": it must NOT contribute a phantom candidate (effectiveConn
// would otherwise echo the default's values under a `cfl`/`jtk` label
// and pollute conflict messages).
func sharedConnHasField(s SharedLegacyConn) bool {
	return s.URL != "" || s.Email != "" || s.AuthMethod != "" || s.CloudID != ""
}

// ConnCandidates assembles the origin-labeled connection candidate set
// for DetectConnDivergence: the shared `default`, the pre-MON-5328
// per-tool sections AS effective overrides (default ⊕ section) — but
// ONLY when that per-tool section actually set a connection field of its
// own — and the legacy cfl/jtk files. Tool-agnostic: the candidate set
// is the same whichever tool runs init.
func ConnCandidates(
	sharedPath string,
	def Section,
	proj *SharedLegacyProjection,
	cflLegacy, jtkLegacy *LegacyCreds,
) []NamedConn {
	var out []NamedConn
	add := func(label, section, path string, c ConnProfile) {
		if !hasConn(c) {
			return
		}
		out = append(out, NamedConn{Label: label, Section: section, Path: path, Conn: c})
	}
	add("shared config", "default", sharedPath, ConnProfile{
		URL: def.URL, Email: def.Email, AuthMethod: def.AuthMethod, CloudID: def.CloudID,
	})
	if proj != nil {
		if sharedConnHasField(proj.CFL) {
			add("shared config", "cfl", sharedPath, effectiveConn(proj.Default, proj.CFL))
		}
		if sharedConnHasField(proj.JTK) {
			add("shared config", "jtk", sharedPath, effectiveConn(proj.Default, proj.JTK))
		}
	}
	if cflLegacy != nil {
		add("legacy cfl config", "", cflLegacy.Path, legacyConn(cflLegacy))
	}
	if jtkLegacy != nil {
		add("legacy jtk config", "", jtkLegacy.Path, legacyConn(jtkLegacy))
	}
	return out
}

// ConnConflictError renders the §2.2 fail-loud message: every
// conflicting field with its source descriptors (no values, §1.12) and
// a remediation listing every distinct candidate file PATH. Paths come
// from the structured NamedConn set (never re-parsed out of formatted
// descriptor strings — a path may contain '(') and are sorted
// (deterministic across re-runs).
func ConnConflictError(conflicts []ConnConflict, candidates []NamedConn, tool string) error {
	pathSet := map[string]struct{}{}
	for _, c := range candidates {
		if c.Path != "" {
			pathSet[c.Path] = struct{}{}
		}
	}
	paths := make([]string, 0, len(pathSet))
	for p := range pathSet {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var b strings.Builder
	b.WriteString("connection config diverges across sources; init will not pick a winner. Conflicts:\n")
	for _, c := range conflicts {
		b.WriteString("  - ")
		b.WriteString(c.Field)
		b.WriteString(": ")
		b.WriteString(strings.Join(c.Sources, ", "))
		b.WriteString("\n")
	}
	b.WriteString("Resolve by editing/removing all but one connection in: ")
	b.WriteString(strings.Join(paths, ", "))
	b.WriteString(" — then re-run ")
	b.WriteString(tool)
	b.WriteString(" init. (No values shown; secrets live only in the OS keyring.)")
	return errors.New(b.String())
}
