# Integration Tests

This document catalogs the manual integration test suite for `cfl`. These tests verify real-world behavior against a live Confluence instance and catch edge cases that are difficult to cover with unit tests.

## Test Environment Setup

### Prerequisites
- A configured `cfl` instance (`cfl init` completed)
- Access to a test space (e.g., `confluence`)
- Permission to create, edit, and delete pages/attachments

### Test Data Conventions
- Test pages use `[Test]` prefix: `[Test] My Page`
- Baseline pages (for comparison) use `[Baseline]` prefix
- Always clean up test data after tests complete

---

## Page Operations

### page list

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List pages in space | `cfl page list --space confluence` | Shows table of pages with ID, title, status, version |
| List with limit | `cfl page list --space confluence --limit 5` | Shows only 5 pages with "showing first N results" message |
| JSON output | `cfl page list --space confluence --output json` | Valid JSON array |
| Plain output | `cfl page list --space confluence --output plain` | Tab-separated values |
| List trashed pages | `cfl page list --space confluence --status trashed` | Shows deleted pages |
| List archived pages | `cfl page list --space confluence --status archived` | Shows archived pages |
| Invalid status (draft) | `cfl page list --space confluence --status draft` | Error: API rejects draft status |

### page view

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| View page content | `cfl page view <page-id>` | Shows title, ID, version, and markdown content |
| View raw HTML | `cfl page view <page-id> --raw` | Shows Confluence storage format (XHTML) |
| JSON output | `cfl page view <page-id> --output json` | Full page object as JSON |
| Non-existent page | `cfl page view 99999999999` | Error: 404 not found |

### page create

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Create from stdin | `echo "# Test" \| cfl page create -s confluence -t "Test Page"` | Page created, shows ID and URL |
| Create from file | `cfl page create -s confluence -t "Test" --file content.md` | Page created from file content |
| Create child page | `cfl page create -s confluence -t "Child" --parent <id>` | Page created with parentId set |
| Create with XHTML | `echo "<p>Test</p>" \| cfl page create -s confluence -t "Test" --no-markdown` | Page created without markdown conversion |
| Missing title | `cfl page create -s confluence` | Error: title required |
| Missing space | `cfl page create -t "Test"` | Error: space required |
| Duplicate title | Create same title twice | Error: "page already exists with same TITLE" |
| Very long title (300+ chars) | Create with long title | Error: API rejects (400) |
| Empty content | `echo "" \| cfl page create -s confluence -t "Empty"` | Page created (empty content allowed) |

