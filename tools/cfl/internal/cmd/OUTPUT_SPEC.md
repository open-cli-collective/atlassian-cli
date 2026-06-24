# CFL Output Specification

This document is the authoritative declaration of the target `cfl` output contract.
It defines the text shapes, flag semantics, and token-density rules that `cfl`
must converge on as the presenter-boundary work lands in #271.

Status: `cfl` is already text-first and resource `-o json` is already removed.
Some commands still render through transitional `shared/view` helpers rather than
through a dedicated `tools/cfl/internal/present` package. Where current behavior
does not yet match this document exactly, this spec is the contract to implement.

## Design principles

1. **Text is the primary format.** Stable key-value blocks and unpadded delimited
   rows are easier for both humans and agents to read than escape-heavy JSON.
2. **Default output is agent-first.** The default shape should carry the fields
   needed to decide the next action without extra flags.
3. **`--full` is additive, not alternate.** It reveals richer inspection detail
   on the same text surface instead of switching to a different representation.
4. **Page content defaults to readable markdown.** `page view` should expose
   transformed markdown by default and reserve `--raw` for source-faithful storage.
5. **Padding is not semantics.** Delimiters and labels are stable; visual padding,
   ASCII-boxing, and JSON wrappers are not part of the contract.
6. **Control-plane JSON is separate.** Local boolean `--json` flags such as
   `cfl set-credential --json` remain allowed because they are not resource output
   modes and do not participate in the global `-o` contract.

## Output modes

| Mode | Surface | Purpose |
|---|---|---|
| Default | `-o table` | Canonical human + agent text. Detail commands use stable key-value blocks; list commands use stable delimited rows with headers. |
| Plain | `-o plain` | Script-oriented dense text. For list commands this is TSV. For detail/mutation commands it must remain semantically identical to default text, differing only in presentation details such as color. |
| Full | `--full` | Inspection-oriented additive fields on top of the default/plain contract. |
| Raw | command-specific `--raw` | Source-faithful content for commands that transform body content, primarily `page view`. |

## Global flag semantics

- `-o/--output` accepts `table` and `plain` only.
- Resource `-o json` is invalid and must fail at the root with
  `invalid output format: "json" (valid formats: table, plain)`.
- `--no-color` may change decoration only; it must not change fields or ordering.
- `--full` selects the richer artifact for commands that support it.
- Local control-plane `--json` flags remain command-local and must not be blocked
  by the global output guard.

## Formatting conventions

### Lists

- Header row in ALL_CAPS.
- Default rows use a stable delimited text shape with no semantic dependence on
  alignment padding.
- `-o plain` emits TSV for list commands.
- Empty values render as `-`.
- Empty results emit a single prose line to stderr.
- Pagination hints, when present, are a trailing stderr line:

```text
(showing first N results, use --limit to see more)
```

### Detail views

- Stable `Label: Value` lines, one field per line.
- Content-bearing commands may separate metadata from body with one blank line.
- Omit absent optional fields rather than rendering empty labels, unless a fixed
  single-line row is explicitly part of the command contract.

### Mutations

- Success output is text, not JSON.
- The first line is a short success summary naming the affected resource.
- Follow-on key-value lines expose the identifiers and follow-up handles needed
  for chaining, such as `ID`, `Version`, or `URL`.
- Delete-style mutations may collapse to a single confirmation line when no
  follow-up handles are needed.

## Command contracts

## `me`

Default:

```text
accountId | displayName | email
```

`--id`:

```text
accountId
```

Notes:
- The default row is always exactly three pipe-delimited fields.
- Missing fields render as `-`.

## `space list`

Default columns:

```text
ID | KEY | TYPE | NAME
```

`--full` additions:

```text
ID | KEY | TYPE | STATUS | NAME
```

Notes:
- `TYPE` must reflect the normalized Confluence v2 value used by filtering and
  view output, not a stale alias.
- `-o plain` is TSV with the same columns and ordering.

## `space view <space-key>`

Default:

