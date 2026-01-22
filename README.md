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

### Homebrew (macOS)

```bash
brew tap open-cli-collective/tap
brew install --cask jira-ticket-cli
```

> **Note:** Homebrew installation will be available after the first release.

### Go Install

```bash
go install github.com/open-cli-collective/jira-ticket-cli/cmd/jira-ticket-cli@latest
```

### Binary Download

Download the latest release from the [Releases page](https://github.com/open-cli-collective/jira-ticket-cli/releases).

## Quick Start

### 1. Configure jira-ticket-cli

```bash
jira-ticket-cli config set \
  --domain mycompany \
  --email user@example.com \
  --token YOUR_API_TOKEN
```

Get your API token from: https://id.atlassian.com/manage-profile/security/api-tokens

### 2. List Issues

```bash
jira-ticket-cli issues list --project MYPROJECT
```

### 3. Get Issue Details

```bash
jira-ticket-cli issues get PROJ-123
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

### `jira-ticket-cli config set`

Configure Jira credentials.

```bash
jira-ticket-cli config set --domain mycompany --email user@example.com --token YOUR_TOKEN
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--domain` | `-d` | | Jira domain (e.g., `mycompany` for mycompany.atlassian.net) |
| `--email` | `-e` | | Your Atlassian email |
| `--token` | `-t` | | Your API token |

---

### `jira-ticket-cli issues list`

List issues in a project.

**Aliases:** `jira-ticket-cli issues ls`

```bash
jira-ticket-cli issues list --project MYPROJECT
jira-ticket-cli issues list --project MYPROJECT --sprint current
jira-ticket-cli issues list --project MYPROJECT -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project` | `-p` | | Project key (**required**) |
| `--sprint` | `-s` | | Filter by sprint: sprint ID or `current` |
| `--limit` | `-l` | `50` | Maximum number of issues to return |

---

### `jira-ticket-cli issues get <issue-key>`

Get details of a specific issue.

```bash
jira-ticket-cli issues get PROJ-123
jira-ticket-cli issues get PROJ-123 -o json
```

**Arguments:**
- `<issue-key>` - The issue key (e.g., `PROJ-123`) (**required**)

---

### `jira-ticket-cli issues create`

Create a new issue.

```bash
jira-ticket-cli issues create --project MYPROJECT --type Task --summary "Fix login bug"
jira-ticket-cli issues create -p MYPROJECT -t Story -s "Add new feature" --description "Details here"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project` | `-p` | | Project key (**required**) |
| `--type` | `-t` | | Issue type: `Task`, `Bug`, `Story`, etc. (**required**) |
| `--summary` | `-s` | | Issue summary (**required**) |
| `--description` | `-d` | | Issue description |

---

### `jira-ticket-cli issues update <issue-key>`

Update an existing issue.

```bash
jira-ticket-cli issues update PROJ-123 --summary "New summary"
jira-ticket-cli issues update PROJ-123 --field priority=High
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--summary` | `-s` | | New summary |
| `--description` | `-d` | | New description |
| `--field` | `-f` | | Field to update in `key=value` format (can be repeated) |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jira-ticket-cli issues search`

Search issues using JQL.

```bash
jira-ticket-cli issues search --jql "project = MYPROJECT AND status = 'In Progress'"
jira-ticket-cli issues search --jql "assignee = currentUser()" -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--jql` | `-j` | | JQL query string (**required**) |
| `--limit` | `-l` | `50` | Maximum number of results |

---

### `jira-ticket-cli issues assign <issue-key> <account-id>`

Assign an issue to a user.

```bash
jira-ticket-cli issues assign PROJ-123 5b10ac8d82e05b22cc7d4ef5
```

**Arguments:**
- `<issue-key>` - The issue key (**required**)
- `<account-id>` - The Atlassian account ID (**required**)

---

### `jira-ticket-cli issues fields [issue-key]`

List available fields for issues.

```bash
jira-ticket-cli issues fields
jira-ticket-cli issues fields PROJ-123  # editable fields for specific issue
```

**Arguments:**
- `[issue-key]` - Optional issue key to show editable fields

---

### `jira-ticket-cli transitions list <issue-key>`

List available transitions for an issue.

```bash
jira-ticket-cli transitions list PROJ-123
```

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jira-ticket-cli transitions do <issue-key> <transition>`

Perform a transition on an issue.

```bash
jira-ticket-cli transitions do PROJ-123 "In Progress"
jira-ticket-cli transitions do PROJ-123 "Done"
```

**Arguments:**
- `<issue-key>` - The issue key (**required**)
- `<transition>` - Transition name or ID (**required**)

---

### `jira-ticket-cli comments list <issue-key>`

List comments on an issue.

```bash
jira-ticket-cli comments list PROJ-123
jira-ticket-cli comments list PROJ-123 -o json
```

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jira-ticket-cli comments add <issue-key>`

Add a comment to an issue.

```bash
jira-ticket-cli comments add PROJ-123 --body "This is my comment"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--body` | `-b` | | Comment text (**required**) |

**Arguments:**
- `<issue-key>` - The issue key (**required**)

---

### `jira-ticket-cli sprints list`

List sprints for a board.

```bash
jira-ticket-cli sprints list --board 123
jira-ticket-cli sprints list --board 123 -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--board` | `-b` | | Board ID (**required**) |

---

### `jira-ticket-cli sprints current`

Show the current active sprint.

```bash
jira-ticket-cli sprints current --board 123
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--board` | `-b` | | Board ID (**required**) |

---

### `jira-ticket-cli sprints issues <sprint-id>`

List issues in a sprint.

```bash
jira-ticket-cli sprints issues 456
jira-ticket-cli sprints issues 456 -o json
```

**Arguments:**
- `<sprint-id>` - The sprint ID (**required**)

---

### `jira-ticket-cli boards list`

List boards.

```bash
jira-ticket-cli boards list
jira-ticket-cli boards list --project MYPROJECT
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project` | `-p` | | Filter by project key |

---

### `jira-ticket-cli boards get <board-id>`

Get board details.

```bash
jira-ticket-cli boards get 123
```

**Arguments:**
- `<board-id>` - The board ID (**required**)

---

## Configuration

Configuration is stored in `~/.config/jira-ticket-cli/config.yml`:

```yaml
domain: mycompany
email: user@example.com
api_token: your-api-token
```

### Environment Variables

Environment variables override config file values:

| Variable | Description |
|----------|-------------|
| `JIRA_DOMAIN` | Jira domain (without `.atlassian.net`) |
| `JIRA_EMAIL` | Your Atlassian email |
| `JIRA_API_TOKEN` | Your API token |

---

## Shell Completion

jira-ticket-cli supports tab completion for bash, zsh, fish, and PowerShell.

### Bash

```bash
# Load in current session
source <(jira-ticket-cli completion bash)

# Install permanently (Linux)
jira-ticket-cli completion bash | sudo tee /etc/bash_completion.d/jira-ticket-cli > /dev/null

# Install permanently (macOS with Homebrew)
jira-ticket-cli completion bash > $(brew --prefix)/etc/bash_completion.d/jira-ticket-cli
```

### Zsh

```bash
# Load in current session
source <(jira-ticket-cli completion zsh)

# Install permanently
mkdir -p ~/.zsh/completions
jira-ticket-cli completion zsh > ~/.zsh/completions/_jira-ticket-cli

# Add to ~/.zshrc if not already present:
# fpath=(~/.zsh/completions $fpath)
# autoload -Uz compinit && compinit
```

### Fish

```bash
# Load in current session
jira-ticket-cli completion fish | source

# Install permanently
jira-ticket-cli completion fish > ~/.config/fish/completions/jira-ticket-cli.fish
```

### PowerShell

```powershell
# Load in current session
jira-ticket-cli completion powershell | Out-String | Invoke-Expression

# Install permanently (add to $PROFILE)
jira-ticket-cli completion powershell >> $PROFILE
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
