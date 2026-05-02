# ManageIssueSet Workflow

Create, update, link, or organize **multiple** related Jira issues as a coordinated set. Covers bulk operations, parent+sub-task hierarchies, and mixed create/update/link flows.

For single-issue operations, use `Workflows/ManageIssue.md` instead.

## When to Use This Workflow

Route here instead of ManageIssue when the request involves:

- Creating more than one issue in a single request
- Creating a parent issue plus one or more sub-tasks under it
- Updating multiple existing issues with the same or related changes
- Mixing creates and updates (e.g., "create a PRD, update these 3 to point at it")
- Establishing links between multiple issues as a planned set
- Any creation request where ordering or partial-failure matters

For spec decomposition — "turn this PRD into tickets," "decompose this design doc," etc. — route to `Workflows/SpecToTickets.md` first. That workflow handles the judgment call of proposing a ticket breakdown, composes full ticket content to a workfile, and hands off to this workflow's Execute stage once the user has confirmed the breakdown. If you arrive at this workflow from SpecToTickets with a confirmed and validated workfile, skip the Discovery & Confirmation stages below and proceed directly to Execute — SpecToTickets has already accomplished all of that.

## Four Patterns

Every multi-issue operation fits one of these shapes. Identify which up front — it shapes what you need to discover and confirm.

### Pattern A — Bulk Create

User lists multiple new issues to create. Examples:

- "Create a bug for X and a task for Y"
- "File three stories: A, B, C"
- "Create these 5 tickets in PROJ: ..."

### Pattern B — Hierarchy Create

User wants a parent issue plus sub-tasks (and possibly nested children). Examples:

- "Create a PRD with sub-tasks for design and implementation"
- "File an epic with three stories under it"
- "Make a tech design task and break it into sub-tasks for schema, API, and frontend"

### Pattern C — Bulk Update

User wants the same or related changes across multiple existing issues. Examples:

- "Set priority=High on PROJ-100, PROJ-101, PROJ-102"
- "Reassign all my open tickets in PROJ to Alice"
- "Add label `q2-goal` to these four issues"

### Pattern D — Mixed Create / Update / Link

User wants a combination — creating some issues, updating others, establishing links across the set. Examples:

- "Create a tracking issue and link these 3 existing bugs to it as 'blocks'"
- "Create a PRD, update these tickets to reference it, link them as sub-tasks"
- "File a new epic, move these existing stories under it, and add them all to the current sprint"

## Discovery & Confirmation (before any writes)

Regardless of pattern, work through every applicable step below before writing anything. Every multi-issue operation must get a plan-and-confirm step before execution — the friction is small, the cost of wrong tickets appearing in a live project is not.

1. **Project types** (once per project touched — most sets are in a single project):

   ```bash
   jtk issues types --project KEY
   ```

   Remember which types have `SUBTASK: yes` — those require `--parent KEY` on create.

2. **Per-ticket type confirmation.** Do not apply a uniform type unless the user explicitly said so ("three bugs" is uniform; "some tickets," "these work items," "the tasks from the spec" are not). For ambiguous matches — e.g., user said "design task" and the project has `Tech Design`, `UX Design`, and `Development` — ask which one for each ticket.

3. **Sub-task types specifically.** If the project has more than one type with `SUBTASK: yes` (e.g., `Technical Sub-task` + `Bug Sub-task`), ask which one for each sub-task. If there's only one sub-task type, use it but state it explicitly in the plan.

4. **Priority discovery** (if the user wants priorities set). Run `jtk issues field-options priority --issue EXISTING_KEY` against any existing issue in the target project to learn the scheme. Match the user's phrasing to the discovered values. Ask when ambiguous.

5. **Per-ticket fields.** Don't assume uniform assignee, priority, or labels unless the user said so. Confirm per ticket when in doubt. For a spec-decomposition flow, default individual tickets to unassigned unless the spec or user names a specific owner.

   **Assignee resolution.** `--assignee VALUE` accepts `me`, email, display name, or raw account ID — the resolver handles all four. When a user name is ambiguous ("Alice" on a project with multiple Alices) or the resolver fails ("user not found", "multiple users matched"), fall back to `jtk users search "QUERY" --max 10` and apply the match-count logic (0 / 1-clear / 1-doubt / 2-9 / 10-cap). For the full disambiguation pattern, see `Workflows/ManageIssue.md` → **Assign Issue → Fallback when the resolver fails**. Apply the same logic here — never silently commit to an account ID without evidence.

