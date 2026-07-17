# JTK Output Specification

This document declares `jtk` output shapes, flag semantics, and formatting conventions.

## Design principles

1. **Text is the primary format.** Stable `Key: Value` blocks and pipe-delimited tables are parseable without JSON overhead. An agent reading `Status: In Code Review` needs no less capability than one reading `{"status":"In Code Review"}` — and text wins on token density.

2. **Default output is contextually rich, not minimal.** An agent reasoning about an issue needs labels, sprint, parent, points, components — not just key/summary/status. The default output carries the semantic weight required for decision-making without flags.


3. **Additional fields are explicit.** Commands with CSV `--fields` support fetch and render only the requested supported fields while retaining the primary identifier.

4. **JSON is reserved for round-trip payloads.** `automation export` is the only resource command that emits JSON — it writes directly to stdout, bypassing the global flag system. The control-plane `set-credential --json` envelope is the sole other exception; every other command produces text.

5. **The tool knows the instance.** A one-time `jtk init` plus daily cache refresh lets jtk resolve custom fields, users, project types, statuses, link types, and workflow transitions without per-command API calls.

## Output modes

| Mode | Flag | Purpose |
|---|---|---|
| Default | *(none)* | Contextually-rich human + agent text. Stable format. |
| Field projection | `--fields <csv>` | Selects supported output fields explicitly. |
| Full text | `--fulltext` | Disables body/value truncation without changing fields. |
| Identifier | `--id` | Emits only the primary identifier and takes precedence over `--fulltext`. |
| Export | implicit on `automation export` | Round-trip JSON for business-rule import/export. |

`transitions list --fields` is the exception: it is a Boolean flag, not a CSV projection. It fetches required transition fields and adds the optional columns:

```
ID | NAME | TO_STATUS | STATUS_CATEGORY | HAS_SCREEN | CONDITIONAL | REQUIRED_FIELDS
21 | Done | Done | Done | no | no | resolution
```

## Formatting conventions

### List commands: pipe-delimited tables

- Headers in ALL_CAPS
- Separator: ` | ` (space-pipe-space)
- Empty/null values: `-`
- Sorted most-recent-first where time-ordered (sprints, etc.)

### Get / single-entity commands: header + key-value block

- First line: `ID  Name` (two spaces between)
- Attribute lines: `Key: Value   Key: Value` (three spaces between same-line pairs)
- Optional rows (Labels, Components) appear only when non-empty
- Description: blank line → `Description:` label → body text, always last

### Date formatting

- Default: `YYYY-MM-DD`
- Missing/not-yet-set: `-`

### Text truncation

- Descriptions and comment bodies truncate in default mode
- Truncation trailer: `[truncated — use --fulltext for complete body]`
- `--fulltext` disables truncation only; it never adds fields or columns

### Mutations: post-state output

A mutation's success output mirrors the `get` output of the affected entity. The caller sees the post-state in a single call — no follow-up fetch required.

- After create: jtk always re-fetches (the Jira API returns incomplete data from the create response)
- Delete / archive / remove: confirmation line only (`Deleted MON-4820`, `Archived MON-4820`)
- `--id` on any mutation: only the affected entity's identifier

### Error output

Plain prose to stderr. No structured format. Ambiguity errors list all matches. Unknown-entity errors suggest `jtk refresh <resource>`.

### Pagination

Paginated list commands append a continuation line when more results exist:

```
More results available (next: eyJzdGFydEF0IjoxMH0)
```

The token is passed back to fetch the next page:

```
$ jtk issues list -p MON --next-page-token eyJzdGFydEF0IjoxMH0
```

Absence of the continuation line signals a complete result set.

### Name/ID resolution

All entity-reference flags (`--assignee`, `--project`, `--board`, `--sprint`, link type arguments) resolve via instance cache:

- Unique match (by name, email, key, or ID) → resolve silently
- Ambiguous → fail, listing all matches with identifiers
- No match + looks like a raw ID → pass through unchanged
- No match + looks like a name → fail with suggestion to `jtk refresh <resource>`

```
$ jtk issues assign MON-4820 "John Smith"
Ambiguous user "John Smith" — 3 matches:
  5a1b2c... | John Smith | john.smith@ibm.com
  6d3e4f... | John Smith | jsmith@ibm.com
  7g8h9i... | John A. Smith | jasmith@ibm.com
Use account ID or email to disambiguate.
```

```
$ jtk issues assign MON-4820 "Zzznonexistent"
Unknown user "Zzznonexistent" — not found in cache. Try `jtk refresh users` if this user was recently added.
```

