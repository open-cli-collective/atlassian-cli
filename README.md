# cfl - Confluence CLI

A command-line interface for Atlassian Confluence Cloud, inspired by [jira-cli](https://github.com/ankitpokhrel/jira-cli).

## Features

- Manage Confluence pages from the command line
- **Markdown-first**: Write and view pages in markdown, auto-converted to/from Confluence format
- List and browse spaces
- Create, view, and delete pages
- Upload, download, and list attachments
- Multiple output formats (table, JSON, plain)
- Open pages in browser

## Installation

### Homebrew (macOS)

```bash
brew tap rianjs/confluence-cli
brew install --cask cfl
```

### Go Install

```bash
go install github.com/rianjs/confluence-cli/cmd/cfl@latest
```

### Binary Download

Download the latest release from the [Releases page](https://github.com/rianjs/confluence-cli/releases).

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

**Aliases:** `cfl page ls`, `cfl page search`

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
| `--status` | | `current` | Filter by status: `current`, `archived`, `draft` |

---

### `cfl page view <page-id>`

View a Confluence page. **Content is displayed as markdown by default.**

```bash
cfl page view 12345
cfl page view 12345 --raw
cfl page view 12345 --web
cfl page view 12345 -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--raw` | | `false` | Show raw Confluence storage format (XHTML) instead of markdown |
| `--web` | `-w` | `false` | Open page in browser instead of displaying |

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
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--space` | `-s` | (from config) | Space key (**required** if no default) |
| `--title` | `-t` | | Page title (**required**) |
| `--parent` | `-p` | | Parent page ID (for nested pages) |
| `--file` | `-f` | | Read content from file |
| `--editor` | | `false` | Force open in $EDITOR |
| `--no-markdown` | | `false` | Disable markdown conversion (use raw XHTML) |

**Format detection:**
- `.md`, `.markdown` files → markdown (converted to XHTML)
- `.html`, `.xhtml`, `.htm` files → XHTML (used as-is)
- stdin, editor → markdown by default (use `--no-markdown` for XHTML)

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

### `cfl attachment list`

List attachments on a page.

**Aliases:** `cfl attachment ls`, `cfl att list`

```bash
cfl attachment list --page 12345
cfl attachment list -p 12345 -l 50
cfl attachment list -p 12345 -o json
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--page` | `-p` | | Page ID (**required**) |
| `--limit` | `-l` | `25` | Maximum number of attachments to return |

---

### `cfl attachment upload`

Upload a file as an attachment to a page.

```bash
cfl attachment upload --page 12345 --file document.pdf
cfl attachment upload -p 12345 -f image.png --comment "Screenshot"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--page` | `-p` | | Page ID (**required**) |
| `--file` | `-f` | | File to upload (**required**) |
| `--comment` | `-c` | | Comment for the attachment |

---

### `cfl attachment download <attachment-id>`

Download an attachment.

```bash
cfl attachment download abc123
cfl attachment download abc123 --output-file document.pdf
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output-file` | `-o` | (original filename) | Output file path |

**Arguments:**
- `<attachment-id>` - The attachment ID (**required**)

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

Environment variables override config file values:

| Variable | Description |
|----------|-------------|
| `CFL_URL` | Confluence instance URL |
| `CFL_EMAIL` | Your Atlassian email |
| `CFL_API_TOKEN` | Your API token |
| `CFL_DEFAULT_SPACE` | Default space key |

---

## Output Formats

Use `--output` or `-o` to change output format:

```bash
cfl space list -o table  # Default: human-readable table
cfl space list -o json   # JSON for scripting/automation
cfl space list -o plain  # Tab-separated for piping to other tools
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
