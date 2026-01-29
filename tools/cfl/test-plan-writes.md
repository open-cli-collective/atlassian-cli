# Write Operations Test Plan for cfl

## Summary

The documentation (readme.md, cfl-intro.md) makes numerous claims about write operations that have **never been tested against a real Confluence instance**. The previous chaos testing session only covered read operations.

This document inventories all untested assertions and provides comprehensive test cases.

---

## Untested Claims Inventory

### Page Creation (`cfl page create`)

| Claim | Source | Tested? |
|-------|--------|---------|
| Create pages from markdown files | readme.md:191 | NO |
| Create pages from stdin | readme.md:194 | NO |
| Create pages via $EDITOR | readme.md:187 | NO |
| Markdown auto-converted to Confluence storage format | readme.md:184 | NO |
| `.md` files converted, `.html` used as-is | readme.md:215-218 | NO |
| `--no-markdown` disables conversion | readme.md:200, 213 | NO |
| `--parent` creates child pages | readme.md:203 | NO |
| File extension determines format | cfl-intro.md:136 | NO |

### Attachments

| Claim | Source | Tested? |
|-------|--------|---------|
| Upload attachments | readme.md:259-266 | NO |
| Upload with `--comment` | readme.md:265 | NO |
| Download attachments | readme.md:276-288 | Partially (unit test only) |
| Download to custom path (`-O`) | readme.md:282 | NO |

### Page Deletion

| Claim | Source | Tested? |
|-------|--------|---------|
| Delete pages | readme.md:222-235 | NO |
| `--force` skips confirmation | readme.md:228 | NO |
| Confirmation prompt works correctly | readme.md:224 | NO |

---

## Known Issues Found in Code Review

### Page Create Bugs

1. **BUG: `--no-markdown` ignored for stdin** (create.go:205-213)
   - When piping content, `--no-markdown` flag has no effect
   - Stdin is always treated as markdown

2. **No validation of parent page ID** - Invalid IDs passed directly to API

3. **No file size limits** - Could OOM on large files

4. **Loose file extension detection** - Unknown extensions (.txt, .rst) default to markdown

### Attachment Bugs

1. **SECURITY: Path traversal in download** (download.go:71)
   - `attachment.Title` used as filename without sanitization
   - Malicious filename like `../../../etc/passwd` could write outside directory

2. **File overwrite without warning** - `os.Create()` silently truncates existing files

3. **No upload test coverage** - UploadAttachment has zero unit tests

4. **Partial file on download error** - If `io.Copy()` fails, incomplete file remains

---

## Test Plan: Real Confluence Instance

### Prerequisites

- Working `cfl init` configuration
- A test space (create one called `CFLTEST` or similar)
- Note the space key for tests
- Permissions to create/delete pages and attachments

### Test Execution Notes

- Record page IDs created for cleanup
- Run delete tests last
- Some tests are destructive - use test space only

---

## Phase 1: Page Creation Tests

### 1.1 Basic Creation

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 1.1.1 | Create from markdown file | `echo "# Test" > /tmp/test.md && cfl page create -s SPACE -t "MD File Test" -f /tmp/test.md` | Page created with H1 converted | HIGH |
| 1.1.2 | Create from HTML file | `echo "<h1>Test</h1>" > /tmp/test.html && cfl page create -s SPACE -t "HTML File Test" -f /tmp/test.html` | Page created with raw HTML | HIGH |
| 1.1.3 | Create from stdin | `echo "# Stdin Test" \| cfl page create -s SPACE -t "Stdin Test"` | Page created with converted markdown | HIGH |
| 1.1.4 | Verify markdown conversion | View created page in browser | H1 rendered correctly, not literal `#` | HIGH |