---

## Command outputs — reads

### `me`

**Default:**
```
60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io
```

**`--id`:**
```
60e09bae7fcd820073089249
```

### `users`

**`users search <query>`** — default:
```
ACCOUNT_ID | NAME | EMAIL | ACTIVE
60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io | yes
5f3a21... | Aaron Wong | aaron@monitapp.io | yes
```


### `projects`

**`projects list`** — default:
```
KEY | TYPE | LEAD | NAME
INCIDENT | software | - | Incidents
JAR | software | Rusty Hall | Jira Application Requests
MON | software | Rusty Hall | Platform Development
OFF | software | - | On/Offboarding
ON | software | - | Customer Onboarding
```

**`projects get MON`** — default:
```
MON  Platform Development
Type: software   Lead: Rusty Hall   Style: classic
Issue Types: Epic, Kanban, SDLC
Components: 22   Versions: 0
```

**`projects types`** — default:
```
KEY | NAME
product_discovery | Product Discovery
software | Software
service_desk | Service Desk
customer_service | Customer Service
business | Business
```

### `issues`

**`issues list`** — default:
```
KEY | STATUS | TYPE | PTS | ASSIGNEE | SUMMARY
MON-4810 | In Code Review | SDLC | 5 | Aaron Wong | Audit and remediate accessibility issues on CapOne-specific surfaces
MON-4807 | In Code Review | SDLC | 3 | Aaron Wong | Make CapOne key-stack authoritative for zero-state back behavior
MON-4809 | Backlog | SDLC | - | - | Bump PostHog sampling to 100% for CapOne sessions
More results available (next: eyJzdGFydEF0IjoxMH0)
```

**`issues search <jql>`** — same output shape as `issues list`.

**`issues get MON-4810`** — default:
```
MON-4810  Audit and remediate accessibility issues on CapOne-specific surfaces
Status: In Code Review   Type: SDLC   Priority: Medium   Points: 5
Assignee: Aaron Wong   Updated: 2026-04-16
Sprint: MON Sprint 70 (active)
Parent: MON-3165 — 2025-26 Capital One launch (Epic)
Labels: accessibility, capone
Components: Banker Portal

Description:
Perform an accessibility-focused review and remediation pass across CapOne-specific
frontend surfaces in packages/legacy/app, then validate the highest-risk interaction
patterns...
[truncated — use --fulltext for complete body]
```

Labels/Components rows appear only when non-empty. Custom fields selected during `jtk init` (e.g., Team) appear when non-null.

**`issues history MON-4810`** — default:
```
ID | CREATED | AUTHOR | FIELD | FROM | TO
113344 | 2026-04-16 | Aaron Wong | status | Backlog | Ready for Development
113345 | 2026-04-16 | Aaron Wong | assignee | - | Rian Stockbower
113346 | 2026-04-17 | Rian Stockbower | summary | Initial placeholder | Audit and remediate accessibility issues on CapOne-specific surfaces
More results available (next: 50)
```

Rows are chronological in Jira's changelog order. Each row is one changed field item. The `ID` is the changelog group ID and may repeat when Jira groups multiple field changes in one history entry.

**`issues history MON-4810 --id`:**
```
113344
113345
113346
More results available (next: 50)
```


**`issues fields MON-4810`** — default:
```
FIELD_ID | NAME | TYPE | VALUE
summary | Summary | string | Audit and remediate accessibility issues on CapOne-specific surfaces
status | Status | status | In Code Review
assignee | Assignee | user | Aaron Wong
customfield_10035 | Story Points | number | 5
customfield_10050 | Team | option | Platform
...
```

**`issues fields MON-4810 --custom-fields`:** filters to `customfield_*` rows only.

**`issues types MON`** — default:
```
ID | NAME | SUBTASK | DESCRIPTION
10000 | Epic | no | A big user story that needs to be broken down.
10025 | SDLC | no | Task requiring Software Development Life Cycle
10026 | Kanban | no | Task following Kanban Flow
```

**`issues field-options MON-4970 customfield_10050`** — default:
```
ID | VALUE | DISABLED
20001 | Platform | no
20002 | Integration | no
20003 | Frontend | no
```

### `boards`

**`boards list`** — default:
```
ID | TYPE | PROJECT | NAME
12 | kanban | OP | OP board
23 | scrum | MON | MON board
24 | kanban | ON | ON board
25 | kanban | JAR | JAR board
26 | simple | OFF | OFF board
27 | simple | INCIDENT | INCIDENT board
28 | scrum | - | TST board
```

