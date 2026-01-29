# Release Notes

## v0.10.0 (2026-01-17)

Adds Windows ARM64 binary distribution.

### Features
- Add Windows ARM64 build target to releases ([#70](https://github.com/open-cli-collective/confluence-cli/issues/70))
  - Download `cfl_X.Y.Z_windows_arm64.zip` from GitHub Releases
  - Enables future Windows package manager support (Chocolatey, Winget) for ARM64 devices

---

## v0.9.0 (2026-01-16)

Enables clean roundtrip editing for pages with macros.

### Features
- Add `--content-only` flag to `page view` to output only content without metadata headers ([#57](https://github.com/open-cli-collective/confluence-cli/issues/57))

### Example
```bash
# Roundtrip editing with macros preserved
cfl page view 12345 --show-macros --content-only | cfl page edit 12345 --legacy
```

---

## v0.8.1 (2026-01-16)

Bug fixes for page editing and content validation.

### Bug Fixes
- Allow `--parent` flag to move page without requiring content changes ([#60](https://github.com/open-cli-collective/confluence-cli/issues/60))
- Validate empty content client-side before API call ([#59](https://github.com/open-cli-collective/confluence-cli/issues/59))
- Correct Homebrew tap reference in installation docs

---

## v0.8.0 (2026-01-15)

Adds roundtrip support for common Confluence macros.

### Features
- Support TOC, panel, and expand macros with bracket-style markdown syntax ([#51](https://github.com/open-cli-collective/confluence-cli/issues/51))
  - **TOC**: `[TOC]` or `[TOC maxLevel=3]`
  - **Panels**: `[INFO]...[/INFO]`, `[WARNING]...[/WARNING]`, `[NOTE]...[/NOTE]`, `[TIP]...[/TIP]`
  - **Expand**: `[EXPAND title="Click"]...[/EXPAND]`

### Internal
- Refactored macro parser to tokenizer/parser architecture for maintainability

---

## v0.7.0 (2026-01-14)

Move pages to a new parent without losing history.

### Features
- Add `--parent` flag to `cfl page edit` to move pages to a different parent ([#42](https://github.com/open-cli-collective/confluence-cli/issues/42))

---

## v0.6.0 (2026-01-14)

Add shell tab completion support for bash, zsh, fish, and PowerShell.

### Features
- Add `cfl completion` command with subcommands for bash, zsh, fish, and PowerShell ([#43](https://github.com/open-cli-collective/confluence-cli/issues/43))

---

## v0.5.0 (2026-01-13)

Pages now use the modern cloud editor (ADF) format by default.

### Features
- Use cloud editor (ADF) format for page creation by default ([#39](https://github.com/open-cli-collective/confluence-cli/issues/39))
- Add `--legacy` flag to create/edit pages in legacy storage format
- Add format mismatch warning when editing cloud pages with `--legacy`

---

## v0.4.0 (2026-01-12)

Adds Confluence search with CQL query support.

### Features
- Add `cfl search` command with full-text search, space/type filters, and raw CQL support ([#36](https://github.com/open-cli-collective/confluence-cli/issues/36))

---

## v0.3.2 (2026-01-12)

Fixes markdown table conversion when creating pages.

### Changes
- Enable GFM table extension in markdown converter ([#30](https://github.com/open-cli-collective/confluence-cli/issues/30))

---

## v0.3.1 (2026-01-12)

Adds pagination metadata to JSON list output.

### Changes
- Add `_meta` field to JSON output from list commands with `count` and `hasMore` ([#31](https://github.com/open-cli-collective/confluence-cli/issues/31))

---

## v0.3.0 (2026-01-11)

Adds ability to find orphaned attachments.

### Features
- Add `--unused` flag to `attachment list` to filter for orphaned attachments ([#18](https://github.com/open-cli-collective/confluence-cli/issues/18))

---

## v0.2.5 (2026-01-11)

Fixes markdown conversion to preserve tables created in Confluence's web UI.

### Changes
- Preserve tables in HTML to markdown conversion ([#16](https://github.com/open-cli-collective/confluence-cli/issues/16))

---

## v0.2.4 (2026-01-11)

Fixes markdown conversion to preserve code blocks created in Confluence's web UI.

### Changes
- Preserve code blocks from Confluence UI pages in markdown output ([#15](https://github.com/open-cli-collective/confluence-cli/issues/15))

---

## v0.2.3 (2026-01-11)

Improves error messages when invalid page status values are provided.

### Changes
- Reject invalid `--status` values with helpful error message ([#17](https://github.com/open-cli-collective/confluence-cli/issues/17))

---

## v0.2.2 (2026-01-11)

Fixes `page copy` when the `--space` flag is omitted.

### Changes
- Resolve space key from spaceId for page copy ([#14](https://github.com/open-cli-collective/confluence-cli/issues/14))

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
