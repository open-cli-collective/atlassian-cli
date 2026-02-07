# jtk - Jira CLI

A command-line interface for managing Jira Cloud tickets.

## Features

- Manage Jira issues from the command line
- List, create, update, search, and delete issues
- Manage sprints and boards
- Add comments and perform transitions
- Manage attachments
- Manage automation rules
- Search users
- Multiple output formats (table, JSON, plain)
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
| `--output` | `-o` | `table` | Output format: `table`, `json`, `plain` |
| `--no-color` | | `false` | Disable colored output |
| `--verbose` | `-v` | `false` | Enable verbose output |
| `--help` | `-h` | | Show help for command |
| `--version` | | | Show version (root command only) |

---

### `jtk init`

Initialize jtk with guided setup.

```bash
jtk init
jtk init --url https://mycompany.atlassian.net --email user@example.com
```

| Flag | Default | Description |
|------|---------|-------------|
| `--url` | | Jira URL (e.g., `https://mycompany.atlassian.net`) |
| `--email` | | Email address for authentication |
| `--token` | | API token |
| `--no-verify` | `false` | Skip connection verification |

---

### `jtk me`

Show information about the currently authenticated user.

```bash
jtk me
jtk me -o json
```

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

### `jtk issues list`

List issues in a project.

**Aliases:** `jtk issues ls`

```bash
jtk issues list --project MYPROJECT
jtk issues list --project MYPROJECT --sprint current
jtk issues list --project MYPROJECT -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project` | `-p` | | Project key |
| `--sprint` | `-s` | | Filter by sprint: sprint ID or `current` |
| `--max` | `-m` | `50` | Maximum number of issues to return |

---

### `jtk issues get <issue-key>`

Get details of a specific issue.

```bash
jtk issues get PROJ-123
jtk issues get PROJ-123 --full
jtk issues get PROJ-123 -o json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--full` | `false` | Show full description without truncation |

**Arguments:**
- `<issue-key>` - The issue key (e.g., `PROJ-123`) (**required**)

---

### `jtk issues create`

Create a new issue.

```bash
jtk issues create --project MYPROJECT --type Task --summary "Fix login bug"
jtk issues create -p MYPROJECT -t Story -s "Add new feature" --description "Details here"
jtk issues create -p MYPROJECT -s "Custom field issue" --field priority=High --field labels=backend
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project` | `-p` | | Project key (**required**) |
| `--type` | `-t` | `Task` | Issue type: `Task`, `Bug`, `Story`, etc. |
| `--summary` | `-s` | | Issue summary (**required**) |
| `--description` | `-d` | | Issue description |
| `--field` | `-f` | | Additional field in `key=value` format (can be repeated) |

---

### `jtk issues update <issue-key>`

Update an existing issue.

```bash
jtk issues update PROJ-123 --summary "New summary"
jtk issues update PROJ-123 --field priority=High
jtk issues update PROJ-123 --description "Updated description" --field labels=urgent
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--summary` | `-s` | | New summary |
| `--description` | `-d` | | New description |
| `--field` | `-f` | | Field to update in `key=value` format (can be repeated) |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk issues search`

Search issues using JQL.

```bash
jtk issues search --jql "project = MYPROJECT AND status = 'In Progress'"
jtk issues search --jql "assignee = currentUser()" -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--jql` | | | JQL query string (**required**) |
| `--max` | `-m` | `50` | Maximum number of results |

---

### `jtk issues assign <issue-key> [account-id]`

Assign an issue to a user, or unassign it.

```bash
jtk issues assign PROJ-123 5b10ac8d82e05b22cc7d4ef5
jtk issues assign PROJ-123 --unassign
```

| Flag | Default | Description |
|------|---------|-------------|
| `--unassign` | `false` | Remove current assignee |

**Arguments:**
- `<issue-key>` - The issue key (**required**)
- `[account-id]` - The Atlassian account ID (required unless `--unassign`)

---

### `jtk issues delete <issue-key>`

Delete an issue.

```bash
jtk issues delete PROJ-123
jtk issues delete PROJ-123 --force
```

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Skip confirmation prompt |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk issues fields [issue-key]`

List available fields for issues.

```bash
jtk issues fields                    # All fields
jtk issues fields PROJ-123           # Editable fields for a specific issue
jtk issues fields --custom           # Custom fields only
```

| Flag | Default | Description |
|------|---------|-------------|
| `--custom` | `false` | Show only custom fields |

**Arguments:**
- `[issue-key]` - Optional issue key to show editable fields

---

### `jtk issues field-options <field-name-or-id>`

List allowed values for a field.

```bash
jtk issues field-options priority
jtk issues field-options customfield_10001 --issue PROJ-123
```

| Flag | Default | Description |
|------|---------|-------------|
| `--issue` | | Issue key for context-specific options |

**Arguments:**
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
```

| Flag | Default | Description |
|------|---------|-------------|
| `--to-project` | | Target project key (**required**) |
| `--to-type` | (same as source) | Target issue type |
| `--notify` | `true` | Send notifications for the move |
| `--wait` | `true` | Wait for move to complete |

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

### `jtk transitions list <issue-key>`

List available transitions for an issue.

```bash
jtk transitions list PROJ-123
jtk transitions list PROJ-123 --fields
```

| Flag | Default | Description |
|------|---------|-------------|
| `--fields` | `false` | Show required fields for each transition |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk transitions do <issue-key> <transition>`

