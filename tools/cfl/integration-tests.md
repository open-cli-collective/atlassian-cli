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

## Init

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Fresh init | `cfl init` (interactive) | Creates ~/.config/cfl/config.yml with URL, email, token |
| Init with existing config | `cfl init` when config exists | Prompts to overwrite or skip |
| Verify connection | After init, run `cfl space list` | Connection works, spaces listed |
| Invalid credentials | Init with bad API token | Error during verification step |
| Invalid URL | Init with malformed URL | Error: invalid URL format |

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
| View content only | `cfl page view <id> --content-only` | Markdown only, no Title/ID/Version headers |
| Content only with raw | `cfl page view <id> --content-only --raw` | XHTML only, no headers |
| Content only with macros | `cfl page view <id> --content-only --show-macros` | Markdown with [TOC] etc., no headers |
| Roundtrip macros (content-only) | `cfl page view <id> --show-macros --content-only \| cfl page edit <id> --legacy` | Macros preserved |
| Content only JSON error | `cfl page view <id> --content-only -o json` | Error: incompatible flags |
| Content only web error | `cfl page view <id> --content-only --web` | Error: incompatible flags |

### page create

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Create from stdin | `echo "# Test" \| cfl page create -s confluence -t "Test Page"` | Page created, shows ID and URL |
| Create from file | `cfl page create -s confluence -t "Test" --file content.md` | Page created from file content |
| Create child page | `cfl page create -s confluence -t "Child" --parent <id>` | Page created with parentId set |
| Create with XHTML (legacy) | `echo "<p>Test</p>" \| cfl page create -s confluence -t "Test" --no-markdown --legacy` | Page created without markdown conversion |
| Missing title | `cfl page create -s confluence` | Error: title required |
| Missing space | `cfl page create -t "Test"` | Error: space required |
| Duplicate title | Create same title twice | Error: "page already exists with same TITLE" |
| Very long title (300+ chars) | Create with long title | Error: API rejects (400) |
| Empty content | `echo "" \| cfl page create -s confluence -t "Empty"` | Error: "page content cannot be empty" |
| Whitespace-only content | `echo "   " \| cfl page create -s confluence -t "Whitespace"` | Error: "page content cannot be empty" |
| Create (cloud editor) | `echo "# Test" \| cfl page create -s confluence -t "Test"` | Page uses cloud editor (see verification below) |
| Create (legacy editor) | `echo "# Test" \| cfl page create -s confluence -t "Test" --legacy` | Page uses legacy editor |
| Create with code block (cloud) | Create page with fenced code block | Code block preserved as `codeBlock` in ADF |

### page edit

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Edit from file | `cfl page edit <id> --file updated.md` | Page updated, version incremented |
| Edit with --no-markdown (legacy) | `cfl page edit <id> --file content.html --no-markdown --legacy` | Raw XHTML preserved |
| Edit page with tables (markdown mode) | Edit without --no-markdown | See "Confluence UI-Created Content" section |
| Edit page with code blocks (UI-created) | Edit without --no-markdown | See "Confluence UI-Created Content" section |
| Non-existent page | `cfl page edit 99999999999` | Error: 404 not found |
| Edit (cloud editor) | `cfl page edit <id> --file updated.md` | Page stays in cloud editor format |
| Edit (legacy editor) | `cfl page edit <id> --file updated.md --legacy` | Page uses legacy storage format |
| Move to new parent | `cfl page edit <id> --parent <parent-id>` | Page appears under new parent in tree |
| Move and rename | `cfl page edit <id> --parent <parent-id> --title "New Title"` | Page moved AND renamed |
| Move with content update | `cfl page edit <id> --parent <parent-id> --file updated.md` | Page moved with new content |
| Move to invalid parent | `cfl page edit <id> --parent 99999999999` | Error: 404 not found |
| Move preserves history | Move page, then check version history | Previous versions still visible in UI |
| Move page (no content change) | `cfl page edit <id> --parent <parent-id>` | Page moved without opening editor, content unchanged |
| Move and rename (no content change) | `cfl page edit <id> --parent <parent-id> --title "New Title"` | Page moved and renamed without editor |
| Empty content from stdin | `echo "" \| cfl page edit <id>` | Error: "page content cannot be empty" |
| Whitespace-only from stdin | `echo "   " \| cfl page edit <id>` | Error: "page content cannot be empty" |

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
| List unused attachments | `cfl attachment list --page <id> --unused` | Only attachments not referenced in page content |
| No unused attachments | `--unused` on page using all attachments | "No unused attachments found" |

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

## Search Operations

