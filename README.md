# jira-ticket-cli

A command-line interface for managing Jira Cloud tickets.

## Features

- Manage Jira issues from the command line
- List, create, update, and search issues
- Manage sprints and boards
- Add comments and perform transitions
- Multiple output formats (table, JSON, plain)
- Shell completion for bash, zsh, and fish

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

Download `.deb`, `.rpm`, or `.tar.gz` from the [Releases page](https://github.com/open-cli-collective/jira-ticket-cli/releases) - available for x64 and ARM64.

```bash
# Direct .deb install
curl -LO https://github.com/open-cli-collective/jira-ticket-cli/releases/latest/download/jtk_VERSION_linux_amd64.deb
sudo dpkg -i jtk_VERSION_linux_amd64.deb

# Direct .rpm install
curl -LO https://github.com/open-cli-collective/jira-ticket-cli/releases/latest/download/jtk-VERSION.x86_64.rpm
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
# Jira Cloud
jtk config set \
  --url https://mycompany.atlassian.net \
  --email user@example.com \
  --token YOUR_API_TOKEN

# Self-hosted Jira (Data Center / Server)
jtk config set \
  --url https://jira.internal.corp.com \
  --email user@example.com \
  --token YOUR_API_TOKEN
```

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
| `--config` | `-c` | `~/.config/jira-ticket-cli/config.yml` | Path to config file |
| `--output` | `-o` | `table` | Output format: `table`, `json`, `plain` |
| `--help` | `-h` | | Show help for command |
| `--version` | `-v` | | Show version (root command only) |

---

### `jtk config set`

Configure Jira credentials.

```bash
# Jira Cloud
jtk config set --url https://mycompany.atlassian.net --email user@example.com --token YOUR_TOKEN

# Self-hosted Jira
jtk config set --url https://jira.internal.corp.com --email user@example.com --token YOUR_TOKEN
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--url` | | | Jira URL (e.g., `https://mycompany.atlassian.net` or `https://jira.internal.corp.com`) |
| `--email` | `-e` | | Your Atlassian email |
| `--token` | `-t` | | Your API token |

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
| `--project` | `-p` | | Project key (**required**) |
| `--sprint` | `-s` | | Filter by sprint: sprint ID or `current` |
| `--limit` | `-l` | `50` | Maximum number of issues to return |

---

### `jtk issues get <issue-key>`

Get details of a specific issue.

```bash
jtk issues get PROJ-123
jtk issues get PROJ-123 -o json
```

**Arguments:**
- `<issue-key>` - The issue key (e.g., `PROJ-123`) (**required**)

---

### `jtk issues create`

Create a new issue.

```bash
jtk issues create --project MYPROJECT --type Task --summary "Fix login bug"
jtk issues create -p MYPROJECT -t Story -s "Add new feature" --description "Details here"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project` | `-p` | | Project key (**required**) |
| `--type` | `-t` | | Issue type: `Task`, `Bug`, `Story`, etc. (**required**) |
| `--summary` | `-s` | | Issue summary (**required**) |
| `--description` | `-d` | | Issue description |

---

### `jtk issues update <issue-key>`

Update an existing issue.

```bash
jtk issues update PROJ-123 --summary "New summary"
jtk issues update PROJ-123 --field priority=High
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
| `--jql` | `-j` | | JQL query string (**required**) |
| `--limit` | `-l` | `50` | Maximum number of results |

---

### `jtk issues assign <issue-key> <account-id>`

Assign an issue to a user.

```bash
jtk issues assign PROJ-123 5b10ac8d82e05b22cc7d4ef5
```

**Arguments:**
- `<issue-key>` - The issue key (**required**)
- `<account-id>` - The Atlassian account ID (**required**)

---

### `jtk issues fields [issue-key]`

List available fields for issues.

```bash
jtk issues fields
jtk issues fields PROJ-123  # editable fields for specific issue
```

**Arguments:**
- `[issue-key]` - Optional issue key to show editable fields

---

### `jtk transitions list <issue-key>`

List available transitions for an issue.

```bash
jtk transitions list PROJ-123
```

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jtk transitions do <issue-key> <transition>`

Perform a transition on an issue.

```bash
jtk transitions do PROJ-123 "In Progress"
jtk transitions do PROJ-123 "Done"
```

**Arguments:**
- `<issue-key>` - The issue key (**required**)
- `<transition>` - Transition name or ID (**required**)

---

### `jtk comments list <issue-key>`

List comments on an issue.

```bash
jtk comments list PROJ-123
jtk comments list PROJ-123 -o json
```

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

### `jtk sprints list`

List sprints for a board.

```bash
jtk sprints list --board 123
jtk sprints list --board 123 -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--board` | `-b` | | Board ID (**required**) |

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

**Arguments:**
- `<sprint-id>` - The sprint ID (**required**)

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

---

### `jtk boards get <board-id>`

Get board details.

```bash
jtk boards get 123
```

**Arguments:**
- `<board-id>` - The board ID (**required**)

---

## Configuration

Configuration is stored in `~/.config/jira-ticket-cli/config.json`:

```json
{
  "url": "https://mycompany.atlassian.net",
  "email": "user@example.com",
  "api_token": "your-api-token"
}
```

### Environment Variables

Environment variables override config file values:

| Variable | Description |
|----------|-------------|
| `JIRA_URL` | Full Jira URL (e.g., `https://mycompany.atlassian.net` or `https://jira.internal.corp.com`) |
| `JIRA_EMAIL` | Your Atlassian email |
| `JIRA_API_TOKEN` | Your API token |

> **Note:** The legacy `JIRA_DOMAIN` environment variable is still supported for backwards compatibility but is deprecated. Use `JIRA_URL` instead.

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

- Go 1.22 or later
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