Perform a transition on an issue.

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

```bash
jtk comments list PROJ-123
jtk comments list PROJ-123 --full
jtk comments list PROJ-123 -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--max` | `-m` | `50` | Maximum number of comments |
| `--full` | | `false` | Show full comment bodies without truncation |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk comments add <issue-key>`

Add a comment to an issue.

```bash
jtk comments add PROJ-123 --body "This is my comment"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--body` | `-b` | | Comment text (**required**) |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk comments delete <issue-key> <comment-id>`

Delete a comment from an issue.

```bash
jtk comments delete PROJ-123 10042
```

**Arguments:**
- `<issue-key>` - The issue key (**required**)
- `<comment-id>` - The comment ID (**required**)

---

### `jtk attachments list <issue-key>`

List attachments on an issue.

**Aliases:** `jtk attachments ls`

```bash
jtk attachments list PROJ-123
jtk attachments list PROJ-123 -o json
```

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk attachments add <issue-key>`

Upload file(s) to an issue.

```bash
jtk attachments add PROJ-123 --file screenshot.png
jtk attachments add PROJ-123 --file doc.pdf --file image.png
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | | File to attach (**required**, can be repeated) |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk attachments get <attachment-id>`

Download an attachment.

**Aliases:** `jtk attachments download`

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

**Aliases:** `jtk attachments rm`

```bash
jtk attachments delete 12345
```

**Arguments:**
- `<attachment-id>` - The attachment ID (**required**)

---

### `jtk sprints list`

List sprints for a board.

```bash
jtk sprints list --board 123
jtk sprints list --board 123 --state active
jtk sprints list --board 123 -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--board` | `-b` | | Board ID (**required**) |
| `--state` | `-s` | | Filter by state: `active`, `closed`, `future` |
| `--max` | `-m` | `50` | Maximum number of results |

---

### `jtk sprints current`

Show the current active sprint.

```bash
jtk sprints current --board 123
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--board` | `-b` | | Board ID (**required**) |

---

### `jtk sprints issues <sprint-id>`

List issues in a sprint.

```bash
jtk sprints issues 456
jtk sprints issues 456 -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--max` | `-m` | `50` | Maximum number of results |

**Arguments:**
- `<sprint-id>` - The sprint ID (**required**)

---

### `jtk sprints add <sprint-id> <issue-key>...`

Move one or more issues to a sprint.

```bash
jtk sprints add 456 PROJ-123
jtk sprints add 456 PROJ-123 PROJ-124 PROJ-125
```

**Arguments:**
- `<sprint-id>` - The sprint ID (**required**)
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
| `--project` | `-p` | | Filter by project key |
| `--max` | `-m` | `50` | Maximum number of results |

---

### `jtk boards get <board-id>`

Get board details.

```bash
jtk boards get 123
```

**Arguments:**
- `<board-id>` - The board ID (**required**)

---

### `jtk users search <query>`

Search for Jira users.

```bash
jtk users search "john"
jtk users search "john" --max 20
```

| Flag | Default | Description |
|------|---------|-------------|
| `--max` | `10` | Maximum number of results |

**Arguments:**
- `<query>` - Search query (matches display name, email, etc.) (**required**)

---

### `jtk automation list`

List automation rules.

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

```bash
jtk automation get 123
jtk automation get 123 --full
```

| Flag | Default | Description |
|------|---------|-------------|
| `--full` | `false` | Show component type details |

**Arguments:**
- `<rule-id>` - The rule ID (**required**)

---

### `jtk automation export <rule-id>`

Export a rule definition as JSON.

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

> Note: Output is always JSON regardless of the `--output` flag.

---

### `jtk automation create`

Create an automation rule from a JSON file.

```bash
jtk automation create --file rule-definition.json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | | Path to JSON file containing the rule definition (**required**) |

> Note: New rules are created in DISABLED state by default.

---

### `jtk automation update <rule-id>`

Update an automation rule from a JSON file.

```bash
jtk automation update 123 --file updated-rule.json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | | Path to JSON file containing the rule definition (**required**) |

**Arguments:**
- `<rule-id>` - The rule ID (**required**)

> Tip: Use `jtk automation export` to get the current definition before editing.

---

### `jtk automation enable <rule-id>`

Enable a disabled automation rule.

```bash
jtk automation enable 123
```

**Arguments:**
- `<rule-id>` - The rule ID (**required**)

---

### `jtk automation disable <rule-id>`

Disable an enabled automation rule.

```bash
jtk automation disable 123
```

**Arguments:**
- `<rule-id>` - The rule ID (**required**)

---

## Configuration

Configuration is stored in `~/.config/jtk/config.json`:

```json
{
  "url": "https://mycompany.atlassian.net",
  "email": "user@example.com",
  "api_token": "your-api-token"
}
```

### Environment Variables

Environment variables override config file values. Variables are checked in order of precedence (first match wins):

| Setting | Precedence (highest to lowest) |
|---------|-------------------------------|
| URL | `JIRA_URL` → `ATLASSIAN_URL` → config file |
| Email | `JIRA_EMAIL` → `ATLASSIAN_EMAIL` → config file |
| API Token | `JIRA_API_TOKEN` → `ATLASSIAN_API_TOKEN` → config file |

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

## License

MIT License - see [LICENSE](LICENSE) for details.
