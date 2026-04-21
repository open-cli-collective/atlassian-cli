# SpecToTickets Workflow

Decompose a specification, PRD, design doc, plan, or other structured content into a proposed set of Jira tickets. This workflow produces a structured breakdown for user review and composes the full content that will be written to Jira. It does **not** create tickets itself — once the user approves the breakdown, continue into ManageIssueSet for execution.

For multi-issue operations where the user already specifies the tickets directly ("create these 3 issues"), skip this workflow and go straight to ManageIssueSet.

## When to Use This Workflow

Route here when:

- User provides a specification, PRD, design doc, plan, or similar structured content AND asks to turn it into tickets
- User says "decompose this into tickets," "turn this into a ticket set," "break this spec into tasks," "file tickets for this," or similar
- User provides a long/structured document with an implicit or explicit "what tickets should I file?"

If the user directly provides a concrete list of tickets to create (no spec — just "create these 3 issues"), route straight to ManageIssueSet. This workflow is for cases where the breakdown itself is a judgment call the agent is making on behalf of the user.

## Terminology

Consistent word choices used throughout this workflow:

- **Ticket** — one Jira issue. Used interchangeably with "issue" when referring to the external Jira object. A **ticket section** is one `## Ticket: <jira_key>` block inside the workfile.
- **Breakdown** — the overall set of proposed tickets derived from the spec. The thing this workflow produces.
- **Proposal** — the compact user-facing view of the breakdown shown in stage 3b for review. A read-only projection of the workfile.
- **Workfile** — the authoritative persistent document at the agent's working/scratch path. Holds the full breakdown with complete ticket content.
- **Description** — the full multi-paragraph Jira description body, written into the workfile between `<!-- SPECTOTICKETS_DESCRIPTION_START -->` and `<!-- SPECTOTICKETS_DESCRIPTION_END -->` markers, later written to Jira as the issue's `description` field.
- **short_description** — a distinct workfile field (1-3 sentence brief) used in the proposal view. Not the full Description.
- **Temp ID** — a `SpecToTicketWorkflowTemp-N` placeholder for a ticket that has not yet been created in Jira. Replaced with the real Jira key during execution.

## Stages

### 1. Read & Understand the Spec

Read the provided content carefully. Treat its structure — headings, sections, bullets, numbered lists, emphasis — as a hint about natural work units, but don't mechanically map structure to tickets.

Identify:

- **Top-level deliverable.** Often maps to a parent (Epic, Initiative, PRD, or similar — project-dependent).

- **Units of work, and the right granularity for each.** A single conceptual unit from the spec might be any of:
  - A standalone ticket
  - A parent ticket that needs sub-tasks (because the work is too large for one ticket, or because different people will own different parts)
  - Multiple sibling tickets at the same level (when the work has parallel, independent threads of a similar shape)
  - A parent with both sub-tasks AND sibling-level tickets linked by blocks/relates-to

  Choose the granularity deliberately. When in doubt, err on fewer, larger tickets — and explicitly call out where you think splitting might be useful so the user can weigh in during confirmation.

- **Relationships between tickets.** Jira's link-type model stores one link per pair with a canonical NAME and two directional phrasings (OUTWARD from the source, INWARD from the target). For example, the `Blocks` link type has NAME `Blocks`, OUTWARD phrase `blocks`, INWARD phrase `is blocked by`. When composing a link in the workfile, the agent encodes the source-side perspective: the link entry lives in the source ticket's section, and the `type:` value is the canonical NAME. Common link types (by NAME / OUTWARD / INWARD):
  - **Parent/child hierarchy** (sub-tasks, epic-to-story) — this is NOT a link type; it's created via `--parent KEY` on the child at ticket creation, not via `jtk links create`
  - **Blocks** — OUTWARD `blocks`, INWARD `is blocked by`; A must complete before B can start; most common dependency type
  - **Relates** — OUTWARD and INWARD both `relates to`; informal association, no ordering
  - **Duplicate** — OUTWARD `duplicates`, INWARD `is duplicated by`; same work captured twice
  - **Problem/Incident** — OUTWARD `causes`, INWARD `is caused by`; bug-causation links
  - **Cloners** — OUTWARD `clones`, INWARD `is cloned by`; template relationship
  - Project-specific custom link types may also exist.

  **Discover the exact set** and verify NAME values for your instance by running `jtk links types`. The command output shows four columns: ID, NAME, OUTWARD, INWARD. Use the NAME column value for the workfile `type:` field. Use the OUTWARD phrasing when rendering the link to the user in the proposal view.

  Spec language to watch for: "depends on," "requires," "after," "in parallel with," "related to," "caused by," "implements," "fixes," "supersedes," etc. These often map to link types — but **ask the user when direction or type is ambiguous** ("A blocks B" and "B blocks A" are opposite encodings; pick wrong and Jira shows the opposite relationship). Be explicit in the confirmation prompt about which ticket is the source and which is the target.

- **Sequential vs parallel work.** If the spec describes a sequence ("first design, then build, then test"), that's likely a chain of `Blocks` links between siblings — each earlier ticket blocks the next. If it describes parallel threads, they might be siblings with no links, or siblings all linked back to a coordination parent.

- **Implicit fields.** Priorities signaled by words like "urgent," "critical," "must-have," "nice-to-have"; owners mentioned by name; rough sizing hints ("small," "big effort").

- **A flat set of peer siblings with no shared top-level parent.** When a spec is a grab-bag of loosely-related work items that don't need a coordinating parent, a flat set of top-level tickets is fine. Each ticket's `parent` is `none` — the Project itself is the effective parent. Don't invent a coordinator ticket just to satisfy a mental model of hierarchy.

- **Ambiguities.** Things that could map to multiple ticket shapes, where the spec is unclear about work boundaries, or where the relationship between units isn't explicit. Identify them during this stage. They'll be written to the workfile's `ambiguities` field when you compose the workfile in stage 3a (the workfile is the source of truth — don't rely on conversation context to hold them). They're surfaced to the user in stage 3b's proposal and resolved during stage 4 confirmation.

### 2. Discovery (Up-Front and On-Demand)

#### Step 1 — Establish the target project

Every subsequent step in this workflow requires a known target project — type discovery, priority discovery, ticket composition, and ticket creation all depend on it.

If the user has clearly specified a project (by key or by name) in their request, use it.

**If the project is not specified, or is ambiguous, ask the user before proceeding.** Do not guess from spec content — spec titles and section content don't reliably map to Jira project keys, and decomposing against a wrong project wastes effort at every subsequent stage.

Use `jtk projects list` if the user wants to see which projects they have access to.

Once the project is established, it will be recorded in the workfile's batch-metadata `project:` field during stage 3a composition. From here through the rest of Stage 2, "the project" refers to this established key.

#### Step 2 — Issue types (up-front)

Run:

```
jtk issues types --project KEY
```

