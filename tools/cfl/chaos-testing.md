# Chaos Testing Session

**Started:** 2026-01-03
**Focus:** Read-only operations (list, view, download)

---

## Bugs Found

| ID | Severity | Description | Repro Steps | Status |
|----|----------|-------------|-------------|--------|
| BUG-001 | Medium | `--limit 0` silently uses default (25) instead of returning 0 or error | `cfl space list --limit 0` | **FIXED** |
| BUG-002 | Medium | `--limit -1` silently uses default (25) instead of error | `cfl space list --limit -1` | **FIXED** |
| BUG-003 | High | JSON output for `space list` includes trailing message, breaking JSON parsing | `cfl space list -o json \| jq .` fails | **FIXED** |
| BUG-004 | Low | `-o yaml` silently falls back to table format instead of erroring | `cfl space list -o yaml` | **FIXED** |
| BUG-005 | Low | Confluence macro parameters leak into markdown output | View page with TOC macro, see "12" artifact | **FIXED** |
| BUG-006 | **CRITICAL** | `attachment download` panics - flag shorthand conflict | `cfl attachment download <id>` crashes with `-o` conflict | **FIXED** |

---

## Design Decisions

Decisions made during testing:

1. **What should `--limit 0` do?**
   - **DECISION:** Return empty list (0 results)

2. **What should invalid output format do?**
   - **DECISION:** Error with message listing valid formats

3. **Confluence macros in markdown output?**
   - **DECISION:** Default = strip entirely (clean output). Add `--show-macros` flag to opt-in to placeholders like `[TOC]`, `[Info box: ...]`

4. **Pagination message "(showing first X results...)"**
   - **DECISION:** Send to stderr (keeps stdout clean for piping)

---

## Behavior Questions

Things where the current behavior might be correct, but we should confirm:

1. `--type GLOBAL` works (case insensitive) - API handles this, probably fine
2. Trailing "(showing first X results...)" message - should this be stderr instead of stdout?

---

## Test Progress

### Space Listing (`cfl space list`)
| Test | Result | Notes |
|------|--------|-------|
| Basic list | PASS | Shows 25 spaces with pagination message |
| `--limit 0` | BUG | Returns 25 (default) instead of 0 or error |
| `--limit -1` | BUG | Returns 25 (default) instead of error |
| `--limit 1` | PASS | Returns 1 result |
| `--limit abc` | PASS | Good error message |
| `--type global` | PASS | Filters correctly |
| `--type personal` | PASS | Filters correctly |
| `--type GLOBAL` | PASS | Case insensitive (API handles) |
| `--type invalid` | PASS | Good error from API |
| `-o json` | BUG | Invalid JSON due to trailing message |
| `-o plain` | PASS | Tab-separated, but has trailing message |
| `-o yaml` | BUG | Silently falls back to table |

### Page Listing (`cfl page list`)
| Test | Result | Notes |
|------|--------|-------|
| Basic list with space | PASS | Works correctly |
| Invalid space | PASS | Good error message |
| No space, no default | PASS | Helpful error message |

### Page Viewing (`cfl page view`)
| Test | Result | Notes |
|------|--------|-------|
| Valid page ID | PASS | Markdown conversion works |
| Invalid page ID (999999999) | PASS | Good 404 error |
| Non-numeric ID (abc) | PASS | Good 400 error |
| Missing page ID | PASS | "accepts 1 arg(s)" error |
| `--raw` mode | PASS | Shows Confluence storage format |
| `-o json` | PASS | Valid JSON (no trailing message!) |
| Confluence macros | BUG-005 | TOC macro params leak through |

### Attachments
| Test | Result | Notes |
|------|--------|-------|
| List attachments (page with attachments) | PASS | Shows table with ID, title, type, size |
| List attachments (page without) | PASS | "No attachments found." |
| Download attachment | CRASH | BUG-006: Panic due to `-o` flag conflict |

---

## Edge Cases Discovered

1. **Confluence macros in content**: `ac:structured-macro`, `ac:link`, `ri:page` elements don't convert to markdown. TOC macro's `minLevel`/`maxLevel` params appear as orphan numbers ("12").

2. **HTML entities**: `&rsquo;` (right single quote), `&ndash;` (en dash) appear in storage format, need to verify markdown handles them.

---

## Summary

**6 bugs found (ALL FIXED):**
- 1 CRITICAL (panic crash) - FIXED: Changed `-o` shorthand to `-O` for `--output-file`
- 1 HIGH (broken JSON output) - FIXED: Pagination message now goes to stderr
- 2 MEDIUM (silent limit issues) - FIXED: Limit validation (negative = error, 0 = empty list)
- 2 LOW (minor UX issues) - FIXED: Output format validation + macro stripping

**Tests passed:** ~25
**Tests failed:** 0 (all bugs fixed)

---

## Next Steps

- [x] Test attachment list/download - DONE (found crash bug)
- [ ] Test `--web` flag
- [ ] Find page with code blocks to test syntax highlighting conversion
- [x] Test aliases (ls vs list) - DONE (works)

