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
- `cfl page view --full` → agent fields + version, created, modified, author

## Raw Mode (Command-Specific)

Some commands expose a `--raw` mode for source-faithful content where transformation would lose fidelity. This is **not** a general artifact type—it applies only to commands that transform content.

- Shows original storage format (XHTML, ADF JSON) instead of transformed content
- Errors on commands where raw has no meaning (e.g., list commands)
- Mutually exclusive with `--full`

**Example:**
- `cfl page view --raw` → XHTML storage format instead of markdown

## Flag Behavior

```
(none)       → agent artifact (curated, transformed content)
--full       → full artifact (richer curation, transformed content)
--raw        → raw mode (source-faithful content, command-specific)
--full --raw → error: mutually exclusive
```

> **Implementation note:** The mutual exclusivity constraint (`--full --raw → error`) and command-specific `--raw` validation are forward-looking requirements. They will be enforced as commands are migrated in #199 and #200.

## Output Format Interaction

Artifact type and output format (`-o table|json|plain`) are independent concerns:

**JSON output:**
- `agent` = curated JSON with action-relevant fields (defined per-command)
- `full` = richer JSON with inspection fields (defined per-command)
- `raw` = source format as JSON-escaped string or native structure (command-specific)

**Table output:**
- `agent` = focused columns for action (defined per-command)
- `full` = additional columns for inspection (defined per-command, still curated)
- `raw` = error (raw is about content fidelity, not list views; commands should reject `--raw -o table`)

## Design Principles

1. **Intentional artifacts, not field stripping.** Commands project domain objects into purpose-built artifacts. They don't start with everything and strip fields away.

2. **Agent is the default.** LLM/agent consumption is the primary use case. Human inspection is opt-in via `--full`.

3. **Raw is command-specific.** Not every command needs `--raw`. It's only for commands where content transformation occurs.

4. **Curated, not pass-through.** Even `--full` is curated by the CLI. Raw API payloads are not exposed directly.

## Migration from --compact

The `--compact` flag is deprecated. Its behavior is replaced by agent-default semantics:

| Old | New |
|-----|-----|
| Default output | Now produces agent artifact (was full-ish) |
| `--compact` | Deprecated (agent is now default) |
| Explicit full output | Use `--full` |