**`boards get 23`** — default:
```
23  MON board
Type: scrum   Project: MON (Platform Development)
```

### `sprints`

**`sprints list --board 23`** — default:
```
ID | STATE | START | END | NAME
125 | active | 2026-04-10 | 2026-04-24 | MON Sprint 70
126 | future | - | - | MON Sprint 71
124 | closed | 2026-03-27 | 2026-04-10 | MON Sprint 69
123 | closed | 2026-03-13 | 2026-03-27 | MON Sprint 68
```

Sorted most-recent-first. Dates as `YYYY-MM-DD`.

**`sprints current --board 23`** — default:
```
125  MON Sprint 70
State: active   Start: 2026-04-10   End: 2026-04-24
Board: 23 (MON board)
```


### `comments`

**`comments list MON-4810`** — default:
```
ID | AUTHOR | CREATED | BODY
21242 | Aaron Wong | 2026-04-16 | Short audit conclusion after the current code changes: The major source-level accessibility findings on CapOne-specific surfaces appear to be addressed or materially improv...
```

**`comments list MON-4810 --fulltext`:** one block per comment:
```
ID: 21242
Author: Aaron Wong
Created: 2026-04-16
Body:
Short audit conclusion after the current code changes:
The major source-level accessibility findings on CapOne-specific surfaces
appear to be addressed or materially improved:
- loading / redirect states now expose accessible status messaging
- the unsupported-package modal now exposes both title and description correctly
...
```

### `links`

**`links list MON-4818`** — default:
```
LINK_ID | TYPE | DIRECTION | ISSUE | SUMMARY
17844 | Blocker | blocks | MON-4819 | Linked issue B
17845 | Relates | relates to | MON-4700 | Fix ghost row in data table
```


**`links types`** — default:
```
ID | NAME | INWARD | OUTWARD
10100 | Blocker | is blocked by | blocks
10200 | Relates | relates to | relates to
10300 | Cloners | is cloned by | clones
10400 | Duplicate | is duplicated by | duplicates
```

Cached during init/refresh. `links create` accepts the type name ("Blocker"), the outward verb ("blocks"), or the inward verb ("is blocked by").

### `remotelinks`

Remote (web) links are external URLs attached to an issue and shown in the Jira links sidebar — distinct from `links`, which connect two Jira issues.

**`remotelinks list MON-4818`** — default:
```
ID | TITLE | URL
10001 | GitHub #456: Some issue | https://github.com/owner/repo/issues/456
10002 | Design doc | https://example.com/design
```


**`remotelinks add MON-4818 --url ... --title ...`** — post-state detail:
```
Added remote link 10001 to MON-4818
ID: 10001
Issue: MON-4818
Title: GitHub #456: Some issue
URL: https://github.com/owner/repo/issues/456
```

`--title` defaults to the URL when omitted. `--id` emits only the new link ID.

**`remotelinks delete MON-4818 10001`** — confirmation line only:
```
Deleted remote link 10001 from MON-4818
```

### `transitions`

**`transitions list MON-4810`** — default:
```
ID | NAME | TO_STATUS
11 | Backlog | Backlog
21 | Ready for Development | Ready for Development
31 | In Development | In Development
41 | In Code Review | In Code Review
51 | Ready for QA | Ready for QA
61 | Ready for Deployment | Ready for Deployment
71 | Deployed | Deployed
81 | Canceled | Canceled
```

### `attachments`

**`attachments list MON-4810`** — default:
```
ID | FILENAME | SIZE | AUTHOR | CREATED
10234 | audit-notes.md | 4.2 KB | Aaron Wong | 2026-04-16
10235 | screenshot.png | 182 KB | Aaron Wong | 2026-04-16
```

**`attachments get 10234 --output ./audit-notes.md`:**
```
Downloaded 10234 → ./audit-notes.md (4.2 KB)
```

### `automation`

**`automation list`** — default:
```
ID | STATE | NAME
018c2840-57c1-7869-9393-11205cc87ce4 | ENABLED | ON/MON: Create Onboarding Tasks
019d95ba-031c-7000-88df-134a1c924860 | DISABLED | [Archive] Old closer
```

**`automation get <id>`** — default:
```
018c2840-57c1-7869-9393-11205cc87ce4  ON/MON: Create Onboarding Tasks
State: ENABLED
Components: 27 total — 4 conditions, 23 actions
Description: Creates Tasks when a new Onboarding Epic is created
```

**`automation get <id> --show-components`:** dumps the full component tree as indented text (trigger → conditions → actions).

