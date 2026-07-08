# JTK Output Specification

This document is the authoritative declaration of what `jtk` output looks like. It covers design principles, output modes, flag semantics, formatting conventions, and the exact output shape for every command in default and extended modes.

## Design principles

1. **Text is the primary format.** Stable `Key: Value` blocks and pipe-delimited tables are parseable without JSON overhead. An agent reading `Status: In Code Review` needs no less capability than one reading `{"status":"In Code Review"}` — and text wins on token density.

2. **Default output is contextually rich, not minimal.** An agent reasoning about an issue needs labels, sprint, parent, points, components — not just key/summary/status. The default output carries the semantic weight required for decision-making without flags.

3. **Administrative detail hides behind `--extended`.** Anything schema-level, rarely-used, or audit-oriented requires the flag. The test: would a developer need this monthly/yearly vs. daily? Monthly/yearly → `--extended`.

4. **JSON is reserved for round-trip payloads.** Only `automation export` emits JSON — it writes directly to stdout, bypassing the global flag system. Every other command produces text.

5. **The tool knows the instance.** A one-time `jtk init` plus daily cache refresh lets jtk resolve custom fields, users, project types, statuses, link types, and workflow transitions without per-command API calls.

## Output modes

| Mode | Flag | Purpose |
|---|---|---|
| Default | *(none)* | Contextually-rich human + agent text. Stable format. |
| Extended | `--extended` | Adds admin/schema/audit detail on top of default. Always implies `--fulltext`. |
| Identifier | `--id` | Emits only the primary identifier. Takes precedence over `--extended` and `--fulltext`. |
| Export | implicit on `automation export` | Round-trip JSON for business-rule import/export. |

## Formatting conventions

### List commands: pipe-delimited tables

- Headers in ALL_CAPS
- Separator: ` | ` (space-pipe-space)
- Empty/null values: `-`
- `--extended` adds columns; it does not replace default columns
- Sorted most-recent-first where time-ordered (sprints, etc.)

### Get / single-entity commands: header + key-value block

- First line: `ID  Name` (two spaces between)
- Attribute lines: `Key: Value   Key: Value` (three spaces between same-line pairs)
- Optional rows (Labels, Components) appear only when non-empty
- Description: blank line → `Description:` label → body text, always last

### Date formatting

- Default: `YYYY-MM-DD`
- Extended: full ISO 8601 with timezone (`2026-04-16T07:16:24+0000`)
- Missing/not-yet-set: `-`

### Text truncation

- Descriptions and comment bodies truncate in default mode
- Truncation trailer: `[truncated — use --fulltext for complete body]`
- `--fulltext` disables truncation; `--extended` implies `--fulltext`

### Mutations: post-state output

A mutation's success output mirrors the `get` output of the affected entity. The caller sees the post-state in a single call — no follow-up fetch required.

- After create: jtk always re-fetches (the Jira API returns incomplete data from the create response)
- Delete / archive / remove: confirmation line only (`Deleted MON-4820`, `Archived MON-4820`)
- `--id` on any mutation: only the affected entity's identifier

### Extended mode additions

Extended consistently adds across command types:
- Raw IDs alongside human-readable names (account IDs, component IDs, sprint IDs, type IDs)
- Full ISO 8601 timestamps instead of `YYYY-MM-DD`
- Admin fields: watchers, resolution, fix versions, status category, all non-null custom fields
- Available workflow transitions (on issue get)

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

**`--extended`:**
```
60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io
Timezone: Etc/GMT   Locale: en_US   Active: yes
Groups: 9   Application Roles: 1
```

### `users`

**`users search <query>`** — default:
```
ACCOUNT_ID | NAME | EMAIL | ACTIVE
60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io | yes
5f3a21... | Aaron Wong | aaron@monitapp.io | yes
```

**`users search <query> --extended`:**
```
ACCOUNT_ID | NAME | EMAIL | ACTIVE | TIMEZONE | LOCALE
60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io | yes | Etc/GMT | en_US
5f3a21... | Aaron Wong | aaron@monitapp.io | yes | America/New_York | en_US
```