```text
Key: <key>
Name: <name>
ID: <id>
Type: <type>
```

`--full` additions:

```text
Status: <status>
Description: <plain description>
```

## `space create`

Success:

```text
Created space: <name>
Key: <key>
URL: <url>
```

## `space update <space-key>`

Success:

```text
Updated space: <name> (<key>)
```

`--full` may add inspection fields when the command is migrated to presenter-owned
post-state output, but the identifier-bearing summary line remains required.

## `space delete <space-key>`

Success:

```text
Deleted space: <name> (<key>)
```

## `page list`

Default columns:

```text
ID | TITLE | STATUS
```

`--full` additions:

```text
ID | TITLE | STATUS | VERSION | PARENT ID
```

Notes:
- Space context comes from the command argument/default and is not repeated in
  every default row.
- `-o plain` is TSV with the same columns and ordering.

## `page view <page-id>`

Default metadata block:

```text
Title: <title>
ID: <id>
Space: <space-key> (ID: <space-id>)
Version: <version>

<markdown body>
```

`--full` target additions:

```text
Parent ID: <parent-id>
Created At: <timestamp>
Author ID: <account-id>
```

Flag-specific behavior:
- `--content-only` emits only the body and implies untruncated output.
- `--raw` emits the source body instead of transformed markdown.
- `--show-macros` affects body conversion only.
- `--no-truncate` disables the default body truncation guard.
- `--version N` selects a historical version and preserves the same output shape.
- `--web` opens the browser and emits no CLI body text.

Truncation trailer:

```text
... [truncated at 5000 chars, use --no-truncate for complete text]
```

## `page history list <page-id>`

Default columns:

```text
VERSION | CREATED | AUTHOR
```

`--id`:

```text
<version>
```

## `page create`

Success:

```text
Created page: <title>
ID: <id>
URL: <url>
```

## `page edit <page-id>`

Success:

```text
Updated page: <title>
ID: <id>
Version: <version>
URL: <url>
```

## `page copy <page-id>`

Success:

```text
Copied page: <title>
ID: <id>
Space: <space-id-or-key>
Version: <version>
```

The implementation may currently use a different verb or omit the summary line;
the target contract is a success summary plus the follow-up handles shown above.

## `page delete <page-id>`

Success:

```text
Deleted page: <title> (ID: <id>)
```

## `attachment list --page <page-id>`

Default columns:

```text
ID | TITLE | MEDIA TYPE | FILE SIZE
```

`--full` target:

- Same list surface, with richer attachment inspection fields added only when
  presenter-backed output is introduced.
- `--full` must never be a silent no-op once the presenter migration lands.

Empty states:

```text
No attachments found.
No unused attachments found.
```

## `attachment upload`

Success:

```text
Uploaded: <filename>
ID: <attachment-id>
Title: <title>
Size: <human-readable size>
```

The displayed size must reflect the uploaded file size, not a zero-value field
from a partial API response.

## `attachment download <attachment-id>`

Success:

```text
Downloaded: <output-path>
Size: <human-readable size>
```

## `attachment delete <attachment-id>`

Success:

```text
Deleted attachment: <title> (ID: <id>)
```

## `search [query]`

Default columns:

```text
ID | TYPE | SPACE | TITLE
```

`--full` additions:

```text
ID | TYPE | SPACE | TITLE | MODIFIED | URL
```

Notes:
- The space column label is `SPACE`, not `SPACE KEY`. Current search output still
  renders the space key in that column, extracted from the result URL.
- Default search output must not require JSON decoding, unescaping XHTML, or
  double-decoding Unicode escape sequences.
- `--type` validation accepts the documented values and applies even when a
  free-text query is omitted in favor of other filters.

## Historical notes and non-goals

- Historical docs may still mention removed resource JSON paths as obsolete test
  rows. Those notes are allowed when clearly marked historical.
- This spec does not reintroduce any resource JSON format.
- This spec does not require every command to migrate in the same PR; #271 owns
  the presenter/render-mode plumbing that will make these shapes uniform.