The breakdown proposal must use types that actually exist in the target project. The `SUBTASK` column identifies sub-task types (which require `--parent KEY` at creation time). Record these values in the workfile's batch-metadata `discovered_types:` field during stage 3a composition (including the `SUBTASK: yes` annotation where applicable).

#### Step 3 — Priority scheme (conditionally up-front)

If the user intends to set priorities on any ticket in the batch, run priority discovery now. If priorities aren't needed (every ticket will have `priority: none`), skip this step and set `discovered_priorities: none` in the workfile header during stage 3a.

**Two-step command pattern:**

1. Find any existing issue in the target project to use as the `--issue` reference:

   ```
   jtk issues list --project KEY --max 1 --id
   ```

   This returns a single issue key (e.g., `ANP-100`). Any existing issue works as the reference.

2. Use that key for the priority discovery:

   ```
   jtk issues field-options priority --issue ANP-100
   ```

   The output lists the valid priority values for this project's priority scheme. Record these in the workfile's batch-metadata `discovered_priorities:` field during stage 3a composition.

**If step 1 returns no results** (the project has no existing issues at all), `jtk issues field-options priority --issue` cannot run. Inform the user: "This project has no existing issues — priority discovery needs at least one issue to run against. Options: (a) create a bootstrap issue in Jira first (any trivial placeholder), (b) skip priorities entirely for this batch (set `discovered_priorities: none` and every ticket's `priority: none`), or (c) cancel and come back after issues exist." Wait for the user's choice before proceeding. Do not silently fabricate priority values or skip the discovery step without user approval.

**Known limitation — project-scoped assumption.** This workflow treats the priority scheme as project-scoped based on a single `jtk issues field-options priority --issue` query against one existing issue in the project. Most Jira instances have one priority scheme per project, so this is correct. Some instances configure per-type priority schemes (different priority lists for Bug vs Task vs Epic, etc.). Stage 4a validation cannot detect this mismatch — it only verifies each ticket's `priority` is in the single `discovered_priorities` list. If a priority is valid for the reference issue's type but not for another ticket's type, the mismatch surfaces at Phase 1 execution, not at validation. See Stage 5's priority-mismatch handling for what happens then.

#### Step 4 — On-demand discovery (during ticket composition)

The following discoveries are declared here but run during stage 3a composition — when the specific value is needed, not up front. Declaring them here keeps all project-discovery guidance in one section.

