# cfl Presenter Migration Guide

This package is the implementation-local home for `cfl` presenter-boundary
guidance. Shared rules live in `ARCHITECTURE.md`; cfl output shapes live in
`tools/cfl/internal/cmd/OUTPUT_SPEC.md`.

## Current State

`tools/cfl/internal/present/emit.go` already contains the narrow `Emit` bridge.
It accepts a shared `present.OutputModel`, renders it with the shared pure
renderer, and writes the split stdout/stderr result through root options.

`tools/cfl/internal/cmd/root.Options` already exposes `RenderMode()` and
`RenderStyle()`. Legacy `Options.View()` remains only for documented
exceptions, primarily `cfl init`, and derives from that same root mode so cfl
does not grow competing output policy knobs.

Default cfl text output is presenter-backed after #271. This guide documents
the boundary that new command work must preserve.

## Target Pipeline

```text
domain/API data
-> projection helpers when needed
-> cfl presenter
-> present.OutputModel
-> shared present.Render
-> present.Emit
-> opts.Stdout / opts.Stderr
```

The command should orchestrate this pipeline. It should not flatten domain data
into final display strings before the presenter sees it.

## Responsibilities

### Commands

Commands may:

- validate CLI arguments and flags
- load config and API clients
- call domain/API functions
- choose between control-plane JSON, raw content, browser/editor handoff, and
  presenter-backed text output
- call a presenter and `present.Emit`

Commands must not:

- assemble user-facing tables, key-value lines, or success messages
- construct presenter-owned DTOs, rows, fields, labels, or section order
- decide pagination or advisory wording for final rendered output
- normalize, truncate, or escape display content when that choice belongs to
  presentation logic

### Projection Helpers

Projection helpers prepare domain-side facts that presenters need but should not
discover themselves. Config and environment source resolution belongs here, not
inside presenters. A presenter may receive a value such as `URLSource: "env"`;
it should not inspect `CFL_URL`, read config files, or query keyring state.

### Presenters

Presenters own domain-to-presentation mapping:

- field selection
- labels and ordering
- section ordering
- table columns and row cells
- empty-state messages
- pagination and advisory messages
- mutation wording
- stream destination for messages, warnings, and diagnostics

Presenters return `present.OutputModel`. Message sections should set
`present.StreamStderr` for diagnostics, warnings, advisory output, progress, and
other commentary. Primary results and artifacts stay on stdout.

### Renderer

The shared renderer owns layout and style only. It may render human or
agent-friendly table/detail/message layouts from `present.OutputModel`; it must
not inspect cfl API types, infer missing domain values, repair command-built
strings, or decide which fields matter.

## Stream Semantics

- stdout is the primary result or artifact stream.
- stderr is for warnings, diagnostics, advisory text, progress, and prompts.

For presenter-backed output, stream routing is represented in the
`present.OutputModel`, usually through message section stream metadata. Commands
should not write extra explanatory text around presenter output.

## Explicit Exceptions

The presenter migration does not need to model every byte of interactive IO.
These exceptions are allowed when deliberate and tested:

- `cfl init` wizard output
- one-shot confirmation prompts such as delete confirmations
- editor handoffs for page create/edit
- browser handoffs such as `page view --web`
- root `Options.View()` plumbing while an allowed exception still needs
  `shared/view`

Exceptions should stay small and named in code review. They must not become a
general escape hatch for command-local formatting.

Source-faithful modes such as `page view --raw` and `--content-only` are not
presenter-boundary exceptions. They are presenter/projection-owned output modes
whose content selection is intentional.

Progress messages that intentionally complete later may use
`present.MessageSection{NoNewline: true}`. The wording and stream still belong
to the presenter; commands should only decide when to emit the progress model.

## Presenter Inventory

Keep output ownership grouped by output shape rather than by scattered helper
replacement.

### Table/List

- `space list`
- `page list`
- `page history list`
- `search`
- `attachment list`

List presenters own headers, row fields, empty states, pagination hints, and
`-o plain` TSV semantics through the shared renderer.

### Detail

- `space view`
- `page view` metadata
- `config show`
- `me`

Detail presenters own labels, ordering, optional-field omission, and any
domain-side source values passed in by projection helpers.

### Mutation/Composite Success

- space create/update/delete
- page create/edit/copy/delete
- attachment upload/download/delete

Mutation presenters own the success summary plus follow-up handles such as `ID`,
`Version`, `Key`, and `URL`.

### Diagnostic/Composite Status

- `config test`
- `config clear`
- list/search pagination hints
- warnings such as legacy-editor advisories

Diagnostic presenters must route commentary to stderr through the output model.

### Tricky: `page view`

`page view` is not a mechanical key-value rewrite. It mixes metadata, converted
body content, raw/source-faithful modes, truncation, content-only output, web
handoff, historical versions, and macro rendering. Split its migration into
separate presenter/projection decisions for metadata, body mode, and advisory
messages.

## Verification Gates

`tools/cfl/internal/cmd/root/presenter_boundary_test.go` is the authoritative
package-wide enforcement gate. It scans production command files with Go's AST
and allowlists only documented exceptions.

Use these greps as human-readable proof commands. Unexpected matches should be
explained by the enforcement allowlist or fixed.

The target patterns include legacy helpers such as `v.Table`, `v.Success`, and
`v.RenderKeyValue`, plus direct writes such as `fmt.Fprint`, `fmt.Fprintf`, and
`fmt.Fprintln`.

Legacy view helpers in cfl commands:

```bash
rg -n '\bv\.(Table|Success|RenderKeyValue|RenderKeyValues|Info|Warning|Error|Println|Render)\b' tools/cfl/internal/cmd --glob '!**/*_test.go'
```

Direct command-local output text:

```bash
rg -n 'fmt\.F(print|printf|println)\((opts\.(Stdout|Stderr)|v\.Out|os\.Stderr),\s*"' tools/cfl/internal/cmd --glob '!**/*_test.go'
```

Legacy output plumbing that should disappear from primary presenter-backed text
paths as migration completes:

```bash
rg -n 'view\.ValidateFormat|opts\.View\(|github.com/open-cli-collective/atlassian-go/view' tools/cfl/internal/cmd --glob '!**/*_test.go'
```

Presenter tests should assert exact `present.OutputModel` values. Renderer
tests should assert exact stdout/stderr strings for representative cfl shapes.
Command tests should stay lighter and verify wiring, mode selection, exceptions,
and preserved control-plane JSON/raw behavior.

## #271 Proof Index

- `docs/proofs/271-431-cfl-me.md`
- `docs/proofs/271-432-cfl-list-search.md`
- `docs/proofs/271-433-cfl-detail-config.md`
- `docs/proofs/271-434-cfl-page-view.md`
- `docs/proofs/271-435-cfl-mutation-success.md`
- `docs/proofs/271-436-cfl-diagnostics-advisories.md`

Proof transcript directories under `/tmp` are intentionally ephemeral. The
proof files contain the durable redacted excerpts and created/deleted IDs needed
to audit behavior after those temporary directories disappear.
