# Rendering Architecture

This document defines the output architecture for this repository.

It exists to prevent a recurring anti-pattern across the Go CLIs in this repo:

- commands fetch Atlassian data
- commands assemble ad hoc display strings
- shared view helpers rewrite or reshape those strings

That approach is easy to start and hard to scale. It creates weak boundaries, stringly-typed rendering code, and brittle tests.

The target architecture is a clean, testable pipeline:

```text
Atlassian/API domain data -> presenter/formatter -> presentation model -> renderer -> CLI output
```

## Goals

- Keep domain concerns separate from presentation concerns
- Make each stage of the output pipeline pure and easy to unit test
- Make new formatters and renderers easy to add
- Prevent `shared/view` from becoming a string-massaging utility
- Support multiple output styles without embedding formatting rules in commands

## Layers

### 1. Domain layer

This layer contains:

- Atlassian API response types
- internal domain projections
- artifact projections where already established

It should not contain display layout concerns.

Examples:

- `tools/jtk/api/...`
- `tools/cfl/api/...`
- `internal/artifact/...`

### 2. Presenter / formatter layer

This layer maps domain values into a presentation model.

It is responsible for deciding:

- which fields are shown
- field labels
- field ordering
- section ordering
- truncation
- normalization
- empty-state wording
- whether a value is represented as detail, table, message, or nested sections

This layer should be pure:

- input: domain value
- output: presentation model
- no IO
- no CLI side effects

### 3. Renderer layer

This layer turns a presentation model into a concrete output style.

Examples:

- agent renderer
- human renderer
- table-oriented renderer
- plain text renderer

This layer is responsible for layout and style only. It should not:

- inspect raw API types
- decide which domain fields are important
- normalize arbitrary content strings after the fact
- escape or repair display content that should have been handled by the presenter

It should be pure:

- input: presentation model
- output: rendered string or bytes
- no IO

### 4. CLI command layer

Commands should orchestrate, not format.

The command layer should:

- fetch domain data
- choose the right presenter
- choose the right renderer or output mode
- write final output

The command layer should not:

- manually build user-facing tables
- manually assemble `Key: Value` strings
- perform inline truncation/escaping/label selection except as transitional code during migration

## Required separation

The key boundary in this repo is:

```text
domain model != presentation model != rendered string
```

If a function accepts or returns `[][]string`, `[]string`, or preformatted display text as its primary contract, that is usually too low-level for new architecture work.

Prefer explicit presentation types.

## Presentation model

The presentation model should be small and boring. It does not need to be generic-heavy or framework-like.

A minimal shape is enough:

```go
type OutputModel struct {
    Sections []Section
}

type Section interface {
    isSection()
}

type DetailSection struct {
    Fields []DetailField
}

type TableSection struct {
    Columns []Column
    Rows    []TableRow
}

type MessageSection struct {
    Kind    MessageKind
    Message string
}

type DetailField struct {
    Label string
    Value string
}

type Column struct {
    Key   string
    Label string
}

type TableRow struct {
    Cells []Cell
}

type Cell struct {
    Value string
}
```

This is illustrative, not mandatory. The important point is that presenters return structured display intent, and renderers turn that into final text.

## Preferred interfaces

For most cases, prefer a structured presenter plus renderer split:

```go
type Presenter[T any] interface {
    Present(T) OutputModel
}

type Renderer interface {
    Render(OutputModel) string
}
```

This keeps the renderer dumb and reusable.

Avoid collapsing everything into a single formatter that directly returns strings unless the scope is extremely small and unlikely to grow.

## Output styles in this repo

This repository already distinguishes between:

- artifact shape
- output format
- command intent

See [docs/ARTIFACT_CONTRACT.md](docs/ARTIFACT_CONTRACT.md).

That contract still applies. This document adds the architecture for how those outputs should be constructed.

In practice:

- artifact selection decides what information is relevant
- presenter shapes that information into a presentation model
- renderer decides how that presentation model is displayed

## JSON and artifact outputs

JSON and artifact paths are first-class output pathways. They are not fallback representations and should not be incidental byproducts of text rendering.

Rules:

- preserve existing JSON/artifact contracts unless intentionally changing them
- do not route JSON through text-focused helpers
- do not use text-oriented helpers as the canonical output model for JSON

The same domain value may support:

- artifact projection for `-o json`
- presentation modeling for text output

These are related, but not identical, responsibilities.

## Hard rules

For new rendering work in this repo:

1. Do not build user-facing strings in commands except for trivial dispatch or temporary migration glue.
2. Do not make `shared/view` responsible for rewriting arbitrary content strings.
3. Do not normalize, truncate, or escape display content in the renderer if that decision belongs to presentation logic.
4. Do not pass raw Atlassian API DTOs directly into low-level rendering helpers.
5. Do not encode presentation intent as `[][]string` in new code when a structured presentation model would make the boundary clearer.
6. Do not add output policy by sprinkling ad hoc conditionals through commands.

## TDD expectations

New work in this area should be test-first.

### Presenter tests

Given a domain value, assert the exact presentation model produced.

These should be:

- pure unit tests
- no Cobra
- no stdout/stderr
- no network mocks unless the presenter itself requires a domain object assembled by a higher layer

### Renderer tests

Given a presentation model, assert the exact rendered output.

These should be:

- pure unit tests
- exact-string assertions where the contract matters
- separated by render style

### Command wiring tests

Commands should have lighter tests that prove:

- the correct presenter/renderer path is used
- the correct mode is chosen
- JSON/artifact behavior is preserved

The deeper contract tests belong in presenter and renderer tests, not only in Cobra command tests.

## Refactor strategy

When refactoring existing output code:

1. Identify one command/output shape.
2. Introduce a presenter for that domain object.
3. Introduce or adapt a renderer for the desired output style.
4. Lock the behavior with presenter and renderer tests.
5. Replace command-local string building with presenter/renderer wiring.
6. Repeat by output shape, not by scattered one-off string fixes.

Preferred rollout order:

1. representative detail view
2. representative table view
3. representative mutation/message output
4. composite/nested output
5. broader migration

## Repo-specific implications

### `shared/view`

`shared/view` should evolve toward rendering presentation models, not performing string surgery on arbitrary display text.

Its role should be:

- accept a structured presentation model
- render it according to policy/style
- remain pure where feasible

It should not be the place where content semantics are repaired after commands have already flattened domain data into strings.

### `jtk` and `cfl`

The same architectural rules apply to both tools.

It is acceptable for one tool to adopt a renderer/policy earlier than the other, but the shared abstractions should be designed so both can use them without reintroducing command-local formatting logic.

### Artifact contract

The artifact contract remains the source of truth for intentional output content. This architecture defines how that content should move through the system cleanly.

## Review smells

Treat these as warning signs in PR review:

- command code builds `[]string` rows directly from API response fields
- command code uses `fmt.Sprintf` only to create display values
- renderer escapes delimiters or rewrites content strings to recover structure
- output helpers require callers to pre-flatten structured data into strings
- tests only check `strings.Contains` for new output contracts
- adding a new output style requires editing many unrelated commands

## Acceptance criteria

The architecture is in a good state when:

- commands mostly orchestrate
- presenters own content decisions
- renderers own display style only
- new presenters are easy to add
- new renderers are easy to add
- most output behavior is testable without Cobra or stdout

## Non-goals

This document does not require:

- a large inheritance hierarchy
- a complicated registry system
- reflection-heavy rendering
- a framework-style abstraction layer

Prefer simple explicit types and composition.

The standard for success is not “abstract.” It is “cleanly separated, easy to test, easy to extend.”
