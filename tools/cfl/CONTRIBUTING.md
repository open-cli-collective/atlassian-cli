# Contributing to cfl

Thank you for your interest in contributing to cfl!

## Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/open-cli-collective/confluence-cli.git
   cd confluence-cli
   ```

2. Install dependencies:
   ```bash
   make deps
   ```

3. Build and run:
   ```bash
   make build
   ./bin/cfl --version
   ```

## Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-cover

# Run short tests only
make test-short
```

## Code Style

- Run `gofmt` and `goimports` before committing
- Run the linter: `make lint`
- Follow Go conventions and idioms

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new feature
fix: fix a bug
docs: update documentation
test: add tests
refactor: refactor code
ci: update CI configuration
chore: maintenance tasks
```

Examples:
```
feat: add page edit command
fix: handle empty API response in space list
docs: update installation instructions
```

## Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes
4. Run tests: `make test`
5. Run linter: `make lint`
6. Commit with a conventional commit message
7. Push and create a pull request

## Project Structure

```
confluence-cli/
├── cmd/cfl/              # Entry point
├── api/                  # Confluence API client
│   ├── client.go         # HTTP client
│   ├── pages.go          # Page operations
│   ├── spaces.go         # Space operations
│   └── attachments.go    # Attachment operations
├── internal/
│   ├── cmd/              # Command implementations
│   │   ├── init/         # cfl init
│   │   ├── page/         # page commands
│   │   ├── space/        # space commands
│   │   └── root/         # root command
│   ├── config/           # Configuration management
│   └── view/             # Output formatting
├── pkg/
│   └── md/               # Markdown conversion (future)
└── .github/              # GitHub workflows and templates
```

## Questions?

Open an issue or start a discussion on GitHub.
