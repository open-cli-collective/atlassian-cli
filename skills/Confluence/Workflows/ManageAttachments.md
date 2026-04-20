# ManageAttachments Workflow

List, upload, download, and delete attachments on Confluence pages.

## Intent-to-Flag Mapping

### Action Selection

| User Says | Command | Required |
|-----------|---------|----------|
| "list attachments", "show attachments" | `cfl attachment list --page PAGE_ID` | Page ID |
| "orphaned attachments", "unused attachments" | `cfl attachment list --page PAGE_ID --unused` | Page ID |
| "upload file", "attach file", "add attachment" | `cfl attachment upload --page PAGE_ID --file PATH` | Page ID + file path |
| "download attachment", "get attachment" | `cfl attachment download ATT_ID` | Attachment ID |
| "delete attachment", "remove attachment" | `cfl attachment delete ATT_ID` | Attachment ID |

## Execute

### List Attachments

```bash
cfl attachment list --page PAGE_ID

# Increase result count (default 25)
cfl attachment list --page PAGE_ID --limit 100

# Show only unused/orphaned attachments
cfl attachment list --page PAGE_ID --unused
```

### Upload Attachment

```bash
# Basic upload
cfl attachment upload --page PAGE_ID --file /path/to/file

# Upload with comment
cfl attachment upload --page PAGE_ID --file /path/to/file -m "Description of attachment"
```

Verify the file exists before attempting upload. If the path doesn't exist, ask the user for the correct path.

### Download Attachment

```bash
# Download with original filename into the current working directory
cfl attachment download ATT_ID

# Download to a specific file path
cfl attachment download ATT_ID -O /path/to/output
```

- Without `-O`, the file is saved in the current working directory using the attachment's original filename.
- `-O` expects a **file path** (not a directory path).
- **Overwrite behavior:** if the target file already exists, the download fails with an error suggesting `--force`. To avoid relying on `--force`, choose a path that doesn't exist or remove the existing file first.

If user specifies by filename rather than ID, list attachments first, match by name, then download by ID.

### Delete Attachment

**Always confirm with user before deleting.** This is a destructive action.

```bash
cfl attachment delete ATT_ID
```

## Output Format

- **List:** Table with ID, Filename, Size, Created date
- **Upload:** Confirm success with filename and page ID
- **Download:** Confirm download location and filename
- **Delete:** Confirm which attachment was deleted

## Post-Action

After any action:
1. For list: state total attachment count
2. For upload: confirm filename and page ID
3. For download: confirm download path and filename
4. For delete: confirm deletion

## Notes

- Attachment size limits depend on Confluence instance configuration
- Use `--unused` flag to find orphaned attachments not referenced in page content