**`users get <accountId>`** — same one-liner format as `me`. `--extended` same as `me --extended`.

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

**`projects list --extended`:**
```
KEY | TYPE | STYLE | LEAD | ISSUE_TYPES | COMPONENTS | NAME
INCIDENT | software | next-gen | - | Task, Sub-task | 0 | Incidents
JAR | software | classic | Rusty Hall | Task, Sub-task | 3 | Jira Application Requests
MON | software | classic | Rusty Hall | Epic, Kanban, SDLC | 22 | Platform Development
OFF | software | next-gen | - | Task, Sub-task | 0 | On/Offboarding
ON | software | classic | - | Epic, Kanban, SDLC | 5 | Customer Onboarding
```

**`projects get MON`** — default:
```
MON  Platform Development
Type: software   Lead: Rusty Hall   Style: classic
Issue Types: Epic, Kanban, SDLC
Components: 22   Versions: 0
```

**`projects get MON --extended`:**
```
MON  Platform Development
Type: software   Lead: Rusty Hall (60e09bae7fcd820073089249)   Style: classic
Issue Types: Epic (10000), Kanban (10026), SDLC (10025)
Components: 22
  10143 | Admin Portal
  10144 | Admin Service
  10145 | Banker Portal
  10147 | Codat Sync Service
  ... [18 more]
Versions: 0
Simplified: no   Private: no
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

**`projects types --extended`:**
```
KEY | NAME | DESCRIPTION_KEY
product_discovery | Product Discovery | jira.project.type.product_discovery.description
software | Software | jira.project.type.software.description
service_desk | Service Desk | jira.project.type.servicedesk.description.jsm
customer_service | Customer Service | jcs.project.type.customer.service.description
business | Business | jira.project.type.business.description
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

**`issues list --extended`:**
```
KEY | STATUS | TYPE | PTS | ASSIGNEE | REPORTER | SPRINT | PARENT | UPDATED | LABELS | COMPONENTS | SUMMARY
MON-4810 | In Code Review | SDLC | 5 | Aaron Wong | Aaron Wong | MON Sprint 70 | MON-3165 | 2026-04-16 | - | - | Audit and remediate accessibility issues on CapOne-specific surfaces
MON-4807 | In Code Review | SDLC | 3 | Aaron Wong | Aaron Wong | MON Sprint 70 | MON-3165 | 2026-04-16 | - | - | Make CapOne key-stack authoritative for zero-state back behavior
MON-4809 | Backlog | SDLC | - | - | Aaron Wong | - | MON-3165 | 2026-04-16 | - | - | Bump PostHog sampling to 100% for CapOne sessions
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

**`issues get MON-4810 --extended`:**
```
MON-4810  Audit and remediate accessibility issues on CapOne-specific surfaces
Status: In Code Review (category: In Progress)   Type: SDLC   Priority: Medium   Points: 5
Assignee: Aaron Wong (5f3a21...)   Reporter: Aaron Wong (5f3a21...)
Updated: 2026-04-16T07:16:24+0000   Created: 2026-04-16T07:08:49+0000
Sprint: MON Sprint 70 (id: 125, active, 2026-04-10 → 2026-04-24)
Parent: MON-3165 — 2025-26 Capital One launch (Epic)
Labels: accessibility, capone
Components: Banker Portal (10145)
Fix Versions: -
Watchers: 2 (watching: yes)
Resolution: -
customfield_10044: On Track   (Meta Status)
customfield_10050: Platform   (Team)

Transitions:
  11 | Backlog
  21 | Ready for Development
  31 | In Development
  41 | In Code Review
  51 | Ready for QA
  61 | Ready for Deployment
  71 | Deployed
  81 | Canceled