### 1.2 Format Detection

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 1.2.1 | .md extension detected | `cfl page create -s SPACE -t "Test" -f file.md` | Converts markdown | MEDIUM |
| 1.2.2 | .markdown extension | `cfl page create -s SPACE -t "Test" -f file.markdown` | Converts markdown | MEDIUM |
| 1.2.3 | .html extension | `cfl page create -s SPACE -t "Test" -f file.html` | Uses as-is | MEDIUM |
| 1.2.4 | .htm extension | `cfl page create -s SPACE -t "Test" -f file.htm` | Uses as-is | MEDIUM |
| 1.2.5 | .xhtml extension | `cfl page create -s SPACE -t "Test" -f file.xhtml` | Uses as-is | MEDIUM |
| 1.2.6 | .txt extension | `cfl page create -s SPACE -t "Test" -f file.txt` | Treats as markdown (questionable) | LOW |
| 1.2.7 | No extension | `cfl page create -s SPACE -t "Test" -f noext` | Treats as markdown | LOW |

### 1.3 --no-markdown Flag

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 1.3.1 | --no-markdown with .md file | `cfl page create -s SPACE -t "Test" -f test.md --no-markdown` | Raw markdown NOT converted | HIGH |
| 1.3.2 | --no-markdown with stdin | `echo "# Raw" \| cfl page create -s SPACE -t "Test" --no-markdown` | **BUG**: Will still convert | HIGH |
| 1.3.3 | Verify literal # appears | View page in Confluence | Should see literal `#` if working | HIGH |

### 1.4 Parent Page (Child Creation)

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 1.4.1 | Create child page | `cfl page create -s SPACE -t "Child" -p PARENT_ID -f test.md` | Page created under parent | HIGH |
| 1.4.2 | Verify hierarchy | View in Confluence page tree | Child indented under parent | HIGH |
| 1.4.3 | Invalid parent ID | `cfl page create -s SPACE -t "Test" -p 99999999 -f test.md` | Error message (what kind?) | MEDIUM |
| 1.4.4 | Parent in different space | `cfl page create -s SPACE -t "Test" -p OTHER_SPACE_PAGE_ID -f test.md` | Error or success? | MEDIUM |

### 1.5 Editor Integration

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 1.5.1 | Opens editor when no input | `cfl page create -s SPACE -t "Editor Test"` | Opens $EDITOR with template | HIGH |
| 1.5.2 | Template is markdown | Check temp file | Contains markdown example | MEDIUM |
| 1.5.3 | Save and exit creates page | Edit, save, exit editor | Page created | HIGH |
| 1.5.4 | Exit without saving | Exit editor immediately | "no content provided" error | MEDIUM |
| 1.5.5 | Empty content after edit | Delete template, save empty | Error message | MEDIUM |
| 1.5.6 | --editor flag forces editor | `echo "ignored" \| cfl page create -s SPACE -t "Test" --editor` | Opens editor despite stdin | LOW |

### 1.6 Markdown Conversion Quality

| # | Test Case | Content | Verify In Confluence | Priority |
|---|-----------|---------|---------------------|----------|
| 1.6.1 | Headers (h1-h6) | `# H1\n## H2\n### H3` | All levels render | HIGH |
| 1.6.2 | Bold/Italic | `**bold** *italic*` | Formatting correct | HIGH |
| 1.6.3 | Unordered lists | `- item1\n- item2` | Bullet list | HIGH |
| 1.6.4 | Ordered lists | `1. first\n2. second` | Numbered list | HIGH |
| 1.6.5 | Code blocks | ` ```python\ncode\n``` ` | Syntax highlighted | HIGH |
| 1.6.6 | Inline code | `` `code` `` | Monospace | MEDIUM |
| 1.6.7 | Links | `[text](url)` | Clickable link | HIGH |
| 1.6.8 | Images | `![alt](url)` | Image displayed | MEDIUM |
| 1.6.9 | Blockquotes | `> quote` | Blockquote styled | MEDIUM |
| 1.6.10 | Tables | GFM table syntax | Table rendered | MEDIUM |
| 1.6.11 | Horizontal rules | `---` | Horizontal line | LOW |
| 1.6.12 | Nested lists | `- item\n  - nested` | Proper nesting | MEDIUM |