### search

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Full-text search | `cfl search "test"` | Shows matching content with ID, TYPE, SPACE, TITLE |
| Search in space | `cfl search "content" --space DEV` | Only results from specified space |
| Filter by type | `cfl search --type page` | Only pages returned |
| Search by title | `cfl search --title "Test"` | Content with "Test" in title |
| Search by label | `cfl search --label test-label` | Content with specified label |
| Combined filters | `cfl search "deploy" --space DEV --type page` | Filtered results |
| Raw CQL | `cfl search --cql "type=page AND space=DEV"` | CQL executed directly |
| JSON output | `cfl search "test" -o json` | Valid JSON with results and _meta |
| Plain output | `cfl search "test" -o plain` | Tab-separated values |
| Limit results | `cfl search "test" --limit 5` | Max 5 results |
| No results | `cfl search "xyznonexistent123"` | "No results found" message |
| Invalid type | `cfl search --type invalid` | Error: invalid type |

### Search After Create (End-to-End)

| Test Case | Steps | Expected Result |
|-----------|-------|-----------------|
| Search finds new page | 1. `echo "# Test" \| cfl page create -s DEV -t "[Test] Searchable"`<br>2. Wait 5-10s for indexing<br>3. `cfl search "[Test] Searchable"` | New page appears in results |
| Content search | 1. Create page with unique content "xyzUniqueContent789"<br>2. Wait 5-10s<br>3. `cfl search "xyzUniqueContent789"` | Page found by body content |

