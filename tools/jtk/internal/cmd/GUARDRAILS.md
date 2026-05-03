# jtk Command Surface — Design Guardrails

This document captures the design language of the `jtk` CLI: the conventions that govern flag names, verb choices, argument shapes, and behavior. It exists so that humans and AI agents adding new commands can do so consistently with what's already there, rather than rediscovering the same patterns each time.

The rules below are derived from the existing surface and refined through the convergence issues numbered `01`–`08`. When a rule has a corresponding issue, it's referenced inline.

---

## 1. Verb language

`jtk` uses a small set of verbs with specific meanings. Pick the right verb and the user knows the safety profile and lifecycle of the operation before reading any docs.

| Verb | Meaning | Aliases |
|---|---|---|
| `create` | Bring a new top-level resource into existence | — |
| `delete` | Destroy a resource | `rm` |
| `add` | Attach a child to a parent | — |
| `remove` | Detach a child from its parent (without destroying it) | — |
| `archive` | Soft-stash a resource (restorable) | — |
| `restore` | Undo a soft-delete or archive | — |
| `disable` / `enable` | Toggle a resource's active state without modifying its definition | — |
| `list` | Return a paginated collection | `ls` (where established) |
| `get` | Return one or more resources by identifier | — |
| `update` | Modify fields on an existing resource | — |
| `search` | List with a query argument (currently only on `users search`) | — |
| `types` | List enum values valid for a `--type` flag elsewhere | — |

**Implications:**

- `delete` destroys; `remove` detaches. They are not synonyms. `dashboards gadgets remove` doesn't delete the gadget definition — it removes it from a dashboard. `attachments delete` actually destroys the attachment.
- The presence of a `restore` sibling signals soft-delete (`projects delete`, `fields delete`). Hard-delete commands have no `restore` sibling and document irreversibility in their description.
- `<resource> types` always means "list the valid values for `--type` on `create`/`update` of this resource" (`projects types`, `links types`, `issues types`).

---

## 2. Resource references

### 2.1 Cache-backed identity resolution

Any flag or positional that takes a reference to a board, sprint, project, issue type, link type, transition, user, or field **must** accept the human-friendly form alongside the canonical ID, and must route through the `jtk` cache. The cache exists for this reason.

**Person references** specifically accept four forms:
- `accountId`
- email address
- display name
- the literal string `"me"`

This applies to `--lead`, `--assignee`, and the positional `<user>` on `issues assign`. The `"me"` sentinel is part of the contract and must be preserved on any new person-reference flag.

### 2.2 Positional vs flag (see issue #07)

Positional when the entity is:

1. The **primary subject** of the command — what's being acted on.
2. A **constitutive parent** — the parent of a child entity that can't exist without it (comments-of-issue, attachments-of-issue, contexts-of-field).
3. The **destination of an `add`-style** operation, where the command reads as "add to X."
4. The **second party in a binary relation** (`links create <a> <b>`, `transitions do <issue> <transition>`).

Flag when the entity is:

1. An **optional filter** narrowing a list.
2. A **required scope** that is neither subject nor constitutive parent (`sprints current --board`, `issues types --project`).
3. A **setter or payload** during create/update.
4. A `--to-<thing>` **destination** for `move`-style operations.

The rule is role-based, not type-based. The same entity type appears positionally in some commands and as a flag in others, depending on the role it plays in that specific command.

---

## 3. Reading commands (list / get / search)

### 3.1 Output-shape flags

The global flags `--id`, `--extended`, `--fulltext`, and `--fields` form a coordinate system for "how should the result render?" — orthogonal to *what* is fetched.

- **`--id`** — emit only primary identifiers. Overrides `--extended` and `--fulltext`. The contract: machine-friendly output, one identifier per line, suitable for piping into `xargs`.
- **`--extended`** — widen the default column set with admin/schema/audit fields.
- **`--fulltext`** — disable truncation of prose cells.
- **`--fields <csv>`** — explicit column selection. Replaces the default set entirely. Accepts header labels, Jira field IDs, or human names; the flag handles input normalization.

**Mental model:** there's a default column set → `--extended` widens it → `--fields` overrides the whole selection → `--id` short-circuits to identifiers only.

New list/get commands must support all four. Output-shape flags are always long-only.

### 3.2 Pagination (see issue #06)

Every paginated command takes:

- `-m, --max int` — page size, default **50** unless documented otherwise.
- `--next-page-token string` — cursor (long-only; rarely typed by hand).

A non-50 default requires a justification in the command's help text.

### 3.3 Filtering (see issue #02)

Three reserved filter flag names:

- **`--name <substring>`** — case-insensitive substring match against the resource's name field. Use this for any name-only filter. Always long-only.
- **`--query <string>`** — *reserved* for future multi-field full-text search. Don't use it for name-only filters. Always long-only.
- **`--search`** — banned. Verbs don't make good flag names.

List filters never take short aliases. Their job is to be explicit at the call site.

---

## 4. Writing commands (create / update / add)

### 4.1 Setter flags get short aliases

Required setters on `create`/`update` commands take short aliases. The canonical map:

| Short | Long |
|---|---|
| `-n` | `--name` (setter; see §3.3 for the filter case) |
| `-d` | `--description` |
| `-t` | `--type` |
| `-k` | `--key` |
| `-l` | `--lead` |
| `-a` | `--assignee` |
| `-b` | `--body` (or `--board` in scoping context — never co-occurring) |
| `-V` | `--value` |
| `-c` | `--context` |

Optional or rarely-used setters (`--parent`) stay long-only.

### 4.2 Input flags (see issue #05)

| Short | Long | Purpose |
|---|---|---|
| `-f` | `--field` | Repeatable `key=value` setter |
| `-F` | `--file` | Path to a payload file |
| `-o` | `--output` | Path to write output to |

The lowercase/uppercase distinction between `-f` and `-F` is meaningful and reserved.

### 4.3 Scoping flags

| Short | Long |
|---|---|
| `-p` | `--project` |
| `-b` | `--board` |
| `-s` | `--sprint` |
| `-m` | `--max` |

These bind to specific scopes. Don't reuse them for unrelated purposes.

### 4.4 No shorts for these

- Output-shape flags (`--fields`, `--extended`, `--fulltext`, `--id`)
- Pagination cursors (`--next-page-token`)
- Safety flags (`--force`)
- Boolean toggles (`--unassign`, `--notify`, `--wait`, `--compact`, `--show-components`, `--custom-fields`, etc.)
- Enum filters (`--state`)
- List filters (`--name`, `--query`)
- One-off operation knobs (`--to-project`, `--to-type`, `--option`, `--auth-method`, `--cloud-id`, etc.)

The principle: things you type often get shorts; things you toggle once or specify rarely don't.

### 4.5 Principled exceptions

When a short alias would collide (`fields options ... --option` would want `-o`, but `-o` is `--output`), leave the flag long-only rather than invent a non-mnemonic letter.

---

## 5. Mutation safety (see issue #04)

### 5.1 The risk-tiered rule

> **If a user runs this by accident, can they trivially undo it from another `jtk` command without external recovery?**
>
> - **Yes** → no prompt, no `--force` flag.
> - **No** → prompt by default, accept `--force` to skip.

"Trivially reversible" means *one short jtk command away*. If undoing requires the user to remember state they no longer have access to (the contents of a comment, the bytes of an attachment), it isn't trivially reversible.

### 5.2 Corollaries

- The presence of a `restore` sibling does not automatically make a delete low-risk. `projects delete` is restorable but still warrants the prompt because the soft-delete window can lapse and the impact radius is large.
- The flag is always spelled `--force`. Don't reach for `--yes`, `--confirm`, or `-y`.
- `--force` is long-only.

---

## 6. Boolean flags (see issue #08)

### 6.1 Defaults

A boolean flag's default should match the safest or most common case. `--wait` defaults to true because synchronous is more predictable; `--notify` defaults to true because notifications are usually wanted.

### 6.2 Documentation discipline

Any default-true boolean must:

- State the default explicitly in help text.
- Explicitly document the `--no-X` negation form alongside the positive form.
- Show both behaviors in examples.

pflag does not auto-generate `--no-X`. Register `--no-X` explicitly alongside the positive flag (e.g., `--no-wait` alongside `--wait`); in `RunE`, apply the override before use (`if noWait { wait = false }`). Help text and examples are the only surface where the negation becomes discoverable.

---

## 7. Async operations

For operations that are async on the Atlassian side (currently only `issues move`), the convention is:

- The originating command takes `--wait` (default true) and runs synchronously by default.
- Passing `--no-wait` returns immediately with a task ID.
- A companion `<command>-status <task-id>` command polls the operation's status.

If new async surface is added, follow this shape: same flag name, same default, same companion-command pattern. Don't invent `--async`, `--background`, etc.

---

## 8. Naming hygiene

### 8.1 Consistency with existing precedent

Before naming a new flag, search this doc and the command reference for an existing flag that does the same job. Reuse the name. New synonyms (`--search` for `--name`, `--query` for `--name`) are how the surface gets messy.

### 8.2 Aliases

Top-level commands use short, conventional aliases (`projects` → `project`, `proj`, `p`). New top-level commands should follow the pattern: full name, singular form, and a one- or two-letter abbreviation if available without conflict.

Subcommands use `ls` for `list` where established (e.g., `attachments list` has `ls`). Don't invent new subcommand aliases beyond the established ones (`rm` for `delete`, `ls` for `list`).

---

## Decision log

| Issue | Subject |
|---|---|
| #01 | Short-alias consistency (gaps fixed; reserved-letter table established) |
| #02 | Filter-by-name consolidation on `--name`; `--query` reserved; `--search` banned |
| #03 | `rm` as universal alias for `delete`; verb table established |
| #04 | Risk-tiered `--force` rule |
| #05 | `-F`/`-f` swap (`--file`/`--field`) |
| #06 | Pagination default convergence to 50 |
| #07 | Positional-vs-flag rule for entity references |
| #08 | Boolean flag documentation convention |

---

## For AI agents adding new commands

Read sections 1–6 in order. Pick verbs and flag names from the tables, not from intuition. If a command needs a flag whose role isn't covered here, reuse the closest existing precedent rather than coining a new name. When in doubt, follow these rules over what feels natural — consistency across the surface beats local elegance.