### 1.7 Edge Cases & Error Handling

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 1.7.1 | Missing --title | `cfl page create -s SPACE -f test.md` | Error: title required | HIGH |
| 1.7.2 | Missing --space (no default) | `cfl page create -t "Test" -f test.md` | Error: space required | HIGH |
| 1.7.3 | Invalid space key | `cfl page create -s NOTEXIST -t "Test" -f test.md` | Error: space not found | HIGH |
| 1.7.4 | File not found | `cfl page create -s SPACE -t "Test" -f /nonexistent` | Error: file not found | HIGH |
| 1.7.5 | Duplicate title | Create two pages with same title | Error or success? | MEDIUM |
| 1.7.6 | Empty file | `touch /tmp/empty.md && cfl page create -s SPACE -t "Test" -f /tmp/empty.md` | Error or empty page? | MEDIUM |
| 1.7.7 | Binary file | `cfl page create -s SPACE -t "Test" -f /bin/ls` | Error handling? | LOW |
| 1.7.8 | Very large file | Create 10MB markdown file | Timeout? OOM? | LOW |
| 1.7.9 | Special chars in title | `cfl page create -s SPACE -t "Test <>&\"'" -f test.md` | Proper escaping | MEDIUM |
| 1.7.10 | Unicode in content | File with emoji, CJK chars | Preserved correctly | MEDIUM |

### 1.8 Output Formats

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 1.8.1 | Default output | `cfl page create -s SPACE -t "Test" -f test.md` | Success message with ID, URL | HIGH |
| 1.8.2 | JSON output | `cfl page create -s SPACE -t "Test" -f test.md -o json` | Valid JSON with page data | MEDIUM |
| 1.8.3 | Plain output | `cfl page create -s SPACE -t "Test" -f test.md -o plain` | Tab-separated | LOW |

---

## Phase 2: Attachment Tests

### 2.1 Upload

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 2.1.1 | Basic upload | `cfl attachment upload -p PAGE_ID -f image.png` | Attachment created | HIGH |
| 2.1.2 | Upload with comment | `cfl attachment upload -p PAGE_ID -f doc.pdf -c "Version 1"` | Comment visible in Confluence | HIGH |
| 2.1.3 | Verify in Confluence | View page attachments | File listed with correct name/size | HIGH |
| 2.1.4 | Upload to invalid page | `cfl attachment upload -p 99999999 -f image.png` | Error: page not found | MEDIUM |
| 2.1.5 | Upload nonexistent file | `cfl attachment upload -p PAGE_ID -f /nonexistent` | Error: file not found | HIGH |
| 2.1.6 | Upload empty file | `touch /tmp/empty && cfl attachment upload -p PAGE_ID -f /tmp/empty` | Success or error? | MEDIUM |
| 2.1.7 | Large file upload | Upload 50MB file | Success with timeout? | LOW |
| 2.1.8 | Special chars in filename | File with spaces, unicode | Preserved correctly | MEDIUM |
| 2.1.9 | Replace existing attachment | Upload same filename twice | Creates new version? | MEDIUM |

### 2.2 Download

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 2.2.1 | Basic download | `cfl attachment download ATT_ID` | File saved with original name | HIGH |
| 2.2.2 | Download with -O | `cfl attachment download ATT_ID -O custom.png` | File saved as custom.png | HIGH |
| 2.2.3 | Verify content | Compare downloaded to original | Byte-for-byte match | HIGH |
| 2.2.4 | Invalid attachment ID | `cfl attachment download INVALID` | Error: not found | MEDIUM |
| 2.2.5 | Download to existing file | Download when file exists | **Silently overwrites** (document!) | HIGH |
| 2.2.6 | Path in -O | `cfl attachment download ATT_ID -O /tmp/subdir/file.png` | Works if dir exists? | MEDIUM |
| 2.2.7 | **SECURITY: Path traversal** | If attachment named `../../../tmp/evil` | Should sanitize filename | CRITICAL |

### 2.3 List

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 2.3.1 | List page with attachments | `cfl attachment list -p PAGE_ID` | Table with ID, name, type, size | HIGH |
| 2.3.2 | List page without attachments | `cfl attachment list -p PAGE_NO_ATT` | "No attachments" message | MEDIUM |
| 2.3.3 | JSON output | `cfl attachment list -p PAGE_ID -o json` | Valid JSON array | MEDIUM |

---

## Phase 3: Page Delete Tests

