# jtk - Jira CLI

A command-line interface for managing Jira Cloud tickets.

## Features

- Manage Jira issues from the command line
- List, create, update, search, and delete issues
- Manage projects (create, update, delete, restore)
- Manage sprints and boards
- Add comments and perform transitions
- Manage attachments
- Manage custom fields (create, delete, restore, contexts, options)
- Manage automation rules (list, export, create, update, delete, enable/disable)
- Manage dashboards and gadgets
- Create and manage issue links
- Search and look up users
- Text-first output with `--id`, `--extended`, and `--fulltext` modifiers
- Shell completion for bash, zsh, fish, and PowerShell

## Installation

### macOS

**Homebrew (recommended)**

```bash
brew install open-cli-collective/tap/jira-ticket-cli
```

> Note: This installs from our third-party tap.

---

### Windows

**Chocolatey**

```powershell
choco install jira-ticket-cli
```

**Winget**

```powershell
winget install OpenCLICollective.jira-ticket-cli
```

---

### Linux

**Snap**

```bash
sudo snap install ocli-jira
```

> Note: After installation, the command is available as `jtk`.

**APT (Debian/Ubuntu)**

```bash
# Add the GPG key
curl -fsSL https://open-cli-collective.github.io/linux-packages/keys/gpg.asc | sudo gpg --dearmor -o /usr/share/keyrings/open-cli-collective.gpg

# Add the repository
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/open-cli-collective.gpg] https://open-cli-collective.github.io/linux-packages/apt stable main" | sudo tee /etc/apt/sources.list.d/open-cli-collective.list

# Install
sudo apt update
sudo apt install jtk
```

> Note: This is our third-party APT repository, not official Debian/Ubuntu repos.

**DNF/YUM (Fedora/RHEL/CentOS)**

```bash
# Add the repository
sudo tee /etc/yum.repos.d/open-cli-collective.repo << 'EOF'
[open-cli-collective]
name=Open CLI Collective
baseurl=https://open-cli-collective.github.io/linux-packages/rpm
enabled=1
gpgcheck=1
gpgkey=https://open-cli-collective.github.io/linux-packages/keys/gpg.asc
EOF

# Install
sudo dnf install jtk
```

> Note: This is our third-party RPM repository, not official Fedora/RHEL repos.

**Binary download**