**`automation export <id>`:** emits the rule definition as pretty-printed JSON to stdout. This is the round-trip format consumed by `automation create --from-file`. `--compact` minifies. This command bypasses the global flag system.

### `dashboards`

**`dashboards list`** — default:
```
ID | GADGETS | OWNER | FAVOURITE | NAME
10072 | 4 | Rian Stockbower | yes | Team Dashboard
10069 | 2 | Rusty Hall | no | Incidents Overview
```

**`dashboards gadgets list 10072`** — default:
```
ID | POSITION | TITLE | TYPE
10122 | 0,0 | Sprint Burndown | sprint-burndown-gadget
10123 | 0,1 | Created vs Resolved | created-vs-resolved-gadget
```

### `fields`

**`fields list`** — default:
```
ID | TYPE | NAME
summary | string | Summary
status | status | Status
customfield_10035 | number | Story Points
customfield_10050 | option | Team
customfield_10020 | array | Sprint
```

**`fields list --custom-fields`:** filters to `customfield_*` rows only.

**`fields list --name story`:** substring filter on name.

**`fields show customfield_10050`** — flat denormalized view:
```
CONTEXT_ID | CONTEXT | PROJECTS | OPTION_ID | OPTION_VALUE
10100 | Default Context | (global) | 20001 | Platform
10100 | Default Context | (global) | 20002 | Integration
10100 | Default Context | (global) | 20003 | Frontend
10101 | MON Project Context | MON | 20010 | CapOne
10101 | MON Project Context | MON | 20011 | Acme
10102 | ON Project Context | ON | - | -
```

Empty contexts render with `- | -` so the context is discoverable.

---

## Command outputs — mutations

**General rule: a mutation's success output mirrors the `get` output of the affected entity.** The caller sees the post-state in a single call without a follow-up fetch. `--id` on any mutation emits only the affected entity's identifier.

### `issues create / update / assign / transition / archive`

```
$ jtk issues create -p MON --type SDLC --summary "Fix ghost row"
MON-4820  Fix ghost row
Status: Backlog   Type: SDLC   Priority: Medium   Points: -
Assignee: -   Updated: 2026-04-16
Sprint: -
```

```
$ jtk issues create -p MON --type SDLC --summary "Fix ghost row" --id
MON-4820
```

```
$ jtk issues assign MON-4820 "Rian Stockbower"
MON-4820  Fix ghost row
Status: Backlog   Type: SDLC   Priority: Medium   Points: -
Assignee: Rian Stockbower   Updated: 2026-04-16
Sprint: -
```

```
$ jtk issues transition MON-4820 31
MON-4820  Fix ghost row
Status: In Development   Type: SDLC   Priority: Medium   Points: -
Assignee: Rian Stockbower   Updated: 2026-04-16
Sprint: -
```

```
$ jtk issues archive MON-4820
Archived MON-4820
```

### `issues delete`

```
$ jtk issues delete MON-4820
Deleted MON-4820
```

Multi-delete: one line per deleted issue.

### `comments add / delete`

```
$ jtk comments add MON-4810 "Noting that this needs QA review on Safari 16."
MON-4810 #21276 — Rian Stockbower, 2026-04-16
Noting that this needs QA review on Safari 16.
```

```
$ jtk comments add MON-4810 "..." --id
21276
```

```
$ jtk comments delete MON-4810 21276
Deleted comment 21276 from MON-4810
```

### `links create / delete`

```
$ jtk links create MON-4819 MON-4818 --type Blocker
LINK_ID | TYPE | DIRECTION | ISSUE | SUMMARY
17844 | Blocker | blocks | MON-4818 | Linked issue B
```

The first issue is the user-facing subject and the second is the target. `--type` accepts the link type name ("Blocker"), outward verb ("blocks"), or inward verb ("is blocked by"). After create, jtk re-queries to recover the link ID (the Jira API does not return it from the create call).

```
$ jtk links delete 17844
Deleted link 17844
```

### `projects create / update / delete / restore`

```
$ jtk projects create --key GFIL --name "Gap Fill" --type software --lead rian@monitapp.io
GFIL  Gap Fill
Type: software   Lead: Rian Stockbower   Style: next-gen
Issue Types: Task, Sub-task
Components: 0   Versions: 0
```

After create, jtk re-fetches to get the fully-populated project.

```
$ jtk projects delete GFIL
Deleted project GFIL (moved to trash — recoverable for 60 days via `projects restore`)
```

```
$ jtk projects restore GFIL
GFIL  Gap Fill Automation
Type: software   Lead: Rian Stockbower   Style: next-gen
Issue Types: Task, Sub-task
Components: 0   Versions: 0
```

