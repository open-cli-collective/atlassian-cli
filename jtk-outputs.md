# jtk output shapes

`jtk` is text-first. It has no general output-format flag. Use `--id` for an identifier only, `--fields` to project supported fields while retaining the identifier, and `--fulltext` to disable text truncation. `automation export` is the only resource command that emits JSON (a round-trip payload, not an output mode); the control-plane `set-credential --json` envelope is the sole other exception.

The maintained command contract is [tools/jtk/internal/cmd/OUTPUT_SPEC.md](tools/jtk/internal/cmd/OUTPUT_SPEC.md). This reference shows the real shapes callers can expect.

## Shared shapes

List commands use pipe-delimited tables with ALL-CAPS headers:

```
KEY | STATUS | TYPE | PTS | ASSIGNEE | SUMMARY
MON-4810 | In Code Review | SDLC | 5 | Aaron Wong | Fix ghost row
```

Single-resource reads use a heading followed by key-value lines:

```
MON-4810  Fix ghost row
Status: In Code Review   Type: SDLC   Priority: Medium   Points: 5
Assignee: Aaron Wong   Updated: 2026-04-16
Sprint: MON Sprint 70 (active)
```

Missing values render as `-`. A paginated result has a continuation line:

```
More results available (next: eyJzdGFydEF0IjoxMH0)
```

## Output flags

### `--id`

`--id` takes precedence over other output-shape flags and emits one primary identifier per result:

```
$ jtk issues list -p MON --id
MON-4810
MON-4807
```

### `--fields <csv>`

`--fields` is a projection, not an extension of the defaults. It retains the primary identifier and replaces the default columns or detail fields with the selected supported fields:

```
$ jtk issues list -p MON --fields SUMMARY,STATUS
KEY | SUMMARY | STATUS
MON-4810 | Fix ghost row | In Code Review
```

```
$ jtk issues get MON-4810 --fields STATUS,ASSIGNEE
Key: MON-4810
Status: In Code Review
Assignee: Aaron Wong
```

### `--fulltext`

`--fulltext` preserves the selected fields and removes body/value truncation. It does not add columns.

## Identity and users

```
$ jtk me
60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io

$ jtk me --id
60e09bae7fcd820073089249

$ jtk users search rian
ACCOUNT_ID | NAME | EMAIL | ACTIVE
60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io | yes

$ jtk users get 60e09bae7fcd820073089249
60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io
```

## Projects, boards, and sprints

```
$ jtk projects list
KEY | TYPE | LEAD | NAME
MON | software | Rusty Hall | Platform Development

$ jtk projects get MON
MON  Platform Development
Type: software   Lead: Rusty Hall   Style: classic
Issue Types: Epic, Kanban, SDLC
Components: 22   Versions: 0

$ jtk boards list
ID | TYPE | PROJECT | NAME
23 | scrum | MON | MON board

$ jtk sprints list --board 23
ID | STATE | START | END | NAME
125 | active | 2026-04-10 | 2026-04-24 | MON Sprint 70

$ jtk sprints current --board 23
125  MON Sprint 70
State: active   Start: 2026-04-10   End: 2026-04-24
Board: 23 (MON board)
```

## Issues

```
$ jtk issues list -p MON
KEY | STATUS | TYPE | PTS | ASSIGNEE | SUMMARY
MON-4810 | In Code Review | SDLC | 5 | Aaron Wong | Fix ghost row

$ jtk issues search --jql 'project = MON'
KEY | STATUS | TYPE | PTS | ASSIGNEE | SUMMARY
MON-4810 | In Code Review | SDLC | 5 | Aaron Wong | Fix ghost row

$ jtk issues get MON-4810
MON-4810  Fix ghost row
Status: In Code Review   Type: SDLC   Priority: Medium   Points: 5
Assignee: Aaron Wong   Updated: 2026-04-16
Sprint: MON Sprint 70 (active)
Parent: MON-3165 — Platform work (Epic)
Labels: accessibility
Components: Banker Portal

Description:
Fix the rendering issue...
[truncated — use --fulltext for complete body]

$ jtk issues history MON-4810
ID | CREATED | AUTHOR | FIELD | FROM | TO
113344 | 2026-04-16 | Aaron Wong | status | Backlog | In Code Review

$ jtk issues fields MON-4810
FIELD_ID | NAME | TYPE | VALUE
summary | Summary | string | Fix ghost row
status | Status | status | In Code Review

$ jtk issues types --project MON
ID | NAME | SUBTASK | DESCRIPTION
10025 | SDLC | no | Task requiring Software Development Life Cycle
```