**Note:** Confluence search indexing has a delay (typically 5-10 seconds). Integration tests should wait before searching for newly created content.

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
| Tables | Pass | Pass | Preserved (fixed in #25) |
| Code blocks (macro) | Pass | Pass | Preserved (fixed in #24) |
| Info/warning panels | Pass* | Pass* | Preserved with `--show-macros` (fixed in #51) |
| Expand macros | Pass* | Pass* | Preserved with `--show-macros` (fixed in #51) |
| TOC macros | Pass* | Pass* | Preserved with `--show-macros` (fixed in #51) |

**Note**: Tables and code blocks work automatically. For macro-heavy pages, use `--show-macros` when viewing to preserve macros as `[TOC]`, `[INFO]...[/INFO]`, etc. during roundtrip editing.

### Macro Roundtrip (Issue #51)

Tests for `--show-macros` roundtrip support. **Fully implemented: TOC, panels, expand, nested macros.**

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| View TOC with params | `cfl page view <toc-page> --show-macros` | Shows `[TOC maxLevel=3]` with parameters |
| Roundtrip TOC | `cfl page view <id> --show-macros \| cfl page edit <id>` | TOC macro preserved in page |
| Create with TOC | `echo "[TOC]\n# H1\n## H2" \| cfl page create -s SPACE -t "TOC Test"` | Page has working TOC |
| View info panel | `cfl page view <panel-page> --show-macros` | Shows `[INFO]...[/INFO]` |
| Create with panel | `echo "[WARNING]Be careful[/WARNING]" \| cfl page create ...` | Warning panel in page |
| Roundtrip panel | Pipe view to edit | Panel preserved with content |
| View expand | `cfl page view <expand-page> --show-macros` | Shows `[EXPAND]...[/EXPAND]` |
| Create with expand | Create page with expand syntax | Expand works in Confluence |
| Nested macros (create) | `echo "[INFO]\n[TOC]\n[/INFO]\n# H1" \| cfl page create ...` | Both INFO and TOC macros in page |
| Nested macros (view) | `cfl page view <nested-page> --show-macros` | Shows `[INFO]...[TOC]...[/INFO]` |
| Nested macros (roundtrip) | View nested page, pipe to edit | Both macros preserved with correct params |

**Syntax Reference:**
- TOC: `[TOC]` or `[TOC maxLevel=3 minLevel=1]`
- Panels: `[INFO]content[/INFO]`, `[WARNING]`, `[NOTE]`, `[TIP]`
- Expand: `[EXPAND title="Click me"]content[/EXPAND]`
- Nested: `[INFO][TOC maxLevel=2][/INFO]` (macros can be nested)

---

## Cloud Editor vs Legacy Editor

Pages created via `cfl` now use the **cloud editor** format (ADF) by default. Use `--legacy` to create pages in the legacy editor format (storage/XHTML).

### Verifying Editor Format

**Visual verification:**
- Open the page in Confluence web UI
- Legacy pages show a "Legacy editor" badge in the toolbar
- Cloud pages have no badge (or show the modern editor)

**API verification:**
```bash
# Check editor property via v1 API
curl -s -u "$EMAIL:$TOKEN" "$URL/rest/api/content/<page-id>?expand=metadata.properties.editor"

# Cloud editor: editor property is null/absent
# Legacy editor: editor.value = "v1"
```

**ADF structure verification:**
```bash
# Read page as ADF format
curl -s -u "$EMAIL:$TOKEN" "$URL/api/v2/pages/<page-id>?body-format=atlas_doc_format" | jq '.body.atlas_doc_format.value'

# Cloud pages have proper ADF structure:
# {"type":"doc","version":1,"content":[...]}

# Check code blocks are proper codeBlock nodes (not paragraphs with code marks)
# Proper: {"type":"codeBlock","attrs":{"language":"go"},"content":[...]}
# Wrong: {"type":"paragraph","content":[{"type":"text","marks":[{"type":"code"}],...}]}
```

### Cloud Editor Test Matrix

| Test ID | Input | Flags | Expected Format | Verification |
|---------|-------|-------|-----------------|--------------|
| CE-01 | stdin | (none) | ADF | `body.atlas_doc_format` present |
| CE-02 | stdin | --legacy | storage | `body.storage` present |
| CE-03 | file.md | (none) | ADF | No "Legacy editor" badge |
| CE-04 | file.md | --legacy | storage | Shows "Legacy editor" badge |
| CE-05 | file.html | --legacy | storage | Raw HTML passed through |
| CE-06 | stdin | --no-markdown | ADF | Raw content passed through |
| CE-07 | stdin | --no-markdown --legacy | storage | Raw XHTML passed through |

### Round-Trip Tests

| Test ID | Create Format | Edit Format | Expected Result | Notes |
|---------|---------------|-------------|-----------------|-------|
| RT-01 | ADF (default) | ADF (default) | ADF preserved | Happy path |
| RT-02 | --legacy | --legacy | Storage preserved | Legacy happy path |
| RT-03 | ADF (default) | --legacy | Warning shown, storage used | May switch editor |
| RT-04 | --legacy | ADF (default) | ADF used | Page stays legacy until manually converted |

### Test Cases

| Test Case | Steps | Expected Result |
|-----------|-------|-----------------|
| Create page (default) | 1. `echo "# Test" \| cfl page create -s confluence -t "[Test] Cloud"`<br>2. Open in browser | No "Legacy editor" badge |
| Create page (--legacy) | 1. `echo "# Test" \| cfl page create -s confluence -t "[Test] Legacy" --legacy`<br>2. Open in browser | Shows "Legacy editor" badge |
| Code block preservation | 1. Create page with fenced code block<br>2. Read as ADF via API | Has `codeBlock` node with language attr |
| Edit maintains format | 1. Create cloud page<br>2. `cfl page edit <id> --file updated.md`<br>3. View in browser | Still cloud editor |
| Edit with --legacy warning | 1. Create cloud page<br>2. `cfl page edit <id> --file updated.md --legacy` | Warning message shown |
| Complex markdown (ADF) | 1. Create page with tables, code blocks, nested lists<br>2. Read as ADF | All elements preserved as proper ADF nodes |

### Known Behavior

- **Default (cloud editor)**: Markdown converted to ADF JSON, code blocks properly preserved
- **--legacy flag**: Markdown converted to XHTML storage format, warning shown on edit
- **Storage→ADF conversion**: Confluence's built-in conversion loses code block structure (converts to paragraph with code mark)
- **Recommendation**: Use default (cloud editor) for new pages, use `--legacy` only for compatibility with existing legacy pages

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
- [ ] Run `cfl init` (verify config creation works)

### Page CRUD
- [ ] Create page from stdin (cloud editor)
- [ ] Create page with code block (verify ADF codeBlock)
- [ ] Create page from file
- [ ] Create page with --legacy flag
- [ ] Create child page
- [ ] View page (markdown)
- [ ] View page (raw)
- [ ] View page (content-only)
- [ ] View page (content-only with --show-macros for roundtrip)
- [ ] Roundtrip macro page via pipe (`view --show-macros --content-only | edit --legacy`)
- [ ] Edit page from file
- [ ] Edit page with --legacy flag
- [ ] Move page to new parent (`--parent` flag)
- [ ] Move page (no content change, no editor opened)
- [ ] Move and rename page together
- [ ] Move and rename (no content change, no editor opened)
- [ ] Verify page history preserved after move
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

### Search
- [ ] Full-text search returns results
- [ ] Space filter works
- [ ] Type filter works
- [ ] JSON output is valid
- [ ] Raw CQL works

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

## Adding New Tests

When adding new features or fixing bugs:

1. Add test cases to the appropriate section above
2. Include both happy path and error cases
3. Document any known limitations or edge cases
4. Update the "Test Execution Checklist" if needed
