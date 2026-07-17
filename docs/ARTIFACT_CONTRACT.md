# Output Artifact Contract

This document defines the output representation model for `jtk` and `cfl`. It establishes the vocabulary and semantics that all commands must follow.

## Artifact Types

Two general artifact types apply across all commands:

| Type | Purpose | Default? | Flag |
|------|---------|----------|------|
| `agent` | Action-oriented output for LLM/agent consumption. Contains what's needed to decide the next step. | Yes | (none - default) |
| `full` | Inspection-oriented output for human debugging. Richer curated fields beyond agent. | No | `--full` |

### agent (default)

The agent artifact is intentionally curated output containing what you need to decide the next step.

- Each command defines its agent artifact: which fields are action-relevant
- Content is transformed for readability (e.g., markdown instead of XHTML)
- Strips transport metadata (self URLs, `_links`, `_expandable`)
- Strips visual-only fields (`avatarUrls`)

**Examples:**
- `jtk issues get` → key, summary, status, assignee (enough to triage)
- `cfl page view` → id, title, space, ancestors, content as markdown (enough to navigate and act)

### full

The full artifact provides richer inspection-oriented output for debugging and deeper understanding.

- Each command defines additional fields beyond agent (dates, authors, versions, relationships)
- Still curated by the CLI, not a pass-through of API payloads
- Content transformation same as agent (readable format)

**Examples:**
- `jtk issues get --full` → agent fields + created, updated, reporter, components, labels
- `cfl page view --full` → agent fields + parent ID, creation time, author ID

## Page Body Format (cfl)

`cfl page view`, `page create`, and `page edit` select body representation independently from artifact breadth with `--body-format markdown|adf|xhtml`.

- Omission means Markdown.
- `adf` is exact Atlassian Document Format JSON.
- `xhtml` is exact Confluence storage XHTML.
- Markdown conversion fails closed; it never emits raw ADF or XHTML as Markdown.

**Example:**
- `cfl page view PAGE_ID --body-format xhtml` → exact storage XHTML

## Flag Behavior

```
(none)                  → agent artifact with Markdown page bodies
--full                  → full artifact with Markdown page bodies
--body-format adf       → selected artifact with exact ADF body
--body-format xhtml     → selected artifact with exact storage XHTML body
```

`cfl page view --full` cannot be combined with `--content-only` or `--web`.
Commands without a defined full artifact reject `--full` instead of ignoring it.

## Output Format

JTK and CFL both use text-first output. The `-o json` resource surface has been removed from both tools (JTK earlier, then CFL via #392); CFL retains `-o table` and `-o plain` only. JSON is reserved for control-plane envelopes (`cfl set-credential --json`, `jtk set-credential --json`) and round-trip payloads (`jtk automation export`).

`jtk` and `cfl` both use presenter-owned text output for default CLI rendering. `cfl`'s command/output contract is defined in `tools/cfl/internal/cmd/OUTPUT_SPEC.md`; its presenter-boundary guidance and documented exceptions live in `tools/cfl/internal/present/README.md`.

**Text output modes:**
- Default = focused output for human and agent consumption (defined per-command)
- Tool-specific detail/inspection flags stay tool-local: `jtk` uses flags such as
  `--extended`, `--id`, and `--fulltext`; `cfl` uses `--full` plus command-specific
  flags documented in `tools/cfl/internal/cmd/OUTPUT_SPEC.md`

## Design Principles

1. **Intentional artifacts, not field stripping.** Commands project domain objects into purpose-built artifacts. They don't start with everything and strip fields away.

2. **Agent is the default.** LLM/agent consumption is the primary use case. Human inspection is opt-in via each tool's additive inspection flag set.

3. **Body representation is independent.** CFL page commands use `--body-format`; artifact breadth remains controlled separately.

4. **Curated, not pass-through.** Even `--full` is curated by the CLI. Raw API payloads are not exposed directly.

## History: --compact flag

The global `--compact` flag was removed after the artifact projection migration. Its behavior (stripping null fields, avatarUrls, self-links, `_links`, `_expandable`) is superseded by agent-default semantics — commands now produce intentionally shaped artifacts rather than post-processing raw API payloads.

Note: `jtk automation export --compact` is unrelated — it controls JSON formatting (minified vs pretty-printed), not metadata stripping.