- **Users / assignees** — when the spec names a specific person or the user wants one assigned, run `jtk users search "NAME" --max 10` (explicit cap so the result-count check below is predictable; don't rely on CLI defaults). Inspect the result count:
  - **No matches:** set `assignee: none` in the workfile and surface the name as an ambiguity for stage 3b's proposal — ask the user whether they meant someone else or want to leave unassigned.
  - **Multiple matches (2-9):** set `assignee: none`, surface the candidate list during stage 4 confirmation, let the user pick which account ID is correct.
  - **Exactly 10 matches:** this is the cap — the intended user may well be past it. Treat as ambiguous: set `assignee: none`, surface to the user during stage 4, and ask them either to refine the search (fuller name, email domain, exact email) or to pick from the 10 returned.
  - **Single match with clear fit** (full name exact match, email match, etc.): use that account ID. If the match is plausible but you have doubt (common name, loose match against a first name only), treat it as ambiguous and surface for user confirmation rather than silently committing.

  Never use `jtk users search --max 1 --id` in this workflow — it suppresses the ambiguity signal and commits to the first match without evidence of whether there were others.

- **Sprint IDs** — if the user wants tickets added to a sprint, run `jtk sprints current --board BOARD_ID` for the active sprint, or `jtk sprints list --board BOARD_ID` to list all sprints for that board. The user may provide the sprint ID directly; ask if neither the board ID nor the sprint ID is clear.

- **Link types** — if the spec implies a link type beyond the common canonical NAMEs (`Blocks`, `Relates`, `Duplicate`, `Problem/Incident`, `Cloners`), or if the instance may use custom link types, run `jtk links types` to confirm the exact NAME column value for your instance. Use that NAME verbatim in the workfile `type:` field. Ask the user if the match is ambiguous.

### 3. Compose Full Ticket Content, Then Present a Proposal

This stage has two parts: composing the authoritative ticket content to a workfile, and presenting a compact summary to the user for review.

#### 3a. Compose full ticket content to a workfile

Before presenting anything to the user, write the full breakdown to a structured workfile. This file is the source of truth for the proposed tickets — the agent's conversation context may get compacted or truncated, but the file persists.

**Workfile location:** the agent's working/scratch directory. If your harness provides a dedicated scratch area for operational artifacts that persists across conversation turns, use it. Otherwise, before falling back to the system temp directory (`$TMPDIR` or `/tmp` on Unix-like systems, `%TEMP%` on Windows), **warn the user and ask them to choose a location.** Use wording like: "I'd like to write the breakdown workfile to the system temp directory (`/tmp` or equivalent). Many systems auto-clean temp directories (systemd-tmpfiles, macOS periodic cleanup). If this workflow spans multiple sessions or days — e.g., you want to come back later to finish the breakdown or review before execution — the workfile may be deleted before the process completes. Options: (a) proceed with system temp and accept the risk, (b) use a more durable path such as `~/.cache/jira-spec-to-tickets/` or a project directory you specify. Which would you prefer?" Only compose the workfile once the path is agreed. An example filename: `breakdown-YYYYMMDD-HHMMSS-<short-name>.md`. The file must persist across conversation turns and be editable by the agent throughout the workflow.

**File format:** pure Markdown. The file opens with a top section containing batch-level metadata, followed by one `## Ticket: <jira_key>` section per proposed ticket — the heading literally contains that ticket's `jira_key` field value (e.g., `## Ticket: SpecToTicketWorkflowTemp-7` pre-execution, `## Ticket: ANP-501` post-execution). All metadata fields — at the batch level and the ticket level — use bold-labeled bullet lines (`- **field_name:** value`). The `links` field uses nested Markdown bullets (not YAML) with a fixed shape: the `- **links:**` bullet is followed by one entry per link, where each entry is a two-space-indented `  - type: <type>` bullet with a four-space-indented `    target: <target>` continuation line on the next line. The nested-bullet form is structural, not a YAML subset. See Example 2 below for the concrete layout.

Each ticket section contains metadata bullets followed by a `### Description` heading, then the full ticket body enclosed in distinctive HTML-comment marker pairs:

```
### Description

<!-- SPECTOTICKETS_DESCRIPTION_START -->

[the full description body goes here, verbatim — any Markdown is allowed
inside: code blocks, nested headings (## Context, ## Schema, etc.),
tables, horizontal rules, bullet lists, whatever the ticket needs]

<!-- SPECTOTICKETS_DESCRIPTION_END -->
```

The comment markers delimit the description body unambiguously, even if the body contains `##`/`###` headings, horizontal rules, or other Markdown constructs that would otherwise confuse section-boundary parsing. Agents reading the workfile extract everything between `<!-- SPECTOTICKETS_DESCRIPTION_START -->` and `<!-- SPECTOTICKETS_DESCRIPTION_END -->` verbatim as the ticket's `description` field.

Ticket sections are separated by a `---` horizontal rule on its own line between the previous ticket's `<!-- SPECTOTICKETS_DESCRIPTION_END -->` and the next ticket's `## Ticket: <jira_key>` heading. The seam between two tickets looks like this:

```
<!-- SPECTOTICKETS_DESCRIPTION_END -->

---

## Ticket: SpecToTicketWorkflowTemp-N
```

The `---` sits outside description bodies (which are protected by the comment markers), so it can't collide with description content.

**Batch metadata at file top:**

```
# Breakdown — <spec title>

- **spec_title:** Auth Rework PRD
- **spec_source:** Confluence page 12345 (https://...)
- **project:** ANP
- **created:** 2026-04-20T15:30:00-07:00
- **updated:** 2026-04-20T15:45:00-07:00
- **status:** proposed
- **target_sprint:** 597
- **discovered_types:**
  - Task
  - Bug
  - Story
  - Epic
  - Sub-task (SUBTASK: yes, requires --parent)
- **discovered_priorities:**
  - Highest
  - High
  - Medium
  - Low
  - Lowest
- **ambiguities:**
  - Should §2.1 be one ticket or split into schema + lifecycle?
  - "Alice" mentioned as owner — which Alice? (Multiple users named Alice matched `jtk users search`)
  - Unclear if §3.4 implies a dependency on §2.1 — "after" could mean sequential or independent
```

Field notes:

- **`status`** starts as `proposed`, becomes `revised` when the user requests changes during confirmation, and becomes `confirmed` once the user gives final go-ahead. The `updated` timestamp is refreshed on every edit.
- **`target_sprint`** is `none` if no sprint is requested.
- **`discovered_types`** is populated at the end of stage 2 from `jtk issues types --project KEY`. Mark sub-task types explicitly (e.g., "Sub-task (SUBTASK: yes, requires --parent)") so validation and composition can distinguish them. Stage 4a's type-related validation checks reference this list — it's the authoritative record of what types are valid for this batch, independent of conversation context.
- **`discovered_priorities`** is populated at the end of stage 2 from `jtk issues field-options priority --issue EXISTING_KEY`, or `none` if the user doesn't plan to set priorities. Stage 4a's priority validation check references this list.
- **`ambiguities`** is a list of open questions identified during stage 1 that the user needs to resolve. Empty (`none`) if none were found. The agent writes these here during stage 3a composition; stage 3b surfaces them to the user; stage 4 confirmation resolves them (and the agent clears or updates the list as each is resolved). If a revision during stage 4 introduces a new ambiguity, add it here and re-surface.

**Temp IDs:** each ticket has a `jira_key` field initially set to `SpecToTicketWorkflowTemp-N` where N is a batch-local integer. References to other tickets in this batch — in `parent`, in `links[].target`, in description body prose, or in `short_description` — use the same temp ID. During execution (stage 5, inside ManageIssueSet), each temp ID gets replaced file-wide with the real Jira key as that ticket is created.

**Temp ID numbering:** temp IDs are unique integers but do not need to be sequential or gap-free. Initial numbering typically goes 1, 2, 3, ... monotonically as tickets are proposed. When a ticket is removed or merged during revision (stage 4), leave its temp ID retired — do not reuse the number, do not renumber the remaining tickets. Gaps are fine.

**Heading-and-`jira_key` coupling:** the section heading `## Ticket: <jira_key>` literally contains the full `jira_key` value. There is no separate "numeric suffix" concept — the heading and the `jira_key` field always match by construction. If the `jira_key` changes (during Phase 1's find-and-replace when a temp ID becomes a real Jira key, or during manual edits), the heading must change in lockstep. Phase 1's file-wide find-and-replace handles this naturally: replacing `SpecToTicketWorkflowTemp-7` with `ANP-501` rewrites both the `jira_key` field and the heading `## Ticket: SpecToTicketWorkflowTemp-7` → `## Ticket: ANP-501` in one pass. For manual edits (stage 4 revisions), remember to update both. Validation check #17 enforces the coupling.

**Ticket ordering within the workfile:** what matters is file position, not temp ID numbering. Tickets whose `parent` is a temp ID (referring to another ticket in this batch) must appear later in the file than the ticket they reference. Phase 1 execution reads the file top-to-bottom and creates tickets as it encounters them — a parent must already have been created by the time its child is processed. Numbering is independent: `SpecToTicketWorkflowTemp-5` may appear before `SpecToTicketWorkflowTemp-3` in the file — the numeric suffix is just an identifier, not an ordering signal. Tickets whose `parent` is `none` or a real Jira key (existing issue) have no in-batch dependency and can appear anywhere in the file relative to other tickets.

**All fields are listed on every ticket**, even when empty. Use `none` for empty values. This keeps parsing uniform and avoids ambiguity about whether a field was intentionally left blank or forgotten.

**Single-project scope:** a workfile represents a single-project batch. The top-level `project:` field applies to every ticket in the file. If the user's intent spans multiple projects, work with them to either run SpecToTickets separately per project, or adapt this workflow's handling for that specific case (e.g., per-ticket `project:` overrides, coordinated temp-ID namespaces, etc.). Do not silently widen scope within a single workfile.

**Required fields per ticket** (always present; the order shown is recommended for readability but not enforced by the validation pass):

- `jira_key` — `SpecToTicketWorkflowTemp-N` pre-execution, replaced with real Jira key during execution
- `type` — from the discovered types list (stage 2)
- `summary` — concise, under ~150 characters, what will be the Jira summary field verbatim
- `priority` — from the discovered priority scheme, or `none`
- `assignee` — `none`, `me`, an email address, or a discovered account ID. The literal string `me` is supported natively by `jtk issues create` and `jtk issues update` per the CLI's own help text (`Assignee (account ID, email, or "me")`); `jtk` resolves it to the authenticated user's account ID at execution time, so the workflow doesn't need to substitute anything
- `parent` — `none`, a temp ID (another ticket in this batch), or an existing Jira key
- `labels` — comma-space-separated list (e.g., `auth-rework, q2-goal, backend`) or `none`. Individual labels should not contain spaces; use hyphens or underscores per project convention.
- `links` — a structured list of `type` + `target` pairs, or `none`. The `type:` value is the canonical NAME from Jira's link-type model (e.g., `Blocks`, `Relates`, `Cloners`, `Duplicate`) — matches what `jtk links create --type` accepts. Run `jtk links types` to see the NAME column for your instance. The link entry lives in the SOURCE ticket's section; `target:` is the link's INWARD endpoint. A single link is stored once (in the source's section only), not twice: Jira represents the relationship from both sides automatically, rendering the OUTWARD phrase on the source side and the INWARD phrase on the target side. Example: if `SpecToTicketWorkflowTemp-2` blocks `SpecToTicketWorkflowTemp-3`, the entry lives in **#2's** section as `type: Blocks`, `target: SpecToTicketWorkflowTemp-3`. Do NOT add an inverse `is blocked by` entry in #3's section — that would double-count the relationship in Jira.
- `source` — where in the spec this ticket was derived from (section heading, page, URL if applicable)
- `short_description` — 1-3 sentence brief for the proposal view, NOT the ticket description

Plus a `### Description` body enclosed in marker comments — the full multi-paragraph ticket description that will be written to Jira. This is the meat of the ticket and should be substantial enough that someone picking up the ticket cold can act on it without reading the spec. Include:

- **Context** — why this work exists, what problem it solves
- **The work to be done** — detailed, with the specificity necessary to execute. Include requirements, acceptance criteria, expected behavior, API signatures, schema definitions
- **Pulled-forward material** — if the ticket's primary spec section references diagrams, pseudocode, tables, type definitions, or constants that live elsewhere in the spec, pull that material into the description. A ticket must not say "implement the flow shown in §1.4" without the §1.4 content being present in the ticket itself
- **Dependencies and relationships in prose** — state relationships in natural language (e.g., "This ticket blocks `SpecToTicketWorkflowTemp-4` because the frontend wiring depends on this API") in addition to the structured `links` metadata
- **Source attribution** — at the top or bottom of the description, note the spec origin: title, section heading, page, URL if available

Descriptions should be as long as the work requires — multiple paragraphs, headings, code blocks, tables are all fine. Err toward over-inclusion of context.

**Jira markdown support:** Jira Cloud accepts a substantial subset of standard Markdown in the `description` field: headings, bold/italic/strikethrough, inline code and fenced code blocks, ordered/unordered lists, tables, task-list checkboxes, blockquotes, links, and horizontal rules. For the authoritative list of supported syntax, see [Atlassian's Markdown docs](https://support.atlassian.com/jira-software-cloud/docs/markdown-and-keyboard-shortcuts/#Edit-and-create-with-markdown). Most spec content ports directly; exotic extensions (e.g., footnotes, definition lists) may not render.

#### Canonical workfile template

The following is a minimal but complete workfile showing all required structural pieces — the batch-metadata header, one fully-formed ticket section with every required field, the marker-delimited description body, and the `---` seam leading into the next ticket. Agents composing a workfile from scratch should use this as the structural template; the three detailed examples below fill out the variety of ticket shapes (standalone, sub-task with links, sub-task under existing issue):

```
# Breakdown — <spec title>

- **spec_title:** <spec title>
- **spec_source:** <where the spec lives (URL, page ref, inline)>
- **project:** <PROJECT_KEY>
- **created:** <ISO 8601 timestamp>
- **updated:** <ISO 8601 timestamp>
- **status:** proposed
- **target_sprint:** <sprint ID or none>
- **discovered_types:**
  - <Type1>
  - <Type2> (SUBTASK: yes, requires --parent)
- **discovered_priorities:**
  - <Priority1>
  - <Priority2>
- **ambiguities:**
  - <open question 1>
  - <open question 2>

## Ticket: SpecToTicketWorkflowTemp-1

- **jira_key:** SpecToTicketWorkflowTemp-1
- **type:** <Type1>
- **summary:** <concise one-line summary>
- **priority:** <Priority1>
- **assignee:** none
- **parent:** none
- **labels:** none
- **links:** none
- **source:** <spec section reference>
- **short_description:** <1-3 sentence brief for the proposal view>

### Description

<!-- SPECTOTICKETS_DESCRIPTION_START -->

<full multi-paragraph description — context, work to be done, acceptance criteria, pulled-forward material, source attribution>

<!-- SPECTOTICKETS_DESCRIPTION_END -->

---

## Ticket: SpecToTicketWorkflowTemp-2
...
```

#### Example ticket sections

Each example below is a standalone illustration of a common ticket shape. In a real workfile, ticket sections would be separated by `---` per the seam pattern shown above. The examples below use temp IDs 1, 2, and 4 — the gap at 3 is intentional to demonstrate that numbering can have gaps (e.g., after a ticket was removed during revision), as described in the temp-ID numbering rules above.

**Example 1 — Standalone top-level ticket, no parent, no links, no assignee:**

```
## Ticket: SpecToTicketWorkflowTemp-1

- **jira_key:** SpecToTicketWorkflowTemp-1
- **type:** PRD
- **summary:** Auth rework — tracking PRD
- **priority:** High
- **assignee:** none
- **parent:** none
- **labels:** none
- **links:** none
- **source:** PRD §0 (title + overview)
- **short_description:** Parent tracking ticket for the auth rework rollout across Q2.

### Description

<!-- SPECTOTICKETS_DESCRIPTION_START -->

## Context

The auth system rework is a Q2 priority driven by two forces: the compliance requirements raised by legal in Q1 (see §0.3) and the engineering team's need to remove the Postgres-based session store in favor of Redis (see §2.1).

## Scope

This PRD coordinates four distinct workstreams: session token storage redesign (tracked as a Tech Design child), API endpoint rebuild (Development child), frontend integration (Development child), and end-to-end validation (UAT child). Each has its own child ticket linked to this parent.

## Rollout target

Feature flag behind `auth_v2` starting sprint 2604.6 (post-2604.5 freeze), full rollout sprint 2604.8.

## Source

Derived from [PRD Title / Link] §0 (title + overview), §0.3 (compliance), §2.1 (session storage), §3 (API rebuild), §4 (rollout plan).

<!-- SPECTOTICKETS_DESCRIPTION_END -->
```

**Example 2 — Sub-task under another proposed ticket, with links to a sibling-in-batch and an existing issue, has assignee:**

```
## Ticket: SpecToTicketWorkflowTemp-2

- **jira_key:** SpecToTicketWorkflowTemp-2
- **type:** Tech Design
- **summary:** Design session token storage schema and lifecycle
- **priority:** High
- **assignee:** me
- **parent:** SpecToTicketWorkflowTemp-1
- **labels:** auth-rework
- **links:**
  - type: Blocks
    target: SpecToTicketWorkflowTemp-3
  - type: Relates
    target: ANP-489
  - type: Relates
    target: SpecToTicketWorkflowTemp-5
- **source:** PRD §2.1 (Session Token Storage)
- **short_description:** Schema and lifecycle design for Redis-backed session tokens per PRD §2.1. Pulled-forward auth flow from §1.4 and compliance constraints from §0.3. Peer-level relationship with the monitoring sub-task for operational observability of the new store.

### Description

<!-- SPECTOTICKETS_DESCRIPTION_START -->

## Context

The current session storage keeps tokens in a Postgres table indexed by user ID with a 24-hour sliding-window refresh. Per the compliance flags raised in Q1 (§0.3), this approach doesn't meet the new retention requirements.

## Proposed design

Move session tokens to a Redis keyed store with explicit TTL, per the auth flow in PRD §1.4 (pulled forward below):

[auth flow diagram / pseudocode from §1.4 goes here, in a fenced code block]

## Schema

| Field      | Type   | Notes |
|------------|--------|-------|
| token_id   | UUID   | Primary key, client-opaque |
| user_id    | bigint | FK to users table |
| scope      | enum   | `full`, `readonly`, `refresh` |
| expires_at | ts     | Enforced by Redis TTL, stored for audit |

## Acceptance criteria

- [ ] Sessions persist to Redis with TTL set via SETEX
- [ ] Expired sessions are not retrievable (sleep+read test)
- [ ] Audit log in Postgres captures `{user_id, token_id, scope, issued_at, expires_at}`
- [ ] Load-test: 10k concurrent logins produce zero session-lookup failures

## Related existing work

ANP-489 is the middleware layer this design reuses for token validation. No changes required there, but link added as a reference for developers onboarding to this ticket.

## Peer-level related work (no hierarchy)

This ticket has an informal "relates to" relationship with `SpecToTicketWorkflowTemp-5` (the monitoring/observability sub-task), which is a peer sibling under the same PRD parent. Neither blocks the other and there is no ordering dependency — they are flagged as related because the monitoring design needs to reference the schema defined here, and decisions made in either can usefully inform the other. Pure "peer context" link, not a dependency.

## Blocks downstream

This ticket blocks `SpecToTicketWorkflowTemp-3` (API endpoint implementation), which consumes the schema defined here.

## Source

Derived from PRD §2.1 "Session Token Storage". Flow diagram pulled forward from §1.4. Compliance context from §0.3.

<!-- SPECTOTICKETS_DESCRIPTION_END -->
```

**Example 3 — Sub-task under an existing Jira issue (not created in this batch), no links, unassigned:**

```
## Ticket: SpecToTicketWorkflowTemp-4

- **jira_key:** SpecToTicketWorkflowTemp-4
- **type:** Sub-task
- **summary:** Add session-token audit log schema to users-db
- **priority:** Medium
- **assignee:** none
- **parent:** ANP-200
- **labels:** none
- **links:** none
- **source:** PRD §2.1 (Acceptance criteria #3)
- **short_description:** Schema change to the users-db to capture session-token audit fields. Sub-task under the existing quarterly DB infrastructure epic ANP-200.

### Description

<!-- SPECTOTICKETS_DESCRIPTION_START -->

## Context

The session storage redesign in PRD §2.1 introduces an audit log requirement: every session issuance writes a row capturing `{user_id, token_id, scope, issued_at, expires_at}` to Postgres. This sub-task covers the schema change to the users-db to add that audit table.

This work slots under the existing quarterly DB infrastructure epic ANP-200, which already tracks DDL changes for Q2.

## Schema addition

[DDL pulled from spec §2.1 goes here, in a fenced code block]

## Migration plan

1. Add the `session_audit` table via standard migration tooling in the users-db repo
2. Deploy to staging first; validate no impact on existing queries
3. Deploy to prod during the rollout window defined in PRD §4

## Acceptance criteria

- [ ] Migration applied cleanly in staging
- [ ] Existing users-db queries unaffected (regression test pass)
- [ ] Migration applied to prod during rollout window

## Source

Derived from PRD §2.1 acceptance criterion #3. Parent epic ANP-200 tracks infrastructure DB changes for Q2.

<!-- SPECTOTICKETS_DESCRIPTION_END -->
```

#### 3b. Present a compact proposal derived from the workfile

Show the user a compact overview generated from the workfile. For each ticket, display:

- Position in hierarchy / relationships to other tickets
- Type
- Summary (the actual summary from 3a, verbatim)
- `short_description` (from the bullet field — 1-3 sentence brief)
- Priority / assignee if set
- Relationships to other tickets or existing issues
- Source reference (spec section)
- A note that the full description is available for review on request

Example proposal row (full temp IDs used throughout for consistency with the workfile's reference format):

```
SpecToTicketWorkflowTemp-3 [child of SpecToTicketWorkflowTemp-1]   type=Development
   Summary:     "Build API endpoint for X"
   Source:      PRD §3.2 "API Schema Design"
   Brief:       Implements POST /api/x per the auth flow in §1.4.
                Includes schema validation and error-case handling.
   Priority:    Medium
   Relationships:
     - blocks SpecToTicketWorkflowTemp-4 (frontend wiring needs this API)
     - relates to ANP-123 (reuses auth middleware)
   [Full description available on request]
```

Surface any ambiguities from stage 1 as open questions for the user to resolve.

Offer the user the option to review the full description body for any specific ticket. If they ask to see ticket #3's full description, show the content between its `<!-- SPECTOTICKETS_DESCRIPTION_START -->` and `<!-- SPECTOTICKETS_DESCRIPTION_END -->` markers verbatim — do not summarize or paraphrase.

### 4. Confirm with User, Persist Changes to Workfile

Ask the user to confirm, and wait for a concrete answer before proceeding:

- Is this the right breakdown? (Scope correct? Granularity — too fine? Too coarse? Any tickets to merge or split?)
- Are the proposed types correct per ticket?
- Is the hierarchy and relationship structure correct?
- Any priorities, assignees, or fields to adjust?
- Any tickets to add or remove?
- Want to review the full description of any specific ticket before approving?

**If the user requests any changes, update the workfile immediately** before responding. Do not hold changes in conversation context and defer the file update — the workfile is the source of truth and must reflect every confirmed edit.

When updating:

1. Edit the relevant section(s) of the workfile in place
2. Update the top section's `updated` field with the current timestamp
3. Set the top section's `status` field to `revised` (from `proposed`)
4. Present the revised view back to the user from the updated file

Iterate until the user gives explicit go-ahead. If the user substantially revises the breakdown (adds/removes/merges/splits tickets), regenerate the full proposal from the updated workfile and re-confirm — don't partial-apply changes and proceed.

**When a ticket is removed or merged during revision, scan the workfile for any remaining references to that ticket's temp ID.** Check every location a reference could live: `parent` fields on other tickets, `links[].target` entries on other tickets, prose inside description bodies (between the `<!-- SPECTOTICKETS_DESCRIPTION_START -->` / `<!-- SPECTOTICKETS_DESCRIPTION_END -->` markers), prose inside `short_description`, prose inside `source`, and prose inside the batch-level `ambiguities` field. For each remaining reference, present it to the user with its specific location and ask them to choose:

- **(a) Update the reference to point at a different ticket** — user specifies which. Agent edits the reference in place.
- **(b) Remove the reference entirely** — agent deletes the reference from whichever field it's in.
- **(c) Defer — keep the reference in place for now but note it explicitly as unresolved.** The agent is responsible for tracking deferred references within the current session and raising them to the user again before marking `status: confirmed`. If the session ends or context is lost before resolution, stage 4a validation will catch the dangling reference (check #5) and force the user to resolve it at that point. Choosing (c) is a valid "I'll think about it later" path — it is NOT a way to proceed to execution with unresolved references; validation will block.

Do not silently drop references or leave them dangling without user direction.

On confirmation:

1. Set the top section's `status` field to `confirmed`
2. Update the top section's `updated` field
3. Proceed to stage 4a (validation)

### 4a. Pre-execution Validation

Before proceeding to ManageIssueSet's Execute stage, validate the workfile. Do not proceed to stage 5 if any hard-failure check fails.

**Validation is a mandatory gate before every invocation of ManageIssueSet's Execute stage, not just the first time.** If the user hand-edits the workfile after confirmation, if a Phase 1 failure sends them back to revise and retry, if any non-trivial time elapses between confirmation and execution — re-run stage 4a before proceeding. The workfile is the authoritative input to execution, and anything that could have changed it (user edits, filesystem corruption, manual cleanup) must re-pass validation. Never invoke ManageIssueSet on a workfile that hasn't just passed validation.

**Hard failures vs soft warnings.** Most checks below are hard-failure: if any produce errors, validation fails and the user must fix before proceeding. A few checks (currently only the >150 character summary rule in check #13) are soft warnings: they surface in the validation report shown to the user for awareness but do not block execution. The user can choose to revise the ticket or proceed as-is on soft warnings. Treat everything as hard-failure unless the check text explicitly marks it as soft-warn.

**Checks:**

1. **Every ticket section has all required fields and a description body.** Required bullet-field metadata: `jira_key`, `type`, `summary`, `priority`, `assignee`, `parent`, `labels`, `links`, `source`, `short_description`. The order shown in stage 3a is recommended for readability but is NOT enforced here — validation only checks that every required field is present, not that it appears in a specific position. Plus a `### Description` heading followed by content enclosed in `<!-- SPECTOTICKETS_DESCRIPTION_START -->` and `<!-- SPECTOTICKETS_DESCRIPTION_END -->` markers. The file's top section has the batch fields: `spec_title`, `spec_source`, `project`, `created`, `updated`, `status`, `target_sprint`, `discovered_types`, `discovered_priorities`, `ambiguities`.

2. **Description markers are well-formed.** For every ticket section, exactly one `<!-- SPECTOTICKETS_DESCRIPTION_START -->` marker followed by exactly one `<!-- SPECTOTICKETS_DESCRIPTION_END -->` marker, in that order. No nesting. No overlapping pairs. No start without a matching end. No end without a preceding start. File-wide count of start markers equals file-wide count of end markers equals the number of ticket sections.

3. **`jira_key` values are unique within the workfile.** No two tickets share the same temp ID.

4. **`jira_key` values match either the temp-ID pattern or a real Jira key pattern.** Each `jira_key` must be either `SpecToTicketWorkflowTemp-\d+` (with a non-empty digit suffix) OR a real Jira key matching `[A-Z]+-\d+`.
   - A workfile where **every** `jira_key` is a temp ID is a fresh breakdown — the normal pre-execution state.
   - A workfile where **some** `jira_key` values are real Jira keys and others are temp IDs is mid-execution state, indicating Phase 1 partially completed during a prior run (see Stage 5's resume handling). This is valid when the workfile is being re-validated before a resume.
   - A workfile where **every** `jira_key` is a real Jira key is post-execution — Phase 1 completed. Re-running validation on a fully-executed workfile serves only audit purposes; the workflow has nothing further to do.

   Reject values matching neither pattern.

5. **Every temp-ID reference resolves to a declared ticket.** Scan the workfile for occurrences of `SpecToTicketWorkflowTemp-\d+` that appear in a reference context — specifically:
   - `parent` field values
   - `links[].target` values
   - Prose inside description bodies (between the `<!-- SPECTOTICKETS_DESCRIPTION_START -->` and `<!-- SPECTOTICKETS_DESCRIPTION_END -->` markers)
   - Prose inside `short_description`
   - Prose inside `source`
   - Prose inside the batch-level `ambiguities` field

   Each hit must match the `jira_key` value of exactly one ticket section. A reference that doesn't resolve is flagged with its specific location (which ticket section or batch field, and which line).

   **A declared `jira_key` does not need to be referenced anywhere else** — a ticket with no incoming references is a valid standalone ticket in the batch.

6. **Every `parent` value is `none`, a temp ID (resolved per check #5), or a real Jira key** matching the `[A-Z]+-\d+` pattern.

7. **Every `links[].target` value is a temp ID (resolved per check #5) or a real Jira key.**

8. **If `type` is a sub-task type** (per the batch metadata `discovered_types` field — rows annotated with `SUBTASK: yes`), **`parent` must be non-`none`.** Jira requires sub-tasks to have a parent.

9. **`type` is in the workfile's `discovered_types` list** (recorded in batch metadata at the end of stage 2).

10. **`priority` is in the workfile's `discovered_priorities` list** (recorded in batch metadata at the end of stage 2) when set; `none` is valid if the batch isn't setting priorities. If `discovered_priorities` is `none`, every ticket's `priority` must be `none`.

11. **`assignee` is `none`, `me`, an email address (`user@example.com` format), or a discovered Atlassian account ID.** Atlassian Cloud account IDs are typically alphanumeric strings (either a long hex-looking form like `5b10ac8d82e05b22cc7d4ef5` or a prefixed form like `712020:abc123-def456-...`). If the value doesn't match any of these shapes, flag it.

12. **No circular parent chains** within the batch (if A's parent is B, then B's parent can't transitively be A).

13. **`summary` length is within Jira's limit.** Hard-fail if `summary` is empty or exceeds 255 characters (Jira Cloud's maximum). Soft-warn if `summary` exceeds ~150 characters — this workflow prefers concise summaries for readability in the proposal view, but tickets with longer summaries still pass validation.

14. **`labels` is well-formed.** The field is either `none` or a comma-space-separated list. Individual labels must not contain whitespace (Jira Cloud does not allow spaces in labels; the CLI will reject them). Example of valid: `auth-rework, q2-goal, backend`. Example of invalid: `q2 goal` (contains a space), `auth-rework,q2-goal` (missing space after comma).

15. **`source` is non-empty.** Every ticket must attribute where in the spec its content was derived from. `source` must not be `none` or an empty string — at minimum, a section heading or page reference. Tickets with no clear spec origin indicate the agent is padding the breakdown; ask the user to confirm the ticket belongs here rather than silently committing.

16. **Parents-before-children in file position.** For every ticket whose `parent` is a temp ID, the referenced ticket must appear earlier in the file (by section position, not by numeric suffix). If ticket X has `parent: SpecToTicketWorkflowTemp-Y`, then the `## Ticket` section whose `jira_key` is `SpecToTicketWorkflowTemp-Y` must come before X's section in the workfile. This check does not apply when `parent` is `none` or a real Jira key (existing issue) — those have no in-batch position dependency. Rationale: Phase 1 execution iterates tickets in file order and creates each one as it reads it; a parent must already exist as a real Jira key before its child is created.

17. **Section heading matches `jira_key` exactly.** The section heading `## Ticket: <value>` must literally equal `## Ticket: ` followed by that section's `jira_key` field value. Heading `## Ticket: SpecToTicketWorkflowTemp-7` requires `jira_key: SpecToTicketWorkflowTemp-7`; heading `## Ticket: ANP-501` requires `jira_key: ANP-501`. Divergence is a hard failure — the workfile contains two disagreeing claims about the ticket's identifier, and execution logic reads one or the other depending on context. If this check fails, surface the exact mismatch to the user (both values, which section) and ask whether the heading or the `jira_key` is authoritative before fixing.

18. **`links[]` entries are well-formed.** Each link entry in a `- **links:**` block must have a `type:` line and a `target:` line. Standard shape: `  - type: <value>` followed on the next line by `    target: <value>` (two-space indent for the `- type:` continuation bullet, four-space indent for the `target:` sub-line). Malformed or missing `type`/`target` lines within a link entry will silently break Phase 2 link creation — flag them here. Skip this check when the ticket's `links:` field is `none`.

19. **Batch `status` is `confirmed` before execution.** The batch-metadata `status:` field must be `confirmed` when validation runs pre-execution (stage 5 entry). A `status` of `proposed` or `revised` indicates the user hasn't finished approving the breakdown — execution would bypass the confirmation contract. Any value other than `confirmed` is a hard failure. Rationale: this guards against hand-edits or process errors where someone attempts to invoke execution on an unapproved workfile.

**If validation surfaces any issues:**

1. Present every error to the user with specific locations (ticket section number, field name, or line number in a description body for prose-level errors).
2. Ask the user how to fix each issue.
3. Update the workfile to incorporate the fixes.
4. Re-run validation.
5. Loop until validation passes, or the user asks to cancel.

Common causes of validation failures:

- The agent composed a description that references a ticket that was later merged or removed during revision — stale prose reference
- The user renumbered or restructured tickets and references weren't updated
- A sub-task type was picked but no parent was assigned
- A priority was set that doesn't match the discovered scheme
- A circular parent chain emerged through a multi-step revision
- A description marker was omitted or misspelled during an edit

### 5. Proceed to ManageIssueSet for Execution

Once the user has confirmed the breakdown and validation has passed, proceed to ManageIssueSet for execution. This is the same agent continuing into another workflow's instructions, using the workfile you produced in stage 3a as the authoritative input — not a formal handoff to a different actor. Follow ManageIssueSet's Execute section, reading the workfile for every ticket, field, and relationship. Execution semantics (Phase 1 ticket creation, Phase 2 link creation, Phase 3 sprint assignment) are defined canonically in ManageIssueSet's Execute section.

**Entry precondition: stage 4a must have just passed.** If the workfile could have been modified since the last validation run (user hand-edits, time elapsed, filesystem activity), re-run stage 4a before entering Phase 1. ManageIssueSet's Execute assumes it's reading a freshly-validated workfile.

**Resume from partial execution.** If Phase 1 previously ran and failed partway (some tickets created, others not), the workfile will have real Jira keys in the `jira_key` fields of already-created tickets and `SpecToTicketWorkflowTemp-N` placeholders in the rest. This is the "mid-execution state" covered by check #4 — validation passes on this mixed state. When Phase 1 re-enters: **skip any ticket whose `jira_key` already matches the real-Jira-key pattern** (`[A-Z]+-\d+`) — it was already created in the prior run. Phase 1 only re-attempts tickets where `jira_key` is still a `SpecToTicketWorkflowTemp-N` placeholder. The same find-and-replace semantics apply after each new successful create. If the user wants to re-create a ticket that already succeeded (unusual — typically means they changed their mind about its content), they should revert that section's `jira_key` back to its original temp ID in the workfile before resuming, and delete the real Jira ticket manually.

**Detecting a crashed-mid-create scenario.** There's a narrow failure window where `jtk issues create` succeeded (a real Jira ticket exists) but the subsequent file-wide find-and-replace didn't complete — e.g., the agent crashed, the session died, or the file write was interrupted before the workfile was updated. On resume, the workfile would still show that ticket as "not yet created" (still holding its `SpecToTicketWorkflowTemp-N` value), and Phase 1 would attempt to create it again — producing a duplicate in Jira. This is unlikely but real. **Mitigation: on entry to Phase 1 against a workfile that shows any real Jira keys in its `jira_key` fields (i.e., any mid-execution state)**, ask the user: "This workfile shows partial execution progress from a prior run. Did the previous run crash or get interrupted mid-create? If so, any ticket whose workfile entry still shows a temp ID might already exist in Jira — resuming as-is could produce duplicates. Options: (a) proceed as-is (safe if the prior run ended cleanly, e.g., the user chose `abort-and-leave-as-is` after a `jtk issues create` failure), or (b) verify against Jira first — I'll run `jtk issues search --jql "project = KEY AND summary = \"<summary text>\""` for each remaining temp-ID ticket and check whether a match already exists before creating. Use **exact-equality JQL** (`summary = "..."`, not `summary ~ "..."`) to avoid fuzzy-match false positives; JQL's `~` is a contains-operator that would match unrelated tickets sharing words with the summary and silently treat the wrong ticket as 'already-created.' After running the exact-match search, additionally **verify the result's summary equals the workfile ticket's summary character-for-character** before claiming a match — Jira's `summary = "..."` is case-insensitive and treats some punctuation loosely, so a post-query exact-string check closes any residual gap. On a verified exact match, treat the ticket as already-created, update the workfile's `jira_key` to the matched real key (using the same `\b`-anchored find-and-replace as Phase 1), and skip the `jtk issues create` call for that ticket." Wait for the user's choice. If the user selects (b), run the exact-match pre-create search for each remaining temp-ID ticket with the post-query verification described above.

**Phase 1 priority-mismatch failure.** If `jtk issues create` rejects a ticket because its `priority` value isn't valid for its type (typical error wording: "Field 'priority' cannot be set" or similar indicating an invalid option for the issue type in that project), treat this as a hard stop. Do not retry with a different priority automatically. Surface the exact `jtk` error to the user and ask how to proceed. Offer three options:

- **Re-discover priorities for this specific type and pick a valid value.** Two-step: `jtk issues search --jql "project = KEY AND type = TYPENAME" --max 1 --id` to find an existing issue of that type, then `jtk issues field-options priority --issue <that-key>` to see the priorities valid for that type. The user picks a value; update the ticket's `priority` in the workfile; re-run stage 4a; re-enter Phase 1 to retry.
- **Set this ticket's `priority` to `none`** and retry the create (skip the priority field entirely for this ticket).
- **Abort execution** and revise the breakdown — user returns to stage 4 to restructure.

Do not silently choose a replacement priority or pattern-match "something close." The per-type scheme mismatch is a real configuration detail the user needs to see and decide on, not something to paper over. This handling applies equally to the initial Phase 1 run and to any resume-from-failure retry.

**How temp IDs resolve during execution:** this workflow's `SpecToTicketWorkflowTemp-N` convention is specific to the SpecToTickets → ManageIssueSet transition. During ManageIssueSet's Phase 1, each successful `jtk issues create` is followed by a file-wide find-and-replace in the workfile that swaps the ticket's `SpecToTicketWorkflowTemp-N` placeholder with the newly-issued real Jira key. Subsequent tickets that reference that temp ID (in `parent`, `links[].target`, or description prose) will see the real key by the time they're processed. The mechanic is owned by this workflow but executed by ManageIssueSet, so the same semantics are described in ManageIssueSet's "If arriving from SpecToTickets with a workfile" preamble.

**Find-and-replace must use exact token matching, not substring matching.** A naive substring replacement of `SpecToTicketWorkflowTemp-1` would corrupt `SpecToTicketWorkflowTemp-10`, `SpecToTicketWorkflowTemp-11`, `SpecToTicketWorkflowTemp-100`, etc. Use a regex with word boundaries around the temp ID: `\bSpecToTicketWorkflowTemp-<N>\b` where `<N>` is the specific integer being replaced. The `\b` anchor treats digits as word characters, so `SpecToTicketWorkflowTemp-1` and `SpecToTicketWorkflowTemp-10` are distinct tokens and won't cross-contaminate.

Tested regex that works for this pattern across common tools (all three produce identical, correct output on mixed workfiles — they correctly replace the target while leaving other temp IDs that share its prefix untouched):

- **GNU sed:** `sed -i -E 's/\bSpecToTicketWorkflowTemp-1\b/ANP-500/g' workfile.md`
- **perl:** `perl -pi -e 's/\bSpecToTicketWorkflowTemp-1\b/ANP-500/g' workfile.md`
- **Python:** `re.sub(r'\bSpecToTicketWorkflowTemp-1\b', 'ANP-500', content)`

Replacement order does not matter: replacing `-3` first then `-30`, or `-30` first then `-3`, both produce the same correct result because the word boundary prevents any cross-matching.

If using a tool that doesn't support `\b` (e.g., BSD sed without `-E`, basic string-replace functions), either switch to a tool that does, or manually ensure the pattern is **both preceded AND followed by** a non-word character (whitespace, punctuation, end-of-line, or end-of-file) — word boundaries are symmetric, so both sides must be non-word for the match to be safe. Apply this everywhere in the workfile where temp IDs appear: metadata field values, description body prose, `short_description`, `source`, the batch-level `ambiguities` field, and the `## Ticket: <jira_key>` section headings.

**Skip these ManageIssueSet sections** — they have already been accomplished here:

- Pattern classification
- Project discovery (types, priorities)
- Per-ticket type / priority / field confirmation
- Link-direction confirmation
- Plan presentation

Pick up at ManageIssueSet's **Execute** section and follow its ordering rules (parents → children → links → sprint) and failure-handling semantics from there. Do not re-implement execution logic in this workflow.

### Post-ManageIssueSet steps

After ManageIssueSet's Execute completes all applicable phases and presents its post-action summary to the user, control returns to this workflow. SpecToTickets then owns:

1. **Ask about workfile persistence.** The workfile has served its purpose as the execution payload but is also a useful audit artifact. Ask the user: "Keep the workfile at `<path>` for audit/reference, or remove it?" Default to keeping if the user is non-committal.
2. **Handle the user's answer.** Remove the workfile only on explicit user confirmation. Otherwise leave it in place.

**Failure-recovery variants** (when Execute didn't complete cleanly):

- **Abort / leave as-is** — Execute halted at a failure and the user chose to leave partially-completed work. Return here, but skip the workfile cleanup question entirely. The user may want to inspect or retry; let them decide when they've figured out next steps.
- **Resume from failure point** — ManageIssueSet handles the retry itself; control only returns here if the retry eventually succeeds (run the workfile-persistence prompt then) or the user aborts (skip the prompt).
- **Delete newly-created** — ManageIssueSet handles the destructive cleanup of created tickets and links; the workfile is NOT deleted by that recovery. Return here with the workfile intact (real keys mixed with temp IDs for uncreated tickets). Do not prompt for workfile cleanup automatically in this state — the user is in mid-recovery and will decide when to clean up.

In all failure-recovery variants, the workfile persists and remains an accurate record of what was attempted, what succeeded (rows with real Jira keys in `jira_key`), and what never got created (rows still holding `SpecToTicketWorkflowTemp-N` values).

## Anti-patterns (avoid these)

- **Creating tickets from the spec without presenting the breakdown first** — this is not ManageIssueSet, it's SpecToTickets, and the breakdown proposal is the whole point of the workflow
- **Mechanically mapping spec section structure to ticket hierarchy** without considering whether a section should become sub-tasks, siblings with links, or a single larger ticket. Use judgment; surface the choice to the user in the proposal
- **Guessing issue types without running `jtk issues types` first**
- **Inferring assignees from vague language** like "someone should do X" or "the team needs to" — leave unassigned unless the spec explicitly names a specific person
- **Padding the proposal with speculative sub-tasks** to look thorough. Err on fewer, larger tickets; let the user split further if they want
- **Applying uniform type/priority/assignee across the whole breakdown** without confirming per-ticket
- **Skipping the confirmation step to save time** — the confirmation is the value this workflow provides
- **Silently falling back to guessing** when the user's feedback on the breakdown is ambiguous — ask for clarification
- **Composing thin descriptions that just say "do the work described in §X"** without pulling the §X content into the ticket. Tickets must be self-contained
- **Presenting the compact proposal as if it's the full ticket content** — the proposal is a review artifact; the actual ticket descriptions should be substantial and live inside the `<!-- SPECTOTICKETS_DESCRIPTION_START -->` / `<!-- SPECTOTICKETS_DESCRIPTION_END -->` markers in the workfile
- **Holding proposed changes in conversation context** rather than persisting them to the workfile immediately. The workfile is the source of truth
- **Skipping the validation step in stage 4a** — validation catches real problems that would otherwise surface mid-execution in ManageIssueSet
- **Broken description markers.** The `<!-- SPECTOTICKETS_DESCRIPTION_START -->` and `<!-- SPECTOTICKETS_DESCRIPTION_END -->` markers must always come in matched pairs — every start gets exactly one end, every end belongs to exactly one start, in the order they appear. Never nest them (no start-start-end-end, no overlap). Never omit the closing marker. Never misspell either marker. Parsers depend on the markers being well-formed; a broken pair corrupts every ticket section after it until the next correctly-paired section recovers. The validation pass at stage 4a catches unpaired markers, but getting them right the first time avoids needing to re-validate.