### page edit

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Edit from file | `cfl page edit <id> --file updated.md` | Page updated, version incremented |
| Edit with --no-markdown | `cfl page edit <id> --file content.html --no-markdown` | Raw XHTML preserved |
| Edit page with tables (markdown mode) | Edit without --no-markdown | **WARNING: Tables destroyed** (known bug #16) |
| Edit page with code blocks (UI-created) | Edit without --no-markdown | **WARNING: Code blocks stripped** (known bug #15) |
| Non-existent page | `cfl page edit 99999999999` | Error: 404 not found |

### page copy

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Copy with --space | `cfl page copy <id> --title "Copy" --space confluence` | Page copied to same space |
| Copy without --space | `cfl page copy <id> --title "Copy"` | Page copied (space inferred from source) |
| Copy to different space | `cfl page copy <id> --title "Copy" --space OTHER` | Page copied to different space |
| Copy without attachments | `cfl page copy <id> --title "Copy" --no-attachments` | Page copied, attachments excluded |
| Copy without labels | `cfl page copy <id> --title "Copy" --no-labels` | Page copied, labels excluded |
| Duplicate title in space | Copy to existing title | Error: duplicate title |
| Non-existent source | `cfl page copy 99999 --title "Copy" --space confluence` | Error: 404 not found |

### page delete

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Delete with confirmation | `cfl page delete <id>` (type "y") | Page deleted after confirmation |
| Delete cancelled | `cfl page delete <id>` (type "n") | "Deletion cancelled" message |
| Delete with --force | `cfl page delete <id> --force` | Page deleted without confirmation |
| Non-existent page | `cfl page delete 99999999999 --force` | Error: 404 not found |

---

## Attachment Operations

### attachment list

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List attachments | `cfl attachment list --page <id>` | Table of attachments with ID, title, type, size |
| No attachments | List on page with none | "No attachments found" |
| JSON output | `cfl attachment list --page <id> --output json` | Valid JSON array with full attachment metadata |

### attachment upload

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Upload text file | `cfl attachment upload --page <id> --file test.txt` | Attachment created, shows ID |
| Upload with comment | `cfl attachment upload --page <id> --file test.txt --comment "Description"` | Attachment with comment |
| Upload binary file | `cfl attachment upload --page <id> --file image.png` | Binary file uploaded correctly |
| Unicode filename | `cfl attachment upload --page <id> --file "tëst-filé.txt"` | Special characters handled |
| Filename with spaces | `cfl attachment upload --page <id> --file "my file (1).txt"` | Spaces and parens handled |
| Non-existent page | `cfl attachment upload --page 99999 --file test.txt` | Error: page not found |

### attachment download

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Download to current dir | `cfl attachment download <att-id>` | File saved with original filename |
| Download to specific path | `cfl attachment download <att-id> -O /tmp/output.txt` | File saved to specified path |
| Verify content integrity | Upload then download, compare | Files match exactly |
| Non-existent attachment | `cfl attachment download att99999` | Error: attachment not found |
| File already exists | Download to existing file | Error: "file exists (use --force)" |
| Overwrite with --force | `cfl attachment download <id> -O existing.txt --force` | File overwritten |

### attachment delete

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Delete with confirmation | `cfl attachment delete <id>` (type "y") | Attachment deleted |
| Delete cancelled | `cfl attachment delete <id>` (type "n") | "Deletion cancelled" |
| Delete with --force | `cfl attachment delete <id> --force` | Deleted without confirmation |
| Non-existent attachment | `cfl attachment delete att99999 --force` | Error: 404 not found |

---

## Space Operations

### space list

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List all spaces | `cfl space list` | Table of spaces with key, name, type |
| JSON output | `cfl space list --output json` | Valid JSON array |
| Limit results | `cfl space list --limit 5` | Shows first 5 spaces |

---

## Content Fidelity Tests

These tests verify that content survives round-trip conversions.

### Markdown Round-Trip (CLI-created pages)

Pages created via `cfl` use standard HTML and round-trip correctly:

| Content Type | Create | View | Edit | Result |
|--------------|--------|------|------|--------|
| Headers (h1-h6) | Pass | Pass | Pass | Preserved |
| Bold/italic | Pass | Pass | Pass | Preserved |
| Bullet lists | Pass | Pass | Pass | Preserved |
| Numbered lists | Pass | Pass | Pass | Preserved |
| Code blocks (fenced) | Pass | Pass | Pass | Preserved |
| Inline code | Pass | Pass | Pass | Preserved |
| Links | Pass | Pass | Pass | Preserved |
| Blockquotes | Pass | Pass | Pass | Preserved |

### Confluence UI-Created Content

Pages created in Confluence's web UI use proprietary macros that may not round-trip:

| Content Type | View | Edit | Result |
|--------------|------|------|--------|
| Tables | Flattened to text | **DESTROYED** | Use --no-markdown |
| Code blocks (macro) | **STRIPPED** | **STRIPPED** | Use --no-markdown |
| Info/warning panels | Stripped | Stripped | Use --no-markdown |
| Expand macros | Stripped | Stripped | Use --no-markdown |
| TOC macros | Stripped | Stripped | Use --no-markdown |

**Workaround**: Always use `--no-markdown` when editing pages with complex Confluence content.

---

## Edge Cases & Error Handling

### Unicode & Special Characters

| Test Case | Expected Result |
|-----------|-----------------|
| Unicode in page title | `[Test] Spëcial Chàracters 中文` works |
| Unicode in page content | Emojis, CJK characters preserved |
| Unicode in attachment filename | Handled correctly |
| Special chars: `& < > "` | Properly escaped |

### Error Messages

| Scenario | Expected Error |
|----------|----------------|
| Invalid page ID | "API error (status 404): Page not found" |
| Invalid space key | "API error (status 404): Space not found" |
| Permission denied | "API error (status 403): ..." |
| Network timeout | "context deadline exceeded" or similar |
| Invalid credentials | "API error (status 401): ..." |

### Output Formats

| Format | Flag | Verified With |
|--------|------|---------------|
| Table (default) | (none) | Visual inspection |
| JSON | `--output json` | `jq .` parsing |
| Plain | `--output plain` | Tab-separated, scriptable |

---

## Test Execution Checklist

Before GA release, run through this checklist:

### Setup
- [ ] Build latest: `make build`
- [ ] Verify config: `cfl space list` works

### Page CRUD
- [ ] Create page from stdin
- [ ] Create page from file
- [ ] Create child page
- [ ] View page (markdown)
- [ ] View page (raw)
- [ ] Edit page from file
- [ ] Copy page (same space)
- [ ] Copy page (different space)
- [ ] Delete page (with confirmation)
- [ ] Delete page (--force)

### Attachment CRUD
- [ ] Upload attachment
- [ ] List attachments
- [ ] Download attachment
- [ ] Verify downloaded content matches
- [ ] Delete attachment

### Edge Cases
- [ ] Unicode in titles/content
- [ ] Empty content
- [ ] Very long title (expect rejection)
- [ ] Duplicate title (expect rejection)
- [ ] Non-existent resources (expect 404)

### Cleanup
- [ ] Delete all [Test] prefixed pages
- [ ] Verify no test data remains

---

## Known Issues

| Issue | Severity | Workaround | GitHub Issue |
|-------|----------|------------|--------------|
| Tables destroyed on markdown edit | P2 | Use `--no-markdown` | #16 |
| Code blocks stripped (UI pages) | P2 | Use `--no-markdown` | #15 |
| `--status draft` invalid | P3 | Don't use draft status | #17 |

---

## Adding New Tests

When adding new features or fixing bugs:

1. Add test cases to the appropriate section above
2. Include both happy path and error cases
3. Document any known limitations or edge cases
4. Update the "Test Execution Checklist" if needed
