# Release Notes

## v0.2.4 (2026-01-11)

Fixes markdown conversion to preserve code blocks created in Confluence's web UI.

### Changes
- Preserve code blocks from Confluence UI pages in markdown output ([#15](https://github.com/rianjs/confluence-cli/issues/15))

---

## v0.2.3 (2026-01-11)

Improves error messages when invalid page status values are provided.

### Changes
- Reject invalid `--status` values with helpful error message ([#17](https://github.com/rianjs/confluence-cli/issues/17))

---

## v0.2.2 (2026-01-11)

Fixes `page copy` when the `--space` flag is omitted.

### Changes
- Resolve space key from spaceId for page copy ([#14](https://github.com/rianjs/confluence-cli/issues/14))

---

## v0.2.1 (2026-01-11)

Fixes attachment downloads which were broken due to API endpoint changes.

### Changes
- Use downloadLink from attachment metadata for downloads

---

## v0.2.0 (2026-01-10)

Adds page copy and attachment delete commands, plus security hardening for attachment downloads.

### Features
- Add `page copy` command to duplicate pages within or across spaces
- Add `attachment delete` command with confirmation prompt
- Add automated releases via release-please
- Warn before overwriting existing files in attachment download

### Bug Fixes
- Sanitize attachment download filenames to prevent path traversal
- Pin golangci-lint to v2 in CI