6. **For updates (Patterns C and D):** Confirm the exact issue keys being touched. Don't let "all my tickets" or "these" stay ambiguous — translate it to a concrete JQL query, run `jtk issues search --jql "..."` to fetch matches, **show the user the list of matched keys**, and get explicit confirmation before any updates. The user should see exactly which issues you'll modify.

7. **For links (Patterns B and D):** Confirm link direction and type. `A blocks B` and `B blocks A` are very different. Confirm the relationship semantics before creating links. Run `jtk links types` if unsure which link-type name matches the user's wording. The `--type` resolver on `jtk links create` accepts canonical names (e.g., `Blocks`), inward verbs (e.g., `is blocked by`), or outward verbs (e.g., `blocks`) — and when given an inward verb, it automatically swaps the issue key ordering to match the correct semantics.

8. **Present the plan.** List every intended operation in execution order. Use a structured format (table or numbered list) that makes the agent's inferences visible to the user. For example:

   ```
   Plan (8 operations in PROJ):

   Creates:
     1. Create PRD                  summary="[Title]"             assignee=me
     2. Create Tech Design          summary="Design schema"       parent=#1
     3. Create Development          summary="Build API endpoint"  parent=#1
     4. Create Development          summary="Wire frontend"       parent=#1

   Updates:
     5. Update PROJ-489              priority=High

   Links:
     6. Link #2 blocks #3
     7. Link #3 blocks #4

   Sprint:
     8. Add #1, #2, #3, #4 to sprint 597 (Sprint 42)
   ```

   Then ask for explicit go/no-go. Do not proceed until the user confirms. If the user corrects the plan (types, hierarchy, fields, assignees, anything), revise the plan and re-confirm before any writes.

## Execute

**If arriving from SpecToTickets with a workfile:** the workfile is the authoritative plan. Execute in three phases, all reading from the workfile:

**Phase 1 — Ticket creation pass.** Walk the workfile in order. For each ticket section, pull metadata from the bullet fields (`type`, `summary`, `priority`, `assignee`, `labels`, `parent`) and pull the full description body verbatim from between the `<!-- SPECTOTICKETS_DESCRIPTION_START -->` and `<!-- SPECTOTICKETS_DESCRIPTION_END -->` markers (use this as the Jira `description` field). Resolve `parent`: if it matches `SpecToTicketWorkflowTemp-\d+`, that ticket was created earlier in this phase — look up its current `jira_key` in the workfile (which now holds the real Jira key); if `parent` is already a real Jira key, use it directly. Call `jtk issues create`. **Do not call `jtk links create` during this phase** — preserve each ticket's `links[]` metadata unchanged. After each successful create, do a file-wide find-and-replace of that ticket's `SpecToTicketWorkflowTemp-N` with the new real key throughout the workfile — bullet field values and description body prose alike — so subsequent tickets' references resolve correctly.

**Phase 2 — Link creation pass.** Walk the workfile a second time. By this point, every `links[].target` is a real Jira key (either from the start for existing-issue references, or via Phase 1 find-and-replace for in-batch references). For each ticket's `links[]` entries, call `jtk links create SOURCE_KEY TARGET_KEY --type NAME` where SOURCE_KEY is the ticket's current `jira_key` (the source of the link), TARGET_KEY is the link's `target` (the INWARD endpoint), and NAME is the canonical link-type NAME from the workfile's `type:` field (e.g., `Blocks`, `Relates`) — which is what `jtk links create --type` accepts. Jira stores one link per pair; do not call `jtk links create` twice per link.

**Phase 3 — Sprint assignment.** If the workfile's top section has `target_sprint` set to a sprint ID (not `none`), call `jtk sprints add SPRINT_ID PROJ-1 PROJ-2 ...` with all ticket keys from this batch as positional arguments — a single call at the end.

Sequential, not parallel. Ordering rules — do not deviate without user direction:

1. **Parents first.** Epics, PRDs, top-level tasks, standalone new issues created before any child that references them with `--parent`.
2. **Children second.** Sub-tasks and any issue that needs a `--parent` flag, using keys tracked from step 1.
3. **Existing-issue updates third.** Bulk field/assignee/label changes on pre-existing tickets.
4. **Cross-issue links fourth.** `jtk links create` — after all link endpoints exist (whether from this batch or from existing tickets).
5. **Sprint membership last.** `jtk sprints add SPRINT_ID PROJ-1 PROJ-2 ...` — at the end, once all target issues exist. Note: `jtk` has no "remove from sprint" command; sprint membership is implicitly cleaned up when an issue is deleted.

Track every ID as you go — created issue keys, comment IDs, attachment IDs, link IDs. You'll need them for:

- `--parent` on children (parent key)
- `jtk links create` (both endpoint keys) — **does not return a link ID.** The Atlassian REST API responds to `POST /rest/api/3/issueLink` with a 201 and empty body, so the CLI has no ID to emit. To capture the link ID for later `jtk links delete`, immediately follow each `jtk links create SOURCE TARGET --type NAME` with `jtk links list SOURCE` and match the new entry by `(type, target)` to extract its ID. Record that ID in your tracking log.
- `jtk sprints add` (sprint ID + issue keys)
- Partial-failure reporting
- Delete-newly-created recovery if requested

### Failure Handling

**Stop on failure.** If operation N fails partway through, do not continue without user direction.

Produce a status report structured exactly like the Post-Action summary — the user sees the same shape whether the batch succeeded fully or partially:

1. **Succeeded** (same format as a fully-successful batch):
   - Summary table of issues created (keys, types, summaries, hierarchy relationships)
   - Summary of issues updated (keys, what changed)
   - Summary of links created
   - Summary of sprint memberships added
   - Browse URLs for each created issue so the user can click through and inspect

2. **Failed:**
   - Which operation (by plan-step number and description)
   - Exact error message from `jtk`
   - Likely cause if apparent (permission denied, bad type name, missing required field, etc.)

3. **Not attempted:**
   - Remaining plan-steps that never ran, in order

4. **Ask the user how to proceed.** Offer three options with clear consequences:

   - **Abort and leave as-is.** No further operations run; succeeded items stay in place. Choose this if the partial state is actually useful, or if the user wants to investigate before deciding anything further.

   - **Resume from failure point.** If the user fixes the underlying cause (grants permission, clarifies the type name, provides a missing field), retry from operation N and continue through the remaining plan-steps.

   - **Delete the newly-created issues and links from this batch.** Destructive; requires explicit user confirmation before running. Important caveats to state when offering this option:
     - Only issues and links created in THIS batch will be deleted, in reverse order (links → sub-tasks → parents).
     - **Updates applied to pre-existing issues (field changes, reassignments, label additions, comments) cannot be automatically reverted.** The agent does not have a safe way to undo updates. If the user wants those undone, they will need to run inverse updates manually, or use Jira's History view on each affected issue.
     - Confirm by stating exactly: "I'll delete these N created issues and M created links. These X updates to existing issues will NOT be reverted — [list them]." Wait for explicit yes.

Do not proceed past a failure without explicit user direction. Do not assume "abort" is the default.

## Post-Action

Produce a structured summary regardless of whether the batch fully succeeded or partially failed (the Failure Handling section uses this same summary format for the succeeded portion).

1. **Summary table.** List what was created, updated, and linked, with keys and relationships. Example:

   ```
   Created:
     PROJ-500 (PRD)           "[Title]"                  → parent
     PROJ-501 (Tech Design)   "Design schema"            → child of PROJ-500
     PROJ-502 (Development)   "Build API endpoint"       → child of PROJ-500
     PROJ-503 (Development)   "Wire frontend"            → child of PROJ-500

   Updated:
     PROJ-489  priority → High

   Linked:
     PROJ-501 blocks PROJ-502
     PROJ-502 blocks PROJ-503

   Sprint:
     PROJ-500, PROJ-501, PROJ-502, PROJ-503 → sprint 597 (Sprint 42)
   ```

2. **Offer direct links.** For each created issue, include the browse URL (e.g., `https://INSTANCE.atlassian.net/browse/PROJ-500`) so the user can click through.

3. **Flag anything the workflow couldn't do.** If the user asked for sprint addition but the auth is bearer/scoped (Agile operations unavailable), note that the sprint add was skipped. Same for any other skipped operation.

## Anti-patterns (avoid these)

- Applying uniform type, assignee, priority, or labels to a multi-issue set without explicit user confirmation — the user said "some tickets," not "these specific tickets with this specific uniform config"
- Skipping the plan-and-confirm step for a "small" or "simple" multi-issue request — always present the plan
- Creating sub-tasks before their parent exists (ordering violation)
- Creating links before both endpoints exist
- Silently guessing link direction or link type when the user's intent was ambiguous ("link these" — ask which relationship)
- Running a bulk update (Pattern C or D) without first showing the user the exact list of target issue keys
- Running `jtk` calls in parallel within a single batch — stay sequential, keeps causality clear in logs and in the failure report
- Proceeding past a mid-batch failure without user direction
- Offering "rollback" as if Jira had transactions — it doesn't. Use "delete newly-created issues and links" wording, and explicitly note that updates to existing issues can't be auto-reverted
