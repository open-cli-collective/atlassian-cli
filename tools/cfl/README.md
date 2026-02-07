# cfl - Confluence CLI

A command-line interface for Atlassian Confluence Cloud, inspired by [jira-cli](https://github.com/ankitpokhrel/jira-cli).

## Features

- Manage Confluence pages from the command line
- **Markdown-first**: Write and view pages in markdown, auto-converted to/from Confluence format
- List and browse spaces
- Create, view, edit, copy, and delete pages
- **Search content** using CQL (Confluence Query Language)
- Upload, download, list, and delete attachments
- Find unused (orphaned) attachments
- Multiple output formats (table, JSON, plain)
- Open pages in browser

## Installation

### macOS

**Homebrew (recommended)**

```bash
brew install open-cli-collective/tap/cfl
```

> Note: This installs from our third-party tap. The formula is not yet in Homebrew core.

**Binary download**

Download from the [Releases page](https://github.com/open-cli-collective/atlassian-cli/releases) - available for Intel and Apple Silicon.

---

### Windows

**Chocolatey**

```powershell
choco install confluence-cli
```

**Winget**

```powershell
winget install OpenCLICollective.cfl
```

**Binary download**

Download from the [Releases page](https://github.com/open-cli-collective/atlassian-cli/releases) - available for x64 and ARM64.

---

### Linux

**APT (Debian/Ubuntu) - Recommended**

```bash
# Add the GPG key
curl -fsSL https://open-cli-collective.github.io/linux-packages/keys/gpg.asc | sudo gpg --dearmor -o /usr/share/keyrings/open-cli-collective.gpg

# Add the repository
echo "deb [signed-by=/usr/share/keyrings/open-cli-collective.gpg] https://open-cli-collective.github.io/linux-packages/apt stable main" | sudo tee /etc/apt/sources.list.d/open-cli-collective.list

# Update and install
sudo apt update
sudo apt install cfl
```

> Note: This is our third-party APT repository, not the official Debian/Ubuntu repos.

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
sudo dnf install cfl
```

> Note: This is our third-party RPM repository, not the official Fedora/RHEL repos.

**Homebrew**

```bash
brew install open-cli-collective/tap/cfl
```

**Snap**

```bash
sudo snap install ocli-confluence
sudo snap alias ocli-confluence cfl  # Optional: create 'cfl' alias
```

> Note: Snap package is pending store approval. Check availability at https://snapcraft.io/ocli-confluence

**Binary download**

Download `.deb`, `.rpm`, or `.tar.gz` from the [Releases page](https://github.com/open-cli-collective/atlassian-cli/releases) - available for x64 and ARM64.

```bash
# Direct .deb install
curl -LO https://github.com/open-cli-collective/atlassian-cli/releases/latest/download/cfl_VERSION_linux_amd64.deb
sudo dpkg -i cfl_VERSION_linux_amd64.deb

# Direct .rpm install
curl -LO https://github.com/open-cli-collective/atlassian-cli/releases/latest/download/cfl-VERSION.x86_64.rpm
sudo rpm -i cfl-VERSION.x86_64.rpm
```

---

### From Source

**Go install**

```bash
go install github.com/open-cli-collective/confluence-cli/cmd/cfl@latest
```

Requires Go 1.24 or later.

## Quick Start

### 1. Configure cfl

```bash
cfl init
```

This will prompt you for:
- Your Confluence URL (e.g., `https://mycompany.atlassian.net`)
- Your email address
- An API token

**To generate an API token:**
1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
2. Click "Create API token"
3. Copy the token (it won't be shown again)

### 2. List Spaces

```bash
cfl space list
```

### 3. List Pages in a Space

```bash
cfl page list --space DEV
```

### 4. View a Page

```bash
cfl page view 12345
```

### 5. Create a Page

```bash
cfl page create --space DEV --title "My New Page"
```

---

## Command Reference

### Global Flags

These flags are available on all commands:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | `-c` | `~/.config/cfl/config.yml` | Path to config file |
| `--output` | `-o` | `table` | Output format: `table`, `json`, `plain` |
| `--no-color` | | `false` | Disable colored output |
| `--help` | `-h` | | Show help for command |
| `--version` | `-v` | | Show version (root command only) |

---

### `cfl init`

Initialize cfl with your Confluence Cloud credentials.

```bash
cfl init
cfl init --url https://mycompany.atlassian.net
cfl init --url https://mycompany.atlassian.net --email you@example.com
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--url` | | | Pre-populate Confluence URL |
| `--email` | | | Pre-populate email address |
| `--no-verify` | | `false` | Skip connection verification |

---

### `cfl space list`

List Confluence spaces.

**Aliases:** `cfl space ls`

```bash
cfl space list
cfl space list --type global
cfl space list --type personal
cfl space list -l 50 -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--limit` | `-l` | `25` | Maximum number of spaces to return |
| `--type` | `-t` | | Filter by type: `global` or `personal` |

---

### `cfl page list`

List pages in a space.

**Aliases:** `cfl page ls`

```bash
cfl page list --space DEV
cfl page list -s DEV -l 50
cfl page list -s DEV --status archived
cfl page list -s DEV -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--space` | `-s` | (from config) | Space key (**required** if no default) |
| `--limit` | `-l` | `25` | Maximum number of pages to return |
| `--status` | | `current` | Filter by status: `current`, `archived`, `trashed` |

---

### `cfl page view <page-id>`

View a Confluence page. **Content is displayed as markdown by default.**

```bash
cfl page view 12345
cfl page view 12345 --raw
cfl page view 12345 --web
cfl page view 12345 -o json
cfl page view 12345 --content-only             # Output only content (no headers)
cfl page view 12345 --show-macros --content-only | cfl page edit 12345 --legacy  # Roundtrip with macros
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--raw` | | `false` | Show raw Confluence storage format (XHTML) instead of markdown |
| `--web` | `-w` | `false` | Open page in browser instead of displaying |
| `--full` | | `false` | Show full content without truncation |
| `--show-macros` | | `false` | Show Confluence macro placeholders (e.g., `[TOC]`) instead of stripping them |
| `--content-only` | | `false` | Output only page content (no Title/ID/Version headers); implies `--full` |

**Arguments:**
- `<page-id>` - The page ID (**required**)

---

### `cfl page create`

Create a new Confluence page.

Content can be provided via:
- `--file` flag to read from a file
- Standard input (pipe content)
- Interactive editor (default)

**Markdown is the default format.** Content is automatically converted to Confluence storage format.

```bash
# Open markdown editor
cfl page create --space DEV --title "My Page"

# Create from markdown file
cfl page create -s DEV -t "My Page" --file content.md

# Create from markdown stdin
echo "# Hello World" | cfl page create -s DEV -t "My Page"

# Create from XHTML file (auto-detected by extension)
cfl page create -s DEV -t "My Page" --file content.html

# Create from XHTML stdin (disable markdown conversion)
echo "<p>Hello</p>" | cfl page create -s DEV -t "My Page" --no-markdown

# Create as child of another page
cfl page create -s DEV -t "Child Page" --parent 12345

# Create using legacy storage format (for compatibility with legacy editor pages)
cfl page create -s DEV -t "Legacy Page" --file content.md --legacy
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--space` | `-s` | (from config) | Space key (**required** if no default) |
| `--title` | `-t` | | Page title (**required**) |
| `--parent` | `-p` | | Parent page ID (for nested pages) |
| `--file` | `-f` | | Read content from file |
| `--editor` | | `false` | Force open in $EDITOR |
| `--no-markdown` | | `false` | Disable markdown conversion (use raw XHTML) |
| `--legacy` | | `false` | Use legacy storage format instead of cloud editor (ADF) |

**Format detection:**
- `.md`, `.markdown` files → markdown (converted to XHTML)
- `.html`, `.xhtml`, `.htm` files → XHTML (used as-is)
- stdin, editor → markdown by default (use `--no-markdown` for XHTML)

---

### `cfl page edit <page-id>`

Edit an existing Confluence page.

Content can be provided via:
- `--file` flag to read from a file
- Standard input (pipe content)
- Interactive editor (default, opens with existing content)

**Markdown is the default format.** Content is automatically converted to Confluence storage format.

```bash
# Open editor with existing page content
cfl page edit 12345

# Update page content from markdown file
cfl page edit 12345 --file updated-content.md

# Update page content from stdin
echo "# Updated Content" | cfl page edit 12345

# Update only the page title
cfl page edit 12345 --title "New Title"

# Move page to a new parent
cfl page edit 12345 --parent 67890

# Move page and rename in one command
cfl page edit 12345 --parent 67890 --title "New Title"

# Edit using legacy storage format (for pages created in legacy editor)
cfl page edit 12345 --file content.md --legacy
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--title` | `-t` | | New page title (keeps existing if not specified) |
| `--parent` | `-p` | | Move page to new parent page ID |
| `--file` | `-f` | | Read content from file |
| `--editor` | | `false` | Force open in $EDITOR |
| `--no-markdown` | | `false` | Disable markdown conversion (use raw XHTML) |
| `--legacy` | | `false` | Use legacy storage format instead of cloud editor (ADF) |

**Arguments:**
- `<page-id>` - The page ID (**required**)

---

### `cfl page copy <page-id>`

Create a copy of a Confluence page with a new title.

```bash
# Copy a page with a new title (same space)
cfl page copy 12345 --title "Copy of My Page"

# Copy to a different space
cfl page copy 12345 --title "My Page" --space OTHERSPACE

# Copy without attachments (faster for large pages)
cfl page copy 12345 --title "Lightweight Copy" --no-attachments

# Copy without labels
cfl page copy 12345 --title "Fresh Copy" --no-labels
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--title` | `-t` | | Title for the copied page (**required**) |
| `--space` | `-s` | (same as source) | Destination space key |
| `--no-attachments` | | `false` | Don't copy attachments |
| `--no-labels` | | `false` | Don't copy labels |

**Arguments:**
- `<page-id>` - The source page ID (**required**)

---

### `cfl page delete <page-id>`

Delete a Confluence page.

```bash
cfl page delete 12345
cfl page delete 12345 --force
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | `-f` | `false` | Skip confirmation prompt |

**Arguments:**
- `<page-id>` - The page ID (**required**)

---

### `cfl search [query]`

Search for pages, blog posts, attachments, and comments across Confluence.

Uses Confluence Query Language (CQL) under the hood. Convenient flags handle common
filters, or use `--cql` for advanced queries.

```bash
# Full-text search
cfl search "deployment guide"

# Search within a space
cfl search "api docs" --space DEV

# Find only pages
cfl search "meeting notes" --type page

# Filter by label
cfl search --label documentation

# Search by title
cfl search --title "Release Notes"

# Combine filters
cfl search "kubernetes" --space DEV --type page --label infrastructure

# Raw CQL for power users (find pages modified in last 7 days)
cfl search --cql "type=page AND space=DEV AND lastModified > now('-7d')"

# Output as JSON for scripting
cfl search "config" -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--cql` | | | Raw CQL query (advanced) |
| `--space` | `-s` | (from config) | Filter by space key |
| `--type` | `-t` | | Content type: `page`, `blogpost`, `attachment`, `comment` |
| `--title` | | | Filter by title (contains) |
| `--label` | | | Filter by label |
| `--limit` | `-l` | `25` | Maximum number of results |

**Arguments:**
- `[query]` - Full-text search terms (optional if using filters)

**CQL Reference:**
Common CQL operators for `--cql`:
- `=` exact match: `type=page`
- `~` contains/fuzzy: `title ~ "meeting"`
- `AND`, `OR`, `NOT` for combining
- Date functions: `lastModified > now('-7d')`
- [Full CQL documentation](https://developer.atlassian.com/cloud/confluence/advanced-searching-using-cql/)

---

### `cfl config`

Manage cfl configuration.

#### `cfl config show`

Display current configuration with masked credentials and source info.

```bash
cfl config show
```

#### `cfl config test`

Test connectivity with current configuration. Verifies URL reachability, credential validity, and API access.

```bash
cfl config test
```

#### `cfl config clear`

Remove the stored configuration file.

```bash
cfl config clear
cfl config clear --force
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | `-f` | `false` | Skip confirmation prompt |

> Note: Environment variables (`CFL_*`, `ATLASSIAN_*`) will still be used if set.

---

### `cfl attachment list`

List attachments on a page.

**Aliases:** `cfl attachment ls`, `cfl att list`

```bash
cfl attachment list --page 12345
cfl attachment list -p 12345 -l 50
cfl attachment list -p 12345 -o json

# List unused (orphaned) attachments not referenced in page content
cfl attachment list --page 12345 --unused
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--page` | `-p` | | Page ID (**required**) |
| `--limit` | `-l` | `25` | Maximum number of attachments to return |
| `--unused` | | `false` | Show only attachments not referenced in page content |

---

### `cfl attachment upload`

Upload a file as an attachment to a page.

```bash
cfl attachment upload --page 12345 --file document.pdf
cfl attachment upload -p 12345 -f image.png -m "Screenshot"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--page` | `-p` | | Page ID (**required**) |
| `--file` | `-f` | | File to upload (**required**) |
| `--comment` | `-m` | | Comment for the attachment |

---

### `cfl attachment download <attachment-id>`

Download an attachment.

```bash
cfl attachment download abc123
cfl attachment download abc123 -O document.pdf
cfl attachment download abc123 -O existing.pdf --force
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output-file` | `-O` | (original filename) | Output file path |
| `--force` | `-f` | `false` | Overwrite existing file without warning |

**Arguments:**
- `<attachment-id>` - The attachment ID (**required**)

---

### `cfl attachment delete <attachment-id>`

Delete an attachment.

```bash
cfl attachment delete att123
cfl attachment delete att123 --force
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | `-f` | `false` | Skip confirmation prompt |

**Arguments:**
- `<attachment-id>` - The attachment ID (**required**)

---

## Confluence Macro Support

cfl supports roundtrip editing of common Confluence macros using bracket syntax. When viewing pages with `--show-macros`, macros are displayed as readable placeholders that can be edited and re-uploaded.

### Supported Macros

| Macro | Syntax | Description |
|-------|--------|-------------|
| TOC | `[TOC]` or `[TOC maxLevel=3]` | Table of contents |
| Info | `[INFO]content[/INFO]` | Blue info panel |
| Warning | `[WARNING]content[/WARNING]` | Yellow warning panel |
| Note | `[NOTE]content[/NOTE]` | Yellow note panel |
| Tip | `[TIP]content[/TIP]` | Green tip panel |
| Expand | `[EXPAND title="..."]content[/EXPAND]` | Collapsible section |

### Viewing Pages with Macros

By default, macros are stripped from markdown output. Use `--show-macros` to preserve them:

```bash
# Without --show-macros: macros are hidden
cfl page view 12345
# Output: just the page content, no macro markers

# With --show-macros: macros appear as bracket syntax
cfl page view 12345 --show-macros
# Output includes: [TOC maxLevel=3], [INFO]...[/INFO], etc.
```

### Creating Pages with Macros

Use bracket syntax in your markdown:

```bash
# Create a page with TOC
echo '[TOC]

# Introduction
Some content here.

# Details
More content.' | cfl page create -s DEV -t "My Doc" --legacy

# Create a page with info panel
echo '[INFO]
This is important information that readers should know.
[/INFO]

Regular content follows.' | cfl page create -s DEV -t "My Guide" --legacy
```

### Roundtrip Editing

View a page with macros, edit it, and push changes back:

```bash
# Export page with macros to file (use --content-only to exclude metadata headers)
cfl page view 12345 --show-macros --content-only > page.md

# Edit the file (macros appear as [TOC], [INFO]...[/INFO], etc.)
vim page.md

# Push changes back (macros are converted to Confluence format)
cat page.md | cfl page edit 12345 --legacy

# Or pipe directly for quick edits
cfl page view 12345 --show-macros --content-only | cfl page edit 12345 --legacy
```

### Panel Macro Parameters

Panel macros support a `title` parameter:

```markdown
[WARNING title="Security Notice"]
Do not share your API tokens.
[/WARNING]
```

Values with spaces must be quoted. The content between open and close tags is converted as markdown.

---

## Configuration

Configuration is stored in `~/.config/cfl/config.yml`:

```yaml
url: https://mycompany.atlassian.net/wiki
email: you@example.com
api_token: your-api-token
default_space: DEV
output_format: table
```

### Environment Variables

Environment variables override config file values. Variables are checked in order of precedence (first match wins):

| Setting | Precedence (highest to lowest) |
|---------|-------------------------------|
| URL | `CFL_URL` → `ATLASSIAN_URL` → config file |
| Email | `CFL_EMAIL` → `ATLASSIAN_EMAIL` → config file |
| API Token | `CFL_API_TOKEN` → `ATLASSIAN_API_TOKEN` → config file |
| Default Space | `CFL_DEFAULT_SPACE` → config file |

**Shared credentials:** If you use both `cfl` and `jtk` (Jira CLI), set `ATLASSIAN_*` variables once:

```bash
export ATLASSIAN_URL=https://mycompany.atlassian.net
export ATLASSIAN_EMAIL=user@example.com
export ATLASSIAN_API_TOKEN=your-api-token
```

**Per-tool override:** Use `CFL_*` to override for Confluence specifically:

```bash
export ATLASSIAN_EMAIL=user@example.com
export ATLASSIAN_API_TOKEN=your-api-token
export CFL_URL=https://confluence.internal.corp.com  # Different URL for Confluence
```

---

## Output Formats

Use `--output` or `-o` to change output format:

```bash
cfl space list -o table  # Default: human-readable table
cfl space list -o json   # JSON for scripting/automation
cfl space list -o plain  # Tab-separated for piping to other tools
```

---

## Shell Completion

cfl supports tab completion for bash, zsh, fish, and PowerShell.

### Bash

```bash
# Load in current session
source <(cfl completion bash)

# Install permanently (Linux)
cfl completion bash | sudo tee /etc/bash_completion.d/cfl > /dev/null

# Install permanently (macOS with Homebrew)
cfl completion bash > $(brew --prefix)/etc/bash_completion.d/cfl
```

### Zsh

```bash
# Load in current session
source <(cfl completion zsh)

# Install permanently
mkdir -p ~/.zsh/completions
cfl completion zsh > ~/.zsh/completions/_cfl

# Add to ~/.zshrc if not already present:
# fpath=(~/.zsh/completions $fpath)
# autoload -Uz compinit && compinit
```

### Fish

```bash
# Load in current session
cfl completion fish | source

# Install permanently
cfl completion fish > ~/.config/fish/completions/cfl.fish
```

### PowerShell

```powershell
# Load in current session
cfl completion powershell | Out-String | Invoke-Expression

# Install permanently (add to $PROFILE)
cfl completion powershell >> $PROFILE
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