Description:
Perform an accessibility-focused review and remediation pass across CapOne-specific
frontend surfaces in packages/legacy/app, then validate the highest-risk interaction
patterns.
Primary audit artifact:
- docs/capone-accessibility-audit-2026-04-15.md
[... full body ...]
```

Extended always implies `--fulltext`. Adds: reporter with ID, raw timestamps with timezone, status category, sprint dates and ID, component IDs, watchers, resolution, fix versions (even when empty), all non-null custom fields (by name and ID), and available transitions.

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

**`issues history MON-4810 --extended`:**
```
ID | CREATED | AUTHOR | ACCOUNT_ID | FIELD | FIELD_ID | TYPE | FROM_ID | FROM | TO_ID | TO
113344 | 2026-04-16T07:05:10.000+0000 | Aaron Wong | 5f3a21... | status | status | jira | 10000 | Backlog | 10001 | Ready for Development
113345 | 2026-04-16T07:06:42.000+0000 | Aaron Wong | 5f3a21... | assignee | assignee | jira | - | - | 60e09bae7fcd820073089249 | Rian Stockbower
```

`--id` emits one changelog group ID per history group, not one ID per flattened item row. `--fields` projects fixed history columns and prepends `ID` when omitted. Extended-only columns such as `ACCOUNT_ID`, `FIELD_ID`, `TYPE`, `FROM_ID`, and `TO_ID` require `--extended`.

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

**`boards list --extended`:**
```
ID | TYPE | PROJECT | PROJECT_NAME | NAME
12 | kanban | OP | Operations | OP board
23 | scrum | MON | Platform Development | MON board
24 | kanban | ON | Customer Onboarding | ON board
25 | kanban | JAR | Jira Application Requests | JAR board
26 | simple | OFF | On/Offboarding | OFF board
27 | simple | INCIDENT | Incidents | INCIDENT board
28 | scrum | - | - | TST board
```

**`boards get 23`** — default:
```
23  MON board
Type: scrum   Project: MON (Platform Development)
```

**`boards get 23 --extended`:**
```
23  MON board
Type: scrum   Project: MON (Platform Development)
Filter: board filter for MON board (id: 10084)
Column config: Backlog, Ready for Development, In Development, In Code Review, Ready for QA, Ready for Deployment, Deployed
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

**`sprints list --board 23 --extended`:**
```
ID | STATE | START | END | COMPLETED | BOARD | GOAL | NAME
125 | active | 2026-04-10 | 2026-04-24 | - | 23 | Ship CapOne a11y fixes | MON Sprint 70
126 | future | - | - | - | 23 | - | MON Sprint 71
124 | closed | 2026-03-27 | 2026-04-10 | 2026-04-10 | 23 | Complete Q2 integration milestone | MON Sprint 69
```

**`sprints current --board 23`** — default:
```
125  MON Sprint 70
State: active   Start: 2026-04-10   End: 2026-04-24
Board: 23 (MON board)
```

**`sprints current --board 23 --extended`:**
```
125  MON Sprint 70
State: active   Start: 2026-04-10T00:00:45Z   End: 2026-04-24T23:30:00Z
Board: 23 (MON board)
Goal: Ship CapOne a11y fixes
Origin Board: 23
```

**`sprints issues 125`** — same shape as `issues list`. `--extended` matches `issues list --extended`.

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

**`comments list MON-4810 --extended`:**
```
ID | AUTHOR | CREATED | UPDATED | VISIBILITY | BODY
21242 | Aaron Wong | 2026-04-16T09:56:22+0000 | 2026-04-16T09:56:22+0000 | - | Short audit conclusion after the current code changes...
```

### `links`

**`links list MON-4818`** — default:
```
LINK_ID | TYPE | DIRECTION | ISSUE | SUMMARY
17844 | Blocker | blocks | MON-4819 | Linked issue B
17845 | Relates | relates to | MON-4700 | Fix ghost row in data table
```

**`links list MON-4818 --extended`:**
```
LINK_ID | TYPE_ID | TYPE | DIRECTION | ISSUE | STATUS | SUMMARY
17844 | 10100 | Blocker | blocks | MON-4819 | Backlog | Linked issue B
17845 | 10200 | Relates | relates to | MON-4700 | Deployed | Fix ghost row in data table
```