## Comments, links, and attachments

```
$ jtk comments list MON-4810
ID | AUTHOR | CREATED | BODY
21242 | Aaron Wong | 2026-04-16 | Review complete...

$ jtk links list MON-4810
LINK_ID | TYPE | DIRECTION | ISSUE | SUMMARY
17844 | Blocker | blocks | MON-4819 | Linked issue

$ jtk links types
ID | NAME | INWARD | OUTWARD
10100 | Blocker | is blocked by | blocks

$ jtk remotelinks list MON-4810
ID | TITLE | URL
10001 | Design doc | https://example.com/design

$ jtk attachments list MON-4810
ID | FILENAME | SIZE | AUTHOR | CREATED
10234 | audit-notes.md | 4.2 KB | Aaron Wong | 2026-04-16
```

## Transitions, automation, dashboards, and fields

```
$ jtk transitions list MON-4810
ID | NAME | TO_STATUS
31 | In Development | In Development

$ jtk transitions list MON-4810 --fields
ID | NAME | TO_STATUS | STATUS_CATEGORY | HAS_SCREEN | CONDITIONAL | REQUIRED_FIELDS
21 | Done | Done | Done | no | no | resolution

$ jtk automation list
ID | STATE | NAME
018c2840-57c1-7869-9393-11205cc87ce4 | ENABLED | Create Onboarding Tasks

$ jtk dashboards list
ID | GADGETS | OWNER | FAVOURITE | NAME
10072 | 4 | Rian Stockbower | yes | Team Dashboard

$ jtk dashboards gadgets list 10072
ID | POSITION | TITLE | TYPE
10122 | 0,0 | Sprint Burndown | sprint-burndown-gadget

$ jtk fields list
ID | TYPE | NAME
summary | string | Summary
customfield_10050 | option | Team
```

## Mutations

Successful create, update, assign, transition, and restore commands show the affected resource in its normal text shape. Deletes and archives are confirmation lines. Add/create commands that return a row use the corresponding list row shape.

```
$ jtk issues create -p MON --type SDLC --summary "Fix ghost row"
MON-4820  Fix ghost row
Status: Backlog   Type: SDLC   Priority: Medium   Points: -
Assignee: -   Updated: 2026-04-16
Sprint: -

$ jtk issues delete MON-4820
Deleted MON-4820

$ jtk comments add MON-4810 --body "Needs QA"
MON-4810 #21276 — Rian Stockbower, 2026-04-16
Needs QA

$ jtk links delete 17844
Deleted link 17844

$ jtk attachments add MON-4810 --file ./audit-notes.md
10236 | audit-notes.md | 4.2 KB | Rian Stockbower | 2026-04-16

$ jtk projects delete MON
Deleted project MON (moved to trash — recoverable for 60 days via `projects restore`)

$ jtk dashboards gadgets remove 10072 10124
Removed gadget 10124 from dashboard 10072
```

`--id` on a mutation emits only the affected identifier. For example, `jtk issues create ... --id` prints the new issue key.

## JSON round-trip payload

`jtk automation export <id>` is the only resource command that writes JSON. Its pretty-printed payload is intended for `jtk automation create --from-file`; `--compact` minifies that payload. The control-plane `set-credential --json` envelope is the other exception; all other commands use the text shapes above.