Download `.deb`, `.rpm`, or `.tar.gz` from the [Releases page](https://github.com/open-cli-collective/atlassian-cli/releases) - available for x64 and ARM64.

```bash
# Direct .deb install
curl -LO https://github.com/open-cli-collective/atlassian-cli/releases/latest/download/jtk_VERSION_linux_amd64.deb
sudo dpkg -i jtk_VERSION_linux_amd64.deb

# Direct .rpm install
curl -LO https://github.com/open-cli-collective/atlassian-cli/releases/latest/download/jtk-VERSION.x86_64.rpm
sudo rpm -i jtk-VERSION.x86_64.rpm
```

---

### From Source

```bash
go install github.com/open-cli-collective/jira-ticket-cli/cmd/jtk@latest
```

## Quick Start

### 1. Configure jtk

```bash
jtk init
```

This will prompt you for:
- Your Jira URL (e.g., `https://mycompany.atlassian.net`)
- Your email address
- An API token

Get your API token from: https://id.atlassian.com/manage-profile/security/api-tokens

### 2. List Issues

```bash
jtk issues list --project MYPROJECT
```

### 3. Get Issue Details

```bash
jtk issues get PROJ-123
```

---

## Command Reference

### Global Flags

These flags are available on all commands:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--extended` | | `false` | Include admin/schema/audit fields in output |
| `--fulltext` | | `false` | Disable truncation of descriptions and comments |
| `--id` | | `false` | Emit only the primary identifier (takes precedence over `--extended` and `--fulltext`) |
| `--no-color` | | `false` | Disable colored output |
| `--verbose` | `-v` | `false` | Log each request's method/URL, JSON body, status, and any 4xx/5xx response body (each capped at 4 KB). Useful for diagnosing opaque Jira errors like `INVALID_INPUT`. |
| `--help` | `-h` | | Show help for command |
| `--version` | | | Show version (root command only) |

> `automation export` is the only command that emits JSON — it writes directly to stdout.

---

### `jtk init`

Initialize jtk with guided setup.

```bash
# Classic API token (Basic Auth — default)
jtk init
jtk init --url https://mycompany.atlassian.net --email user@example.com

# Service account with scoped token (Bearer Auth)
jtk init --auth-method bearer
jtk init --auth-method bearer --url https://mycompany.atlassian.net \
  --token YOUR_SCOPED_TOKEN --cloud-id YOUR_CLOUD_ID --no-verify
```

| Flag | Default | Description |
|------|---------|-------------|
| `--url` | | Jira URL (e.g., `https://mycompany.atlassian.net`) |
| `--email` | | Email address for authentication |
| `--token` | | API token |
| `--auth-method` | | Auth method: `basic` (default) or `bearer` |
| `--cloud-id` | | Cloud ID for bearer auth (find at `https://your-site.atlassian.net/_edge/tenant_info`) |
| `--no-verify` | `false` | Skip connection verification |

> **Bearer Auth:** For [Atlassian service accounts](https://support.atlassian.com/user-management/docs/manage-api-tokens-for-service-accounts/) with scoped API tokens. Email is not required. Requests route through the `api.atlassian.com` gateway.
>
> **Scope limitations:** Scoped tokens don't have scopes for Agile (boards/sprints), Automation, or Dashboards. These commands are unavailable with bearer auth — this is an Atlassian platform limitation.

---

### `jtk me`

Show information about the currently authenticated user.

```bash
jtk me
jtk me --id        # print just the account ID (for scripting)
jtk me --extended  # include timezone, locale, and group/application-role counts
```

Uses global flags `--id` and `--extended` — no command-specific flags.

---

### `jtk config`

Manage CLI configuration.

#### `jtk config show`

Display current configuration with masked credentials and source info.

```bash
jtk config show
```

#### `jtk config test`

Verify connection to Jira and test authentication.

```bash
jtk config test
```

#### `jtk config clear`

Remove stored configuration file.

```bash
jtk config clear
jtk config clear --force
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | `-f` | `false` | Skip confirmation prompt |

---

### `jtk refresh`

Refresh the local instance cache (fields, projects, users, issue types, statuses, priorities, boards, link types, etc.).

With no arguments refreshes everything. With resource names, refreshes only those plus their dependencies. Use `--status` to check freshness without fetching.

Valid resource names: `fields`, `projects`, `users`, `issuetypes`, `statuses`, `priorities`, `resolutions`, `boards`, `sprints`, `linktypes`

```bash
# Refresh everything
jtk refresh

# Refresh specific resources (auto-expands dependencies)
jtk refresh statuses
jtk refresh users issuetypes

# Show cache freshness without fetching
jtk refresh --status
```

| Flag | Default | Description |
|------|---------|-------------|
| `--status` | `false` | Print cache freshness; no network calls |

---

### `jtk issues list`

List issues in a project.

**Aliases:** `jtk issue list`, `jtk i list`

```bash
jtk issues list --project MYPROJECT
jtk issues list --project MYPROJECT --sprint current
jtk issues list --project MYPROJECT --id

# Auto-pagination: fetch up to 200 results across multiple pages
jtk issues list --project MYPROJECT --max 200

# Explicit column projection
jtk issues list --project MYPROJECT --fields summary,status,customfield_10005
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project` | `-p` | | Project key or name |
| `--sprint` | `-s` | | Filter by sprint: sprint name, numeric ID, or `current` |
| `--max` | `-m` | `50` | Maximum number of results to return |
| `--fields` | | | Comma-separated display columns (headers, Jira field IDs, or human names) |
| `--next-page-token` | | | Token for next page of results |

---

### `jtk issues get <issue-key> [issue-key...]`

Get details of one or more issues. A single key shows full detail; multiple keys show a summary table.

```bash
jtk issues get PROJ-123
jtk issues get PROJ-123 PROJ-456 PROJ-789
jtk issues get PROJ-123 --fulltext
jtk issues get PROJ-123 --id
jtk issues get PROJ-123 --fields Status,Assignee
jtk issues get PROJ-123 --custom-fields
```

| Flag | Default | Description |
|------|---------|-------------|
| `--fields` | | Comma-separated display fields (labels, Jira field IDs, or human names) |
| `--custom-fields` | `false` | Append custom fields section to output |
| `--fulltext` | `false` | Show full description without truncation (global) |
| `--id` | `false` | Emit only the issue key (global) |

**Arguments:**
- `<issue-key> [issue-key...]` - One or more issue keys (**required**)

---

### `jtk issues create`

Create a new issue.

```bash
jtk issues create --project MYPROJECT --type Task --summary "Fix login bug"
jtk issues create -p MYPROJECT -t Story -s "Add new feature" --description "Details here"
jtk issues create -p MYPROJECT -s "Custom field issue" --field priority=High --field labels=backend

# Assign to yourself, by email, or by display name
jtk issues create -p MYPROJECT -t Task -s "My task" --assignee me
jtk issues create -p MYPROJECT -t Task -s "Their task" --assignee user@example.com
jtk issues create -p MYPROJECT -t Task -s "Their task" --assignee "Aaron Wong"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project` | `-p` | | Project key or name (**required**) |
| `--type` | `-t` | `Task` | Issue type: `Task`, `Bug`, `Story`, etc. |
| `--summary` | `-s` | | Issue summary (**required**) |
| `--description` | `-d` | | Issue description (supports `\n`, `\t`, `\\` escape sequences) |
| `--parent` | | | Parent issue key (epic or parent issue) |
| `--assignee` | `-a` | | Assignee (account ID, email, display name, or `"me"`) |
| `--field` | `-f` | | Additional field in `key=value` format (can be repeated) |

---

### `jtk issues update <issue-key>`

Update an existing issue.

```bash
jtk issues update PROJ-123 --summary "New summary"
jtk issues update PROJ-123 --field priority=High
jtk issues update PROJ-123 --description "Updated description" --field labels=urgent

# Unassign an issue
jtk issues update PROJ-123 --assignee none

# Change workflow status (routes to the transitions API under the hood).
# Quote multi-word status names: --status "In Progress"
jtk issues update PROJ-123 --status "Done"

# Multi-value fields: repeat --field with the same key to accumulate values
jtk issues update PROJ-123 --field customfield_10050=Option1 --field customfield_10050=Option2
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--summary` | `-s` | | New summary |
| `--description` | `-d` | | New description (supports `\n`, `\t`, `\\` escape sequences) |
| `--parent` | | | Parent issue key (epic or parent issue) |
| `--assignee` | `-a` | | Assignee (account ID, email, display name, `"me"`, or `"none"` to unassign) |
| `--type` | `-t` | | New issue type (uses Jira Cloud bulk move API) |
| `--status` | | | New workflow status (uses Jira transitions API; resolved before any writes) |
| `--field` | `-f` | | Field to update in `key=value` format (can be repeated; repeating the same key accumulates values for multi-select fields) |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk issues search`

Search issues using JQL.

```bash
jtk issues search --jql "project = MYPROJECT AND status = 'In Progress'"
jtk issues search --jql "assignee = currentUser()" --id

# Auto-pagination: fetch up to 200 results across multiple pages
jtk issues search --jql "project = MYPROJECT" --max 200

# Explicit column projection
jtk issues search --jql "project = MYPROJECT" --fields summary,status
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--jql` | | | JQL query string (**required**) |
| `--max` | `-m` | `50` | Maximum number of results to return |
| `--fields` | | | Comma-separated display columns (headers, Jira field IDs, or human names) |
| `--next-page-token` | | | Token for next page of results |

---

### `jtk issues assign <issue-key> [user]`

Assign an issue to a user, or unassign it. The `[user]` argument accepts an account ID, email, display name, or `"me"`.

```bash
jtk issues assign PROJ-123 5b10ac8d82e05b22cc7d4ef5
jtk issues assign PROJ-123 "Aaron Wong"
jtk issues assign PROJ-123 aaron@example.com
jtk issues assign PROJ-123 me
jtk issues assign PROJ-123 --unassign
```

| Flag | Default | Description |
|------|---------|-------------|
| `--unassign` | `false` | Remove current assignee |

**Arguments:**
- `<issue-key>` - The issue key (**required**)
- `[user]` - Account ID, email, display name, or `"me"` (required unless `--unassign`)

---

### `jtk issues delete <issue-key> [issue-key...]`

Delete one or more issues.

```bash
jtk issues delete PROJ-123
jtk issues delete PROJ-123 PROJ-124 PROJ-125
jtk issues delete PROJ-123 --force
```

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Skip confirmation prompt |

**Arguments:**
- `<issue-key> [issue-key...]` - One or more issue keys (**required**)

---

### `jtk issues archive <issue-key> [issue-key...]`

Archive one or more issues. Archived issues are hidden from boards and search by default but remain in Jira. There is no `issues restore` command — use the Jira UI to unarchive.

```bash
jtk issues archive PROJ-123
jtk issues archive PROJ-123 PROJ-124 PROJ-125
jtk issues archive PROJ-123 --id
```

**Arguments:**
- `<issue-key> [issue-key...]` - One or more issue keys (**required**)

---

### `jtk issues check <issue-key>`

Check whether an issue has values for expected fields. Useful as a guardrail
before transitions or as a CI step. Each field can be named by its display name
(e.g. `Story Points`), Jira field ID (e.g. `customfield_10035`), or property
key (e.g. `assignee`).

```bash
# Default warn list (Summary, Description, Assignee, Priority, Labels,
# Story Points, Sprint, Components, Fix Version/s) — fields not on the
# project's schema are silently skipped.
jtk issues check PROJ-123

# Hard-fail (non-zero exit) if Story Points or Sprint are missing.
jtk issues check PROJ-123 --require "Story Points" --require Sprint

# Mix required and warning fields, comma-separated.
jtk issues check PROJ-123 --require "Story Points,Sprint" --warn "Description,Assignee"

# Emit only the IDs of MISSING fields.
jtk issues check PROJ-123 --require Sprint --id
```

| Flag | Default | Description |
|------|---------|-------------|
| `--require` | (none) | Field must be populated; missing → non-zero exit (repeatable) |
| `--warn` | (curated list, only when neither flag is provided) | Field flagged if missing; never fails the check (repeatable) |

When `--require` is provided alone, the curated default warn-list is **not** applied — only the explicitly-named fields are checked.

Use `--id` to emit only the IDs of fields whose status is `MISSING`.

**Exit codes:** `0` if all `--require` fields populated; `1` if any are missing.

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk issues fields [issue-key]`

List available fields, or show all fields with their current values for a specific issue.

```bash
jtk issues fields                    # All fields
jtk issues fields PROJ-123           # Field values for a specific issue
jtk issues fields --custom-fields    # Custom fields only
```

| Flag | Default | Description |
|------|---------|-------------|
| `--custom-fields` | `false` | Show only custom fields |

**Arguments:**
- `[issue-key]` - Optional issue key to show field values

---

### `jtk issues field-options [issue-key] <field-name-or-id>`

List allowed values for a field. Providing an issue key uses that issue's project context (recommended); omitting it uses the global field context.

The first positional argument is treated as an issue key if it matches the `PROJ-123` pattern (uppercase letters/digits, hyphen, digits); otherwise it is treated as the field name.

```bash
jtk issues field-options PROJ-123 priority
jtk issues field-options PROJ-123 customfield_10001
jtk issues field-options priority   # without issue context (single arg = field name)
```

**Arguments:**
- `[issue-key]` - Optional issue key for context-specific options (must match `KEY-NNN` pattern)
- `<field-name-or-id>` - Field name or ID (**required**)

---

### `jtk issues types`

List available issue types for a project.

```bash
jtk issues types --project MYPROJECT
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project` | `-p` | | Project key (**required**) |

---

### `jtk issues move <issue-key>...`

Move one or more issues to a different project (Cloud only, max 1000 issues).

```bash
jtk issues move PROJ-123 --to-project OTHERPROJ
jtk issues move PROJ-123 PROJ-124 PROJ-125 --to-project OTHERPROJ --to-type Bug

# Move without waiting for completion
jtk issues move PROJ-123 --to-project OTHERPROJ --no-wait

# Move without notifications
jtk issues move PROJ-123 --to-project OTHERPROJ --no-notify
```

| Flag | Default | Description |
|------|---------|-------------|
| `--to-project` | | Target project key or name (**required**) |
| `--to-type` | (same as source) | Target issue type name |
| `--notify` | `true` | Send notifications; use `--no-notify` to disable |
| `--wait` | `true` | Wait for move to complete; use `--no-wait` to return immediately with the task ID |

**Arguments:**
- `<issue-key>...` - One or more issue keys (**required**)

---

### `jtk issues move-status <task-id>`

Check the status of an asynchronous move operation.

```bash
jtk issues move-status 12345
```

**Arguments:**
- `<task-id>` - The task ID returned by `issues move` (**required**)

---

### `jtk links list <issue-key>`

List all links on an issue.

**Aliases:** `jtk link list`, `jtk l list`

```bash
jtk links list PROJ-123
jtk links list PROJ-123 --id
jtk links list PROJ-123 --fields TYPE,ISSUE
```

| Flag | Default | Description |
|------|---------|-------------|
| `--fields` | | Comma-separated display columns |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk links create <issue-key> <target-issue-key>`

Create a link between two issues. The first issue is the outward issue and the second is the inward issue. `--type` accepts the canonical name, the outward verb, or the inward verb.

```bash
# A blocks B
jtk links create PROJ-123 PROJ-456 --type Blocks

# A is blocked by B (inward verb — issues are interpreted from user's perspective)
jtk links create PROJ-123 PROJ-456 --type "is blocked by"

# A relates to B
jtk links create PROJ-123 PROJ-456 --type Relates
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--type` | `-t` | | Link type: canonical name, outward verb, or inward verb (**required**) |

**Arguments:**
- `<issue-key>` - The outward issue key (**required**)
- `<target-issue-key>` - The inward issue key (**required**)

> Tip: Use `jtk links types` to see available link types.

---

### `jtk links delete <link-id>`

Delete an issue link.

```bash
jtk links delete 10001
```

**Arguments:**
- `<link-id>` - The link ID (**required**)

> Tip: Use `jtk links list PROJ-123` to find link IDs.

---

### `jtk links types`

List available issue link types.

```bash
jtk links types
jtk links types --id
```

| Flag | Default | Description |
|------|---------|-------------|
| `--fields` | | Comma-separated display columns |

---

### `jtk transitions list <issue-key>`

List available transitions for an issue.

**Aliases:** `jtk transition list`, `jtk tr list`

```bash
jtk transitions list PROJ-123
jtk transitions list PROJ-123 --extended
jtk transitions list PROJ-123 --id
```

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk transitions do <issue-key> <transition>`

Perform a transition on an issue. For ordinary status changes, prefer
`jtk issues update <key> --status <name>` — it hides the Jira API split.
Reach for `transitions do` when you need to disambiguate multiple
transitions to the same target status, set fields-on-transition, or pick
a transition by ID.

**Aliases:** `jtk transition do`, `jtk tr do`

```bash
jtk transitions do PROJ-123 "In Progress"
jtk transitions do PROJ-123 "Done"
jtk transitions do PROJ-123 "Done" --field resolution=Fixed
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--field` | `-f` | | Field to set during transition in `key=value` format (can be repeated) |

**Arguments:**
- `<issue-key>` - The issue key (**required**)
- `<transition>` - Transition name or ID (**required**)

---

### `jtk comments list <issue-key>`

List comments on an issue.

**Aliases:** `jtk comment list`, `jtk c list`

```bash
jtk comments list PROJ-123
jtk comments list PROJ-123 --fulltext
jtk comments list PROJ-123 --fields ID,AUTHOR
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--max` | `-m` | `50` | Maximum number of comments |
| `--fulltext` | | `false` | Show full comment bodies without truncation (global) |
| `--fields` | | | Comma-separated display fields |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk comments add <issue-key>`

Add a comment to an issue.

**Aliases:** `jtk comment add`, `jtk c add`

```bash
jtk comments add PROJ-123 --body "This is my comment"
jtk comments add PROJ-123 --body "Line one\nLine two\n\tIndented line"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--body` | `-b` | | Comment text (supports `\n`, `\t`, `\\` escape sequences) (**required**) |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk comments delete <issue-key> <comment-id>`

Delete a comment from an issue.

**Aliases:** `jtk comment delete`, `jtk c delete`

```bash
jtk comments delete PROJ-123 10042
```

**Arguments:**
- `<issue-key>` - The issue key (**required**)
- `<comment-id>` - The comment ID (**required**)

---

### `jtk attachments list <issue-key>`

List attachments on an issue.

**Aliases:** `jtk attachments ls`, `jtk attachment list`, `jtk att list`

```bash
jtk attachments list PROJ-123
jtk attachments list PROJ-123 --id
```

| Flag | Default | Description |
|------|---------|-------------|
| `--fields` | | Comma-separated display columns |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk attachments add <issue-key>`

Upload file(s) to an issue.

**Aliases:** `jtk attachment add`, `jtk att add`

```bash
jtk attachments add PROJ-123 --file screenshot.png
jtk attachments add PROJ-123 --file doc.pdf --file image.png
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-F` | | File to attach (**required**, can be repeated) |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk attachments get <attachment-id>`

Download an attachment.

**Aliases:** `jtk attachments download`, `jtk attachment get`, `jtk att get`

```bash
jtk attachments get 12345
jtk attachments get 12345 --output ./downloads/
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `.` | Output path (directory or filename) |

**Arguments:**
- `<attachment-id>` - The attachment ID (**required**)

---

### `jtk attachments delete <attachment-id>`

Delete an attachment.

**Aliases:** `jtk attachments rm`, `jtk attachment delete`, `jtk att delete`

```bash
jtk attachments delete 12345
```

**Arguments:**
- `<attachment-id>` - The attachment ID (**required**)

---

### `jtk sprints list`

List sprints for a board. `--board` accepts a board ID or name.

```bash
jtk sprints list --board 123
jtk sprints list --board "MON board" --state active
jtk sprints list --board 123 --id
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--board` | `-b` | | Board ID or name (**required**) |
| `--state` | `-s` | | Filter by state: `active`, `closed`, `future` |
| `--max` | `-m` | `50` | Maximum number of results |
| `--fields` | | | Comma-separated display columns |
| `--next-page-token` | | | Token for next page of results |

---

### `jtk sprints current`

Show the current active sprint.

```bash
jtk sprints current --board 123
jtk sprints current --board "MON board"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--board` | `-b` | | Board ID or name (**required**) |
| `--fields` | | | Comma-separated display fields |

---

### `jtk sprints issues <sprint>`

List issues in a sprint. Accepts a sprint ID or name (resolved via cache).

```bash
jtk sprints issues 456
jtk sprints issues "MON Sprint 70"
jtk sprints issues 456 --id
jtk sprints issues 456 --fields KEY,STATUS,customfield_10005
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--max` | `-m` | `50` | Maximum number of results |
| `--fields` | | | Comma-separated display columns |
| `--next-page-token` | | | Token for next page of results |

**Arguments:**
- `<sprint>` - Sprint ID or name (**required**)

---

### `jtk sprints add <sprint> <issue-key>...`

Move one or more issues to a sprint. Accepts a sprint ID or name.

```bash
jtk sprints add 456 PROJ-123
jtk sprints add "MON Sprint 70" PROJ-123
jtk sprints add 456 PROJ-123 PROJ-124 PROJ-125
```

**Arguments:**
- `<sprint>` - Sprint ID or name (**required**)
- `<issue-key>...` - One or more issue keys (**required**)

---

### `jtk sprints remove <issue-key>...`

Move one or more issues from their current sprint to the backlog.

```bash
jtk sprints remove PROJ-456
jtk sprints remove PROJ-456 PROJ-789 PROJ-101
```

**Arguments:**
- `<issue-key>...` - One or more issue keys (**required**)

---

### `jtk boards list`

List boards.

```bash
jtk boards list
jtk boards list --project MYPROJECT
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project` | `-p` | | Filter by project key or name |
| `--max` | `-m` | `50` | Maximum number of results |
| `--fields` | | | Comma-separated display columns |
| `--next-page-token` | | | Token for next page of results |

---

### `jtk boards get <board>`

Get board details. Accepts a board ID or name (resolved via cache).

```bash
jtk boards get 123
jtk boards get "MON board"
```

| Flag | Default | Description |
|------|---------|-------------|
| `--fields` | | Comma-separated display fields |

**Arguments:**
- `<board>` - Board ID or name (**required**)

---

### `jtk projects list`

List Jira projects.

**Aliases:** `jtk project list`, `jtk proj list`, `jtk p list`

```bash
jtk projects list
jtk projects list --query "my project"
jtk projects list --max 10
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--query` | `-q` | | Filter projects by name |
| `--max` | `-m` | `50` | Maximum number of results |
| `--fields` | | | Comma-separated display columns |
| `--next-page-token` | | | Token for next page of results |

---

### `jtk projects get <project-key>`

Get details for a specific project.

```bash
jtk projects get MYPROJECT
jtk projects get 10001
```

| Flag | Default | Description |
|------|---------|-------------|
| `--fields` | | Comma-separated display fields |

**Arguments:**
- `<project-key>` - Project key or numeric ID (**required**)

---

### `jtk projects create`

Create a new Jira project.

```bash
jtk projects create --key MYPROJ --name "My Project" --lead me
jtk projects create --key BIZ --name "Business" --type business --lead "Aaron Wong" --description "Business project"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--key` | `-k` | | Project key (**required**) |
| `--name` | `-n` | | Project name (**required**) |
| `--type` | `-t` | `software` | Project type: `software`, `service_desk`, `business` |
| `--lead` | `-l` | | Lead: account ID, email, display name, or `"me"` (**required**) |
| `--description` | `-d` | | Project description |

> Tip: Use `jtk users search` to find account IDs, or `jtk me` to get your own.

---

### `jtk projects update <project-key>`

Update a project's metadata. Only specified fields are changed.

```bash
jtk projects update MYPROJ --name "New Name"
jtk projects update MYPROJ --description "Updated description"
jtk projects update MYPROJ --lead "Aaron Wong"
jtk projects update MYPROJ --lead aaron@example.com
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | `-n` | | New project name |
| `--description` | `-d` | | New project description |
| `--lead` | `-l` | | New lead: account ID, email, display name, or `"me"` |

**Arguments:**
- `<project-key>` - Project key (**required**)

---

### `jtk projects delete <project-key>`

Soft-delete a project (moves it to trash). Can be restored with `jtk projects restore`.

```bash
jtk projects delete MYPROJ
jtk projects delete MYPROJ --force
```

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Skip confirmation prompt |

**Arguments:**
- `<project-key>` - Project key (**required**)

---

### `jtk projects restore <project-key>`

Restore a project from the trash.

```bash
jtk projects restore MYPROJ
```

**Arguments:**
- `<project-key>` - Project key (**required**)

---

### `jtk projects types`

List available project types for creating new projects.

```bash
jtk projects types
```

---

### `jtk users search <query>`

Search for Jira users.

**Aliases:** `jtk user search`, `jtk u search`

```bash
jtk users search "john"
jtk users search "john" --max 20
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--max` | `-m` | `50` | Maximum number of results |
| `--fields` | | | Comma-separated display columns |
| `--next-page-token` | | | Token for next page of results |

**Arguments:**
- `<query>` - Search query (matches display name, email, etc.) (**required**)

---

### `jtk users get <account-id>`

Get details for a specific user by account ID.

**Aliases:** `jtk user get`, `jtk u get`

```bash
jtk users get 5b10ac8d82e05b22cc7d4ef5
jtk users get 5b10ac8d82e05b22cc7d4ef5 --id     # global flag: emit only account ID
jtk users get 5b10ac8d82e05b22cc7d4ef5 --extended
```

| Flag | Default | Description |
|------|---------|-------------|
| `--fields` | | Comma-separated display fields |
| `--id` | `false` | Emit only the account ID (global flag) |

**Arguments:**
- `<account-id>` - The Atlassian account ID (**required**)

---

### `jtk automation list`

List automation rules.

**Aliases:** `jtk auto list`

```bash
jtk automation list
jtk automation list --state ENABLED
```

| Flag | Default | Description |
|------|---------|-------------|
| `--state` | | Filter by state: `ENABLED` or `DISABLED` |

---

### `jtk automation get <rule-id>`

Get details of an automation rule.

**Aliases:** `jtk auto get`

```bash
jtk automation get 123
jtk automation get 123 --show-components
```

| Flag | Default | Description |
|------|---------|-------------|
| `--show-components` | `false` | Show component type details |

**Arguments:**
- `<rule-id>` - The rule ID (**required**)

---

### `jtk automation export <rule-id>`

Export a rule definition as JSON.

**Aliases:** `jtk auto export`

```bash
jtk automation export 123
jtk automation export 123 --compact
jtk automation export 123 > rule-backup.json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--compact` | `false` | Output minified JSON |

**Arguments:**
- `<rule-id>` - The rule ID (**required**)

> Note: Output is always JSON — this is the only jtk command that emits JSON directly.

---

### `jtk automation create`

Create an automation rule from a JSON file.

**Aliases:** `jtk auto create`

```bash
jtk automation create --file rule-definition.json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-F` | | Path to JSON file containing the rule definition (**required**) |

> Note: New rules are created in DISABLED state by default.

---

### `jtk automation update <rule-id>`

Update an automation rule from a JSON file.

**Aliases:** `jtk auto update`

```bash
jtk automation update 123 --file updated-rule.json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-F` | | Path to JSON file containing the rule definition (**required**) |

**Arguments:**
- `<rule-id>` - The rule ID (**required**)

> Tip: Use `jtk automation export` to get the current definition before editing.

---

### `jtk automation delete <rule-id>`

Permanently delete an automation rule. If the rule is currently ENABLED, it will be automatically disabled before deletion. This action cannot be undone.

**Aliases:** `jtk auto delete`

```bash
jtk automation delete 123
jtk automation delete 123 --force
```

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Skip confirmation prompt |

**Arguments:**
- `<rule-id>` - The rule ID (**required**)

---

### `jtk automation enable <rule-id>`

Enable a disabled automation rule.

**Aliases:** `jtk auto enable`

```bash
jtk automation enable 123
```

**Arguments:**
- `<rule-id>` - The rule ID (**required**)

---

### `jtk automation disable <rule-id>`

Disable an enabled automation rule.

**Aliases:** `jtk auto disable`

```bash
jtk automation disable 123
```

**Arguments:**
- `<rule-id>` - The rule ID (**required**)

---

### `jtk dashboards list`

List accessible dashboards.

**Aliases:** `jtk dashboard list`, `jtk dash list`

```bash
jtk dashboards list
jtk dashboards list --search "Sprint"
jtk dashboards list --max 10
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--search` | | | Search dashboards by name |
| `--max` | `-m` | `50` | Maximum number of results |

> Note: Dashboard commands are not available with bearer auth (scoped tokens lack the Dashboard scope).

---

### `jtk dashboards get <dashboard-id>`

Get dashboard details including gadgets.

```bash
jtk dashboards get 10001
```

**Arguments:**
- `<dashboard-id>` - The dashboard ID (**required**)

---

### `jtk dashboards create`

Create a new dashboard.

```bash
jtk dashboards create --name "My Dashboard"
jtk dashboards create --name "Sprint Board" --description "Sprint tracking"
```

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | | Dashboard name (**required**) |
| `--description` | | Dashboard description |

---

### `jtk dashboards delete <dashboard-id>`

Delete a dashboard.

```bash
jtk dashboards delete 10001
```

**Arguments:**
- `<dashboard-id>` - The dashboard ID (**required**)

---

### `jtk dashboards gadgets list <dashboard-id>`

List gadgets on a dashboard.

```bash
jtk dashboards gadgets list 10001
jtk dashboards gadgets list 10001 --id
```

**Arguments:**
- `<dashboard-id>` - The dashboard ID (**required**)

---

### `jtk dashboards gadgets add <dashboard-id>`

Add a gadget to a dashboard by its module key.

```bash
jtk dashboards gadgets add 10001 --type com.atlassian.jira.gadgets:sprint-burndown-gadget
jtk dashboards gadgets add 10001 --type com.atlassian.jira.gadgets:filter-results-gadget --position 1,0 --title "My Filter"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--type` | `-t` | | Gadget module key (**required**) |
| `--position` | `-p` | | Position as `row,column` (e.g. `1,0`) |
| `--title` | | | Gadget title |
| `--color` | | | Gadget color |

**Arguments:**
- `<dashboard-id>` - The dashboard ID (**required**)

---

### `jtk dashboards gadgets remove <dashboard-id> <gadget-id>`

Remove a gadget from a dashboard.

```bash
jtk dashboards gadgets remove 10001 42
```

**Arguments:**
- `<dashboard-id>` - The dashboard ID (**required**)
- `<gadget-id>` - The gadget ID (**required**)

---

### `jtk fields list`

List all fields (system and custom). Supports filtering by name with case-insensitive substring matching.

**Aliases:** `jtk field list`, `jtk f list`

```bash
jtk fields list
jtk fields list --custom-fields
jtk fields list --name "story point"
jtk fields list --id
```

| Flag | Default | Description |
|------|---------|-------------|
| `--custom-fields` | `false` | Show only custom fields |
| `--name` | | Filter fields by name (case-insensitive substring match) |

#### `jtk fields show <field-id>`

Show a flat denormalized view of a field's contexts, project mappings, and options.

```bash
jtk fields show customfield_10001
jtk fields show customfield_10001 --id   # emit context IDs only
```

**Arguments:**
- `<field-id>` - The field ID (**required**)

#### `jtk fields create`

Create a new custom field.

```bash
jtk fields create --name "My Select Field" --type com.atlassian.jira.plugin.system.customfieldtypes:select
jtk fields create --name "My Text Field" --type com.atlassian.jira.plugin.system.customfieldtypes:textarea --description "A text area field"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | `-n` | | Field name (**required**) |
| `--type` | `-t` | | Field type (**required**) |
| `--description` | `-d` | | Field description |

#### `jtk fields delete <field-id>`

Trash a custom field (can be restored).

```bash
jtk fields delete customfield_10001
jtk fields delete customfield_10001 --force
```

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Skip confirmation prompt |

**Arguments:**
- `<field-id>` - The field ID (**required**)

#### `jtk fields restore <field-id>`

Restore a trashed custom field.

```bash
jtk fields restore customfield_10001
```

**Arguments:**
- `<field-id>` - The field ID (**required**)

#### `jtk fields contexts list <field-id>`

List contexts for a custom field.

**Aliases:** `jtk fields context list`, `jtk fields ctx list`

```bash
jtk fields contexts list customfield_10001
jtk fields contexts list customfield_10001 --id
```

**Arguments:**
- `<field-id>` - The field ID (**required**)

#### `jtk fields contexts create <field-id>`

Create a context for a custom field.

**Aliases:** `jtk fields context create`, `jtk fields ctx create`

```bash
jtk fields contexts create customfield_10001 --name "My Context"
jtk fields contexts create customfield_10001 --name "Project Context" --project 10001
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | `-n` | | Context name (**required**) |
| `--project` | `-p` | | Project ID to scope the context to |

**Arguments:**
- `<field-id>` - The field ID (**required**)

#### `jtk fields contexts delete <field-id> <context-id>`

Delete a context from a custom field.

**Aliases:** `jtk fields context delete`, `jtk fields ctx delete`

```bash
jtk fields contexts delete customfield_10001 10100
jtk fields contexts delete customfield_10001 10100 --force
```

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Skip confirmation prompt |

**Arguments:**
- `<field-id>` - The field ID (**required**)
- `<context-id>` - The context ID (**required**)

#### `jtk fields options list <field-id>`

List options for a select/multi-select custom field. Auto-detects the default context if `--context` is not specified.

**Aliases:** `jtk fields option list`, `jtk fields opt list`

```bash
jtk fields options list customfield_10001
jtk fields options list customfield_10001 --context 10001
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--context` | `-c` | | Context ID (auto-detected if omitted) |

**Arguments:**
- `<field-id>` - The field ID (**required**)

#### `jtk fields options add <field-id>`

Add an option to a select/multi-select custom field.

**Aliases:** `jtk fields option add`, `jtk fields opt add`

```bash
jtk fields options add customfield_10001 --value "New Option"
jtk fields options add customfield_10001 --value "Staging" --context 10001
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--value` | `-V` | | Option value (**required**) |
| `--context` | `-c` | | Context ID (auto-detected if omitted) |

**Arguments:**
- `<field-id>` - The field ID (**required**)

#### `jtk fields options update <field-id>`

Update an existing option value.

**Aliases:** `jtk fields option update`, `jtk fields opt update`

```bash
jtk fields options update customfield_10001 --option 10200 --value "Updated Value"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--option` | | | Option ID to update (**required**) |
| `--value` | `-V` | | New option value (**required**) |
| `--context` | `-c` | | Context ID (auto-detected if omitted) |

**Arguments:**
- `<field-id>` - The field ID (**required**)

#### `jtk fields options delete <field-id>`

Delete an option from a select/multi-select custom field.

**Aliases:** `jtk fields option delete`, `jtk fields opt delete`

```bash
jtk fields options delete customfield_10001 --option 10200
jtk fields options delete customfield_10001 --option 10200 --force
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--option` | | | Option ID to delete (**required**) |
| `--force` | | `false` | Skip confirmation prompt |
| `--context` | `-c` | | Context ID (auto-detected if omitted) |

**Arguments:**
- `<field-id>` - The field ID (**required**)

---

## Configuration

`jtk init` stores the **API token in your OS keyring** (macOS Keychain /
Linux Secret Service / Windows Credential Manager, or an opt-in
encrypted-file backend) and writes only **non-secret** config to the
shared store at `~/.config/atlassian-cli/config.yml`:

```yaml
default:
  url: https://mycompany.atlassian.net
  email: user@example.com
  auth_method: basic                # or "bearer"
  cloud_id: ""                      # required for bearer
jtk:
  default_project: MYPROJECT        # jtk-only defaults
```

There is **no `api_token:` field** — the secret never touches a
plaintext file. The same config file and keyring bundle are shared with
`cfl` — one Atlassian token, both tools. Run `jtk init` after `cfl init`
(or vice versa) and you'll be offered to reuse the credentials. To set a
token non-interactively: `jtk set-credential` (reads stdin or
`--from-env VAR`).

Legacy per-tool config keeps working indefinitely (Linux: `~/.config/jira-ticket-cli/config.json`; macOS: `~/Library/Application Support/jira-ticket-cli/config.json`). The first command auto-migrates any pre-existing plaintext token into the keyring and scrubs the plaintext in place.

Run `jtk config show` to inspect the resolved values, including the keyring ref, backend, and whether a token is configured (the token value itself is never displayed). Token/keyring reporting is authoritative; the non-secret rows reflect env + the legacy per-tool file only, so a value set solely in the shared store appears as "-" there even though jtk uses it at runtime. `jtk config clear` removes the tool's resolved key; `jtk config clear --all` removes the whole bundle plus the non-secret config file.

### Environment Variables

Environment variables override file-based config. Variables are checked in order of precedence (first match wins):

| Setting | Precedence (highest to lowest) |
|---------|-------------------------------|
| URL | `JIRA_URL` → `ATLASSIAN_URL` → shared `jtk` override → shared `default` → legacy → `JIRA_DOMAIN` |
| Email | `JIRA_EMAIL` → `ATLASSIAN_EMAIL` → shared `jtk` → shared `default` → legacy |
| API Token | `JIRA_API_TOKEN` → `ATLASSIAN_API_TOKEN` → keyring `jtk_api_token` → keyring `api_token` (OS keyring, never a plaintext file) |
| Default Project | `JIRA_DEFAULT_PROJECT` → shared `jtk.default_project` → legacy |
| Auth Method | `JIRA_AUTH_METHOD` → `ATLASSIAN_AUTH_METHOD` → shared → legacy → `basic` |
| Cloud ID | `JIRA_CLOUD_ID` → `ATLASSIAN_CLOUD_ID` → shared → legacy |

**Shared credentials:** If you use both `jtk` and `cfl` (Confluence CLI), set `ATLASSIAN_*` variables once:

```bash
export ATLASSIAN_URL=https://mycompany.atlassian.net
export ATLASSIAN_EMAIL=user@example.com
export ATLASSIAN_API_TOKEN=your-api-token
```

**Per-tool override:** Use `JIRA_*` to override for Jira specifically:

```bash
export ATLASSIAN_EMAIL=user@example.com
export ATLASSIAN_API_TOKEN=your-api-token
export JIRA_URL=https://jira.internal.corp.com  # Different URL for Jira
```

> **Note:** The legacy `JIRA_DOMAIN` environment variable is still supported for backwards compatibility but is deprecated.

---

## Shell Completion

jtk supports tab completion for bash, zsh, fish, and PowerShell.

### Bash

```bash
# Load in current session
source <(jtk completion bash)

# Install permanently (Linux)
jtk completion bash | sudo tee /etc/bash_completion.d/jtk > /dev/null

# Install permanently (macOS with Homebrew)
jtk completion bash > $(brew --prefix)/etc/bash_completion.d/jtk
```

### Zsh

```bash
# Load in current session
source <(jtk completion zsh)

# Install permanently
mkdir -p ~/.zsh/completions
jtk completion zsh > ~/.zsh/completions/_jtk

# Add to ~/.zshrc if not already present:
# fpath=(~/.zsh/completions $fpath)
# autoload -Uz compinit && compinit
```

### Fish

```bash
# Load in current session
jtk completion fish | source

# Install permanently
jtk completion fish > ~/.config/fish/completions/jtk.fish
```

### PowerShell

```powershell
# Load in current session
jtk completion powershell | Out-String | Invoke-Expression

# Install permanently (add to $PROFILE)
jtk completion powershell >> $PROFILE
```

---

## Development

### Prerequisites

- Go 1.24 or later
- golangci-lint (for linting)

### Build

```bash
make build
```

### Test

```bash
make test
```

### Lint

```bash
make lint
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

Adding a new command or flag? Read these specs first — they're the contract every command in this CLI is held to:

- [internal/cmd/GUARDRAILS.md](internal/cmd/GUARDRAILS.md) — verb language, flag aliases, pagination, mutation safety, boolean conventions, positional-vs-flag rule
- [internal/cmd/OUTPUT_SPEC.md](internal/cmd/OUTPUT_SPEC.md) — list/get/mutation output shapes, `--id` / `--extended` / `--fulltext` semantics, error conventions

## License

MIT License - see [LICENSE](LICENSE) for details.
