# ManageComments Workflow

View and add comments on Jira issues.

## Intent-to-Flag Mapping

### Action Selection

| User Says | Command | Required |
|-----------|---------|----------|
| "view comments", "list comments", "show comments" | `jtk comments list PROJ-123` | Issue key |
| "add comment", "comment on", "post comment" | `jtk comments add PROJ-123 --body "TEXT"` | Issue key + `--body` text |
| "delete comment", "remove comment" | `jtk comments delete PROJ-123 COMMENT_ID` | Issue key + comment ID |

## Execute

### List Comments

```bash
jtk comments list PROJ-123
```

Optional: `--max N` to control result count (default 50). Use `--fulltext` global flag to disable truncation of long comment bodies.

### Add Comment

The comment text goes through the **required** `--body` flag (or `-b`):

```bash
jtk comments add PROJ-123 --body "Comment text here"
```

Do **not** pass the comment as a positional argument — it will be rejected with "required flag(s) \"body\" not set".

For multi-line comments, either pass the literal newline in the quoted string, or use the `\n` escape sequence (`--body` supports `\n`, `\t`, `\\`):
```bash
# Literal newlines (works in most shells)
jtk comments add PROJ-123 --body "Line 1
Line 2
Line 3"

# Escape sequences (works everywhere)
jtk comments add PROJ-123 --body "Line 1\nLine 2\nLine 3"
```

### Delete Comment

**Agent must confirm with user before calling this command.** `jtk comments delete` executes immediately with no interactive CLI-level prompt. The operation is destructive and cannot be undone; the agent is responsible for the pre-call confirmation.

```bash
jtk comments delete PROJ-123 COMMENT_ID
```

The `COMMENT_ID` is a numeric ID (e.g., `12345`). If the user refers to the comment by content or author, list comments first (`jtk comments list PROJ-123`) to recover the ID, then delete by ID.

## Output Format

- **List comments:** Show each comment with author, timestamp, and body
- **Add comment:** Confirm comment was added with issue key and a snippet of the comment text

## Post-Action

After any action:
1. For list: state total comment count
2. For add: confirm the comment was posted with issue key and author
3. When adding comments, remind user it will be posted as the authenticated user

## Notes

- Comments support Jira wiki markup / markdown depending on instance configuration
- Comment bodies are truncated by default in list output — pass `--fulltext` to see full text
