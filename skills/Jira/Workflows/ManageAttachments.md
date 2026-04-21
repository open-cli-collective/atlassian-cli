# ManageAttachments Workflow

List, upload, download, and delete attachments on Jira issues.

## Intent-to-Flag Mapping

### Action Selection

| User Says | Command | Required |
|-----------|---------|----------|
| "list attachments", "show attachments" | `jtk attachments list PROJ-123` | Issue key |
| "attach file", "upload file", "add attachment" | `jtk attachments add PROJ-123 --file PATH` | Issue key + file path |
| "download attachment", "get attachment" | `jtk attachments get ATTACHMENT_ID` | Attachment ID |
| "delete attachment" | `jtk attachments delete ATTACHMENT_ID` | Attachment ID |

The `get` command has an alias `download` — both work identically.

## Execute

### List Attachments

```bash
jtk attachments list PROJ-123
```

### Upload Attachment

Single file:
```bash
jtk attachments add PROJ-123 --file /path/to/file
```

Multiple files (the `--file` / `-f` flag is repeatable):
```bash
jtk attachments add PROJ-123 --file doc.pdf --file image.png
```

Verify files exist before attempting upload. If a path doesn't exist, ask the user for the correct path.

### Download Attachment

```bash
# First list to get attachment IDs
jtk attachments list PROJ-123

# Download to current directory (uses original filename)
jtk attachments get ATTACHMENT_ID

# Download to a specific directory
jtk attachments get ATTACHMENT_ID --output ./downloads/

# Download with a custom filename
jtk attachments get ATTACHMENT_ID --output ./downloads/renamed.pdf
```

If the user specifies by filename rather than ID: list attachments first, match by name, then download by ID.

### Delete Attachment

**Agent must confirm with user before calling this command.** `jtk attachments delete` executes immediately with no interactive CLI-level prompt — there is no safety net in the tool itself. This is a destructive action; the agent is responsible for the pre-call confirmation.

```bash
jtk attachments delete ATTACHMENT_ID
```

## Output Format

- **List:** Table with ID, Filename, Size, Author, Created date
- **Upload:** Confirm success with filename(s) and issue key
- **Download:** Confirm download location and filename
- **Delete:** Confirm which attachment was deleted

## Post-Action

After any action:
1. For list: state total attachment count
2. For upload: confirm filename(s), size, and issue key
3. For download: confirm download path and filename

## Notes

- **Attachment IDs are numeric** (long integers) and come from the output of `jtk attachments list PROJ-123` — there's no "friendly" path to a specific attachment. Always list first if the user refers to an attachment by filename.
- Large file uploads may take time — inform user if file is substantial
- Attachment size limits depend on Jira instance configuration (default 10MB for Cloud)