### `sprints add`

```
$ jtk sprints add MON-4820 125
MON-4820 added to MON Sprint 70 (active, ends 2026-04-24)
```

Accepts sprint ID or name (resolved via cache).

### `attachments add / delete`

```
$ jtk attachments add MON-4810 ./audit-notes.md
10236 | audit-notes.md | 4.2 KB | Rian Stockbower | 2026-04-16
```

```
$ jtk attachments delete 10236
Deleted attachment 10236
```

### `dashboards create / delete`

```
$ jtk dashboards create --name "Release Watch" --share private
ID | GADGETS | OWNER | FAVOURITE | NAME
10073 | 0 | Rian Stockbower | yes | Release Watch
```

Matches `dashboards list` row format — the mutation output is a single row in the same shape.

```
$ jtk dashboards delete 10073
Deleted dashboard 10073
```

### `automation create / enable / disable / update / delete`

```
$ jtk automation create --from-file rule.json
019e1234-abcd-7000-8888-112233445566  [Test] My Rule
State: ENABLED
Components: 5 total — 1 condition, 4 actions
```

```
$ jtk automation disable 019e1234-abcd-7000-8888-112233445566
019e1234-abcd-7000-8888-112233445566  [Test] My Rule
State: DISABLED
Components: 5 total — 1 condition, 4 actions
```

```
$ jtk automation update 019e1234-abcd-7000-8888-112233445566 --file rule.json
019e1234-abcd-7000-8888-112233445566  [Test] My Rule
State: ENABLED
Components: 5 total — 1 condition, 4 actions
```

Post-state mirrors `automation get`. On success `update` also writes an advisory to **stderr** — automation reads are eventually consistent, so a re-export/get run immediately afterward may show the prior definition even though the update applied. The advisory is suppressed under `--id`, which emits only the rule UUID.

```
$ jtk automation delete 019e1234-abcd-7000-8888-112233445566
Deleted automation 019e1234-abcd-7000-8888-112233445566
```

### `fields create / delete / restore`

```
$ jtk fields create --name "Team" --type select
customfield_10223 | option | Team
```

Matches `fields list` row format.

```
$ jtk fields delete customfield_10223
Deleted field customfield_10223 (moved to trash — use `fields restore` to recover)
```

```
$ jtk fields restore customfield_10223
customfield_10223 | option | Team
```

### `fields contexts create / delete`

```
$ jtk fields contexts create customfield_10050 --name "MON Context" --projects MON
10401 | MON Context | MON
```

```
$ jtk fields contexts delete customfield_10050 10401
Deleted context 10401 from customfield_10050
```

### `fields options add / update / delete`

```
$ jtk fields options add customfield_10050 --context 10100 --value "DevOps"
20004 | DevOps | no
```

```
$ jtk fields options update customfield_10050 --context 10100 --option 20004 --value "Platform Engineering"
20004 | Platform Engineering | no
```

```
$ jtk fields options delete customfield_10050 --context 10100 --option 20004
Deleted option 20004 from context 10100
```

### `dashboards gadgets add / remove`

```
$ jtk dashboards gadgets add 10072 --type sprint-burndown-gadget --position 1,0
10124 | 1,0 | Sprint Burndown | sprint-burndown-gadget
```

```
$ jtk dashboards gadgets remove 10072 10124
Removed gadget 10124 from dashboard 10072
```

---

## Command aliases

All aliases produce identical output to their canonical form.

### Top-level aliases

| Alias | Canonical |
|---|---|
| `jtk issue`, `jtk i` | `jtk issues` |
| `jtk project`, `jtk proj`, `jtk p` | `jtk projects` |
| `jtk board`, `jtk b` | `jtk boards` |
| `jtk sprint`, `jtk sp` | `jtk sprints` |
| `jtk user`, `jtk u` | `jtk users` |
| `jtk auto` | `jtk automation` |
| `jtk transition`, `jtk tr` | `jtk transitions` |
| `jtk comment`, `jtk c` | `jtk comments` |
| `jtk attachment`, `jtk att` | `jtk attachments` |
| `jtk field`, `jtk f` | `jtk fields` |
| `jtk link`, `jtk l` | `jtk links` |
| `jtk dash`, `jtk dashboard` | `jtk dashboards` |

### Subcommand aliases

| Alias | Canonical |
|---|---|
| `ls` | `list` |
| `rm` | `delete` |
| `ctx`, `context` | `contexts` |
| `opt`, `option` | `options` |
