# Atlassian CLI Tools

Unified CLI tools for Atlassian Cloud products.

> **Consolidation note:** This repo is the home for both `jtk` (Jira CLI) and `cfl` (Confluence CLI), previously maintained as separate repos ([jira-ticket-cli](https://github.com/open-cli-collective/jira-ticket-cli) and [confluence-cli](https://github.com/open-cli-collective/confluence-cli)). Those repos are now archived. All development happens here.

## Table of Contents

- [Tools](#tools)
- [Installation](#installation)
  - [macOS](#macos)
  - [Windows](#windows)
  - [Linux](#linux)
  - [Build from Source](#build-from-source)
- [Migrating from the Old Repos](#migrating-from-the-old-repos)
- [Getting Started](#getting-started)
  - [Configuration](#configuration)
  - [Authentication](#authentication)
  - [Shared Credentials](#shared-credentials)
- [jtk - Jira CLI](#jtk---jira-cli)
- [cfl - Confluence CLI](#cfl---confluence-cli)
- [Shell Completion](#shell-completion)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Tools

| Tool | Description | Full Documentation |
|------|-------------|-------------------|
| `jtk` | Jira Cloud CLI for issues, sprints, and boards | [jtk README](tools/jtk/README.md) |
| `cfl` | Confluence Cloud CLI for markdown-first page management | [cfl README](tools/cfl/README.md) |

## Installation

### macOS

**Homebrew (recommended)**

```bash
# Install jtk (Jira CLI)
brew install open-cli-collective/tap/jira-ticket-cli

# Install cfl (Confluence CLI)
brew install open-cli-collective/tap/cfl

# Upgrade to latest
brew upgrade jira-ticket-cli cfl
```

> **Note:** If `brew upgrade` doesn't pick up a new version, your local tap may be stale. Run `git -C $(brew --repository open-cli-collective/tap) pull` to refresh it, then retry the upgrade.

**Binary download**

Download from the [Releases](https://github.com/open-cli-collective/atlassian-cli/releases) page for your architecture (Intel or Apple Silicon).

---

### Windows

**Chocolatey**

```powershell
# Install jtk
choco install jira-ticket-cli

# Install cfl
choco install confluence-cli
```

**Winget**

```powershell
# Install jtk
winget install OpenCLICollective.jira-ticket-cli

# Install cfl
winget install OpenCLICollective.cfl
```

**Binary download**

Download from the [Releases](https://github.com/open-cli-collective/atlassian-cli/releases) page for your architecture.

---

### Linux

**Debian/Ubuntu (APT)**

```bash
# Download the .deb package from Releases
sudo dpkg -i jtk_*.deb
sudo dpkg -i cfl_*.deb
```

**RPM-based (Fedora, RHEL, etc.)**

```bash
# Download the .rpm package from Releases
sudo rpm -i jtk-*.rpm
sudo rpm -i cfl-*.rpm
```

**Binary download**

Download from the [Releases](https://github.com/open-cli-collective/atlassian-cli/releases) page for your architecture (amd64 or arm64).

---

### Build from Source

Requires Go 1.21 or later.

```bash
git clone https://github.com/open-cli-collective/atlassian-cli.git
cd atlassian-cli
make build
# Binaries are in bin/
```

## Migrating from the Old Repos

If you previously installed from `jira-ticket-cli` or `confluence-cli`:

**Homebrew users:**

```bash
# If you installed via the 'jtk' cask (legacy)
brew uninstall jtk
brew install open-cli-collective/tap/jira-ticket-cli

# If you installed via 'jira-ticket-cli', you're already set — just upgrade
brew upgrade jira-ticket-cli

# If brew upgrade says "already installed" but you're on an old version,
# refresh your local tap first:
git -C $(brew --repository open-cli-collective/tap) pull
brew upgrade jira-ticket-cli
```

**GitHub release users:**

All future releases are published here. Update your bookmarks/scripts to download from:
https://github.com/open-cli-collective/atlassian-cli/releases

Your existing configuration (`~/.config/jtk/` and `~/.config/cfl/`) is unchanged — no reconfiguration needed.

## Getting Started

### Configuration

Both tools support interactive setup:

```bash
# Configure Jira credentials
jtk init

# Configure Confluence credentials
cfl init
```

The init wizards will prompt for:
- Atlassian URL (e.g., `https://mycompany.atlassian.net`)
- Email address
- API token

Configuration is stored in:
- jtk: `~/.config/jtk/config.json`
- cfl: `~/.config/cfl/config.yml`

### Authentication

Both tools use Atlassian API tokens for authentication. To create a token:

1. Go to [Atlassian Account Settings](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Click "Create API token"
3. Give it a descriptive label (e.g., "CLI Tools")
4. Copy the token and use it during `init` or set it as an environment variable

### Shared Credentials

Use `ATLASSIAN_*` environment variables for shared authentication across both tools:

| Variable | Description |
|----------|-------------|
| `ATLASSIAN_URL` | Base URL (e.g., `https://mycompany.atlassian.net`) |
| `ATLASSIAN_EMAIL` | Your Atlassian account email |
| `ATLASSIAN_API_TOKEN` | Your API token |

Tool-specific variables take precedence:
- jtk: `JIRA_URL`, `JIRA_EMAIL`, `JIRA_API_TOKEN`
- cfl: `CFL_URL`, `CFL_EMAIL`, `CFL_API_TOKEN`

**Example:**

```bash
# Set shared credentials (used by both tools)
export ATLASSIAN_URL="https://mycompany.atlassian.net"
export ATLASSIAN_EMAIL="you@example.com"
export ATLASSIAN_API_TOKEN="your-api-token"

# Now both tools work without additional configuration
jtk issues list --project PROJ
cfl page list --space DEV
```

## jtk - Jira CLI

Manage Jira issues, sprints, and boards from the command line.

```bash
# List issues in a project
jtk issues list --project PROJ

# Create an issue
jtk issues create --project PROJ --type Task --summary "Fix bug"

# View issue details
jtk issues get PROJ-123

# Search with JQL
jtk issues search "project = PROJ AND status = 'In Progress'"

# List transitions and move issue
jtk transitions list PROJ-123
jtk transitions do PROJ-123 "Done"

# Manage comments
jtk comments list PROJ-123
jtk comments add PROJ-123 "This is fixed"

# View current sprint
jtk sprints current --board 123

# Manage attachments
jtk attachments list PROJ-123
jtk attachments add PROJ-123 --file screenshot.png
jtk attachments get 12345 --output ./downloads/
```

**Full documentation:** [tools/jtk/README.md](tools/jtk/README.md)

## cfl - Confluence CLI

Manage Confluence pages with a markdown-first workflow.

```bash
# List pages in a space
cfl page list --space DEV

# View page in markdown
cfl page view 123456

# Create page from markdown
cfl page create --space DEV --title "New Page" --file content.md

# Edit page in your editor
cfl page edit 123456

# Search pages
cfl page search "my search query"
cfl page search --cql "space = DEV AND type = page"

# Copy a page
cfl page copy 123456 --title "Copy of Page"

# List spaces
cfl space list

# Manage attachments
cfl attachment list 123456
cfl attachment upload 123456 --file diagram.png
cfl attachment download 123456 image.png --output ./
```

**Full documentation:** [tools/cfl/README.md](tools/cfl/README.md)

## Shell Completion

Both tools support shell completion for bash, zsh, and fish.

**Bash:**

```bash
# jtk
jtk completion bash > /etc/bash_completion.d/jtk

# cfl
cfl completion bash > /etc/bash_completion.d/cfl
```

**Zsh:**

```bash
# jtk
jtk completion zsh > "${fpath[1]}/_jtk"

# cfl
cfl completion zsh > "${fpath[1]}/_cfl"
```

**Fish:**

```bash
# jtk
jtk completion fish > ~/.config/fish/completions/jtk.fish

# cfl
cfl completion fish > ~/.config/fish/completions/cfl.fish
```

## Development

This is a Go workspace monorepo. Both tools can be built and tested together.

```bash
# Build both tools
make build

# Run all tests
make test

# Run linter
make lint

# Build, test, and lint
make all

# Build individual tools
make build-jtk
make build-cfl

# Run tests for a specific tool
go test ./tools/jtk/...
go test ./tools/cfl/...
```

### Project Structure

```
atlassian-cli/
├── go.work              # Go workspace file
├── Makefile             # Build automation
├── shared/              # Shared packages (auth, client, errors)
└── tools/
    ├── cfl/             # Confluence CLI
    │   ├── api/         # API client
    │   ├── cmd/cfl/     # Entry point
    │   └── internal/    # Commands and config
    └── jtk/             # Jira CLI
        ├── api/         # API client
        ├── cmd/jtk/     # Entry point
        └── internal/    # Commands and config
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Make your changes with tests
4. Run `make all` to verify
5. Commit with conventional commit messages (`feat:`, `fix:`, etc.)
6. Push and create a pull request

See the individual tool CLAUDE.md files for detailed development guidance:
- [tools/jtk/CLAUDE.md](tools/jtk/CLAUDE.md)
- [tools/cfl/CLAUDE.md](tools/cfl/CLAUDE.md)

## License

MIT License. See [LICENSE](LICENSE) for details.
