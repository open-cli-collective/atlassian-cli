# Chocolatey Package for jira-ticket-cli

This directory contains the Chocolatey package definition for distributing jira-ticket-cli on Windows.

## Automated Publishing

Publishing to Chocolatey is automated via GitHub Actions. When a new release tag is pushed, the release workflow:

1. Builds binaries with GoReleaser
2. Packs the Chocolatey package with the release version
3. Pushes to the Chocolatey Community Repository

**Required secret:** `CHOCOLATEY_API_KEY` must be configured in repository settings.

## Package Structure

```
packaging/chocolatey/
├── jira-ticket-cli.nuspec       # Package manifest
├── tools/
│   ├── chocolateyInstall.ps1    # Downloads and installs from GitHub Releases
│   └── chocolateyUninstall.ps1  # Cleanup script
└── README.md
```

## Local Testing

### Prerequisites

- Windows with [Chocolatey installed](https://chocolatey.org/install)
- PowerShell (Admin)

### Build the Package

```powershell
cd packaging/chocolatey

# Update version in nuspec to match a real release (e.g., 0.1.0)
# Then pack:
choco pack
```

This creates `jira-ticket-cli.<version>.nupkg`.

### Install Locally

```powershell
# Install from local package
choco install jira-ticket-cli -s . --force

# Verify
jira-ticket-cli --version

# Uninstall
choco uninstall jira-ticket-cli
```

## Publishing to Chocolatey Community Repository

### First-Time Setup

1. Create an account at https://community.chocolatey.org
2. Get your API key from https://community.chocolatey.org/account
3. Configure your API key:
   ```powershell
   choco apikey --key <your-api-key> --source https://push.chocolatey.org/
   ```

### Publishing a New Version

1. Update the `<version>` in `jira-ticket-cli.nuspec` to match the GitHub release
2. Pack the package:
   ```powershell
   choco pack
   ```
3. Push to Chocolatey:
   ```powershell
   choco push jira-ticket-cli.<version>.nupkg --source https://push.chocolatey.org/
   ```

### Moderation Process

- New packages go through moderation (typically 1-3 days)
- Automated checks verify the package downloads correctly
- Human moderators review the package
- Status updates are sent via email

## Architecture Support

The install script automatically detects Windows architecture:

| Architecture | Download |
|--------------|----------|
| ARM64 | `jira-ticket-cli_<version>_windows_arm64.zip` |
| x64 | `jira-ticket-cli_<version>_windows_amd64.zip` |
| x86 | Not supported (error) |

## Manual Retry Workflow

If automated publishing fails (e.g., moderation issues), use the manual workflow:

1. Go to Actions → "Publish to Chocolatey"
2. Click "Run workflow"
3. Enter the version (e.g., `0.1.0`)
4. Click "Run workflow"