### 3.1 Basic Deletion

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 3.1.1 | Delete with confirmation | `cfl page delete PAGE_ID` then `y` | Page deleted | HIGH |
| 3.1.2 | Cancel deletion | `cfl page delete PAGE_ID` then `n` | "Deletion cancelled" | HIGH |
| 3.1.3 | Delete with --force | `cfl page delete PAGE_ID --force` | Deleted without prompt | HIGH |
| 3.1.4 | Delete with -f shorthand | `cfl page delete PAGE_ID -f` | Deleted without prompt | MEDIUM |
| 3.1.5 | Verify deletion | `cfl page view PAGE_ID` after delete | 404 error | HIGH |

### 3.2 Error Cases

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 3.2.1 | Invalid page ID | `cfl page delete 99999999` | Error: page not found | HIGH |
| 3.2.2 | Already deleted | Delete same page twice | Error on second attempt | MEDIUM |
| 3.2.3 | Missing page ID | `cfl page delete` | Error: requires page ID | HIGH |
| 3.2.4 | Non-numeric ID | `cfl page delete abc` | Error message | MEDIUM |
| 3.2.5 | Delete page with children | Delete parent with child pages | Children orphaned? Blocked? | MEDIUM |

### 3.3 Confirmation Prompt Details

| # | Test Case | Input | Expected | Priority |
|---|-----------|-------|----------|----------|
| 3.3.1 | Lowercase y | `y` | Deletes | HIGH |
| 3.3.2 | Uppercase Y | `Y` | Deletes | HIGH |
| 3.3.3 | "yes" | `yes` | Cancels (only y/Y accepted) | MEDIUM |
| 3.3.4 | Empty input (Enter) | `` | Cancels | MEDIUM |
| 3.3.5 | Other input | `maybe` | Cancels | LOW |

---

## Phase 4: Integration Tests

### 4.1 Create-View-Delete Cycle

| # | Test Case | Steps | Expected | Priority |
|---|-----------|-------|----------|----------|
| 4.1.1 | Full lifecycle | 1. Create page 2. View it 3. Delete it | All steps succeed | HIGH |
| 4.1.2 | Create with attachment | 1. Create page 2. Upload attachment 3. List attachments | Attachment visible | HIGH |
| 4.1.3 | Create child, delete parent | 1. Create parent 2. Create child 3. Delete parent | Child behavior? | MEDIUM |

### 4.2 Piping and Scripting

| # | Test Case | Command | Expected | Priority |
|---|-----------|---------|----------|----------|
| 4.2.1 | Pipe file content | `cat notes.md \| cfl page create -s SPACE -t "Test"` | Page created | HIGH |
| 4.2.2 | Capture created ID | `cfl page create -s SPACE -t "Test" -f x.md -o json \| jq -r .id` | Extracts page ID | HIGH |
| 4.2.3 | Script: create many pages | Loop creating 10 pages | All succeed | MEDIUM |

---

## Cleanup Checklist

After testing, delete all test artifacts:
- [ ] All pages created in test space
- [ ] Test space itself (if created for testing)
- [ ] Local temp files (/tmp/test.md, etc.)

---

## Summary of Critical Tests

**Must pass before claiming write operations work:**

1. `cfl page create -f file.md` creates page with converted markdown
2. `cfl page create` with stdin works
3. `--parent` creates proper page hierarchy
4. `cfl attachment upload` works with comment
5. `cfl attachment download` preserves file content
6. `cfl page delete` with confirmation works
7. `cfl page delete --force` bypasses prompt

**Known issues to document (not fix):**

1. `--no-markdown` has no effect on stdin input
2. Download overwrites existing files without warning
3. Path traversal possible in download filename (security issue)

---

## Test Progress Tracking

### Page Create
| Test | Result | Notes |
|------|--------|-------|
| 1.1.1 | | |
| 1.1.2 | | |
| 1.1.3 | | |
| 1.1.4 | | |

### Attachments
| Test | Result | Notes |
|------|--------|-------|
| 2.1.1 | | |
| 2.1.2 | | |
| 2.2.1 | | |
| 2.2.2 | | |

### Page Delete
| Test | Result | Notes |
|------|--------|-------|
| 3.1.1 | | |
| 3.1.2 | | |
| 3.1.3 | | |
