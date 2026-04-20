# ManagePage Workflow

Create, edit, copy, move, and delete Confluence pages.

## Intent-to-Flag Mapping

### Action Selection

| User Says | Action | Command |
|-----------|--------|---------|
| "create page", "new page", "add page" | Create | `cfl page create` |
| "edit page", "update page", "change content" | Edit | `cfl page edit PAGE_ID` |
| "rename page", "change title" | Rename | `cfl page edit PAGE_ID --title "New Title"` |
| "move page", "reparent" | Move | `cfl page edit PAGE_ID --parent NEW_PARENT_ID` |
| "copy page", "duplicate" | Copy | `cfl page copy PAGE_ID --title "Copy Title"` |
| "delete page", "remove page" | Delete | `cfl page delete PAGE_ID` |

### Content Source Mapping (for create/edit)

| User Says | Flag | Notes |
|-----------|------|-------|
| "from file", "use this file" | `--file PATH` | Reads content from file |
| provides content inline | Pipe via stdin | `echo "content" \| cfl page create ...` |
| "open editor" | `--editor` | Opens interactive editor |
| "legacy format" | `--legacy` | Uses legacy editor format |
| "raw XHTML", "storage format" | `--storage` | Sends raw Confluence XHTML |

## Execute

### Create Page

```bash
# From file (most common)
cfl page create --space KEY --title "Page Title" --file content.md

# With parent page
cfl page create --space KEY --title "Child Page" --parent PARENT_ID --file content.md

# From stdin
echo "# Page Content" | cfl page create --space KEY --title "Page Title"

# From raw Confluence storage format (XHTML) — preserves macros exactly
cfl page create --space KEY --title "Page Title" --storage --file content.xhtml
```

The `--storage` flag works on `page create` as well as `page edit`. Use it when the source content is already raw Confluence XHTML (e.g. copied from another page's storage format).

If the user provides content as part of the request, write it to a temp file and use `--file`. Use `mktemp` to avoid collisions:
```bash
# Create a unique temp file
TMPFILE=$(mktemp /tmp/confluence-content-XXXXXX.md)

cat > "$TMPFILE" << 'CONTENT'
# The content here
CONTENT

cfl page create --space KEY --title "Page Title" --file "$TMPFILE"
rm -f "$TMPFILE"
```

### Edit Page

```bash
# Update content from file
cfl page edit PAGE_ID --file content.md

# Update title only
cfl page edit PAGE_ID --title "New Title"

# Move page to new parent
cfl page edit PAGE_ID --parent NEW_PARENT_ID

# Move and rename
cfl page edit PAGE_ID --parent NEW_PARENT_ID --title "New Title"
```

For editing existing content: view first, modify, then update. Use `mktemp` so concurrent edits don't collide:
```bash
# Get current content into a unique temp file
TMPFILE=$(mktemp /tmp/confluence-edit-XXXXXX.md)
cfl page view PAGE_ID --content-only > "$TMPFILE"

# (modify $TMPFILE)

# Push updated content
cfl page edit PAGE_ID --file "$TMPFILE"
rm -f "$TMPFILE"
```

#### Lossless Edit (Storage-Format Round-Trip)

The markdown round-trip above is convenient but **lossy** — macros (TOC, include, status, etc.) and some formatting are stripped. For edits that must preserve everything:

- Fetch the page with `-o json` (see ViewPage.md for the JSON structure) — the `content` field holds the raw storage XHTML
- Modify the XHTML as needed
- Send the modified XHTML back via `cfl page edit PAGE_ID --storage` — reads from stdin, or pass via `--file`

The `--storage` flag sends the input directly via the storage representation API, preserving all macros and formatting exactly. Use this whenever:
- The page contains macros you need to keep (TOC, include, excerpt, status, etc.)
- You're doing a find/replace that must not touch surrounding markup
- You're scripting against pages with complex structure

### Copy Page

```bash
# Copy within same space
cfl page copy PAGE_ID --title "Copy of Page"

# Copy to different space
cfl page copy PAGE_ID --title "Page Title" --space OTHER_KEY

# Copy without attachments or labels
cfl page copy PAGE_ID --title "Light Copy" --no-attachments --no-labels
```

**Placement:** `cfl page copy` always places the new page at the root of the destination space — it does not inherit the source page's parent, and it does not accept `--parent`. To place the copy under a specific page, follow the copy with a reparent edit:

```bash
NEW_ID=$(cfl page copy SOURCE_ID --title "Copy of Page" -o json | ...)   # capture the new page ID from JSON
cfl page edit $NEW_ID --parent DESIRED_PARENT_ID
```

Or do it as two explicit steps (capture ID from the table output, then edit).

### Delete Page

**Always confirm with user before deleting.** This is a destructive action.

```bash
cfl page delete PAGE_ID
```

## Post-Action

After any action:
1. For creates: show the new page ID and confirm title/space
2. For edits: confirm what was changed (content, title, parent)
3. For copies: show the new page ID and location
4. For deletes: confirm which page was deleted
5. For moves: confirm old and new parent