Extended adds the link type ID and the linked issue's current status.

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

**`remotelinks list MON-4818 --extended`:**
```
ID | RELATIONSHIP | TITLE | URL | SUMMARY
10001 | mentioned in | GitHub #456: Some issue | https://github.com/owner/repo/issues/456 | Tracks the upstream fix
10002 | - | Design doc | https://example.com/design | -
```

Extended adds the relationship label and the link summary.

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

**`transitions list MON-4810 --extended`:**
```
ID | NAME | TO_STATUS | STATUS_CATEGORY | HAS_SCREEN | CONDITIONAL | REQUIRED_FIELDS
11 | Backlog | Backlog | To Do | no | no | -
21 | Ready for Development | Ready for Development | To Do | no | no | -
31 | In Development | In Development | In Progress | no | no | -
41 | In Code Review | In Code Review | In Progress | no | no | -
51 | Ready for QA | Ready for QA | In Progress | no | no | -
61 | Ready for Deployment | Ready for Deployment | In Progress | no | no | -
71 | Deployed | Deployed | Done | no | no | -
81 | Canceled | Canceled | Done | no | no | -
```

### `attachments`

**`attachments list MON-4810`** — default:
```
ID | FILENAME | SIZE | AUTHOR | CREATED
10234 | audit-notes.md | 4.2 KB | Aaron Wong | 2026-04-16
10235 | screenshot.png | 182 KB | Aaron Wong | 2026-04-16
```

**`attachments list MON-4810 --extended`:**
```
ID | FILENAME | SIZE | BYTES | MIME_TYPE | AUTHOR | CREATED
10234 | audit-notes.md | 4.2 KB | 4301 | text/markdown | Aaron Wong | 2026-04-16T09:00:00+0000
10235 | screenshot.png | 182 KB | 186368 | image/png | Aaron Wong | 2026-04-16T09:01:12+0000
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

**`automation list --extended`:**
```
ID | STATE | LABELS | TAGS | AUTHOR | NAME
018c2840-57c1-7869-9393-11205cc87ce4 | ENABLED | onboarding | auto-create | Rian Stockbower | ON/MON: Create Onboarding Tasks
019d95ba-031c-7000-88df-134a1c924860 | DISABLED | - | - | Rusty Hall | [Archive] Old closer
```

**`automation get <id>`** — default:
```
018c2840-57c1-7869-9393-11205cc87ce4  ON/MON: Create Onboarding Tasks
State: ENABLED
Components: 27 total — 4 conditions, 23 actions
Description: Creates Tasks when a new Onboarding Epic is created
```

**`automation get <id> --extended`:**
```
018c2840-57c1-7869-9393-11205cc87ce4  ON/MON: Create Onboarding Tasks
State: ENABLED
Components: 27 total — 4 conditions, 23 actions
Description: Creates Tasks when a new Onboarding Epic is created
Labels: onboarding
Tags: auto-create
Author: Rian Stockbower
Scope: project (MON, ON)
Created: 2023-12-04   Updated: 2026-03-15
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

**`dashboards list --extended`:**
```
ID | GADGETS | OWNER | FAVOURITE | RANK | PERMISSIONS | NAME
10072 | 4 | Rian Stockbower | yes | 0 | private | Team Dashboard
10069 | 2 | Rusty Hall | no | 1 | group:developers | Incidents Overview
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

**`fields list --extended`:**
```
ID | TYPE | SEARCHABLE | NAVIGABLE | ORDERABLE | CLAUSE_NAMES | NAME
summary | string | yes | yes | yes | summary | Summary
status | status | yes | yes | no | status | Status
customfield_10035 | number | yes | yes | yes | cf[10035], Story Points | Story Points
customfield_10050 | option | yes | yes | yes | cf[10050], Team[Dropdown], Team | Team
customfield_10020 | array | yes | yes | yes | cf[10020], Sprint | Sprint
```

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
