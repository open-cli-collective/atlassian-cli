# Winget Package for jira-ticket-cli

This directory contains the Winget manifest templates for distributing jtk on Windows via `winget install OpenCLICollective.jira-ticket-cli`.

## Automated Publishing

Publishing to Winget is automated via GitHub Actions. When a new release tag is pushed, the release workflow copies these templates, substitutes version/checksum placeholders, and uses `wingetcreate submit` to submit a PR to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs).

**Required secret:** `WINGET_GITHUB_TOKEN` - A GitHub PAT with `public_repo` scope, needed to create PRs on microsoft/winget-pkgs.

**Note:** Unlike Chocolatey (direct publish), Winget submissions are PRs that go through Microsoft's automated validation before merging.

## Manifest Structure

```
packaging/winget/
├── OpenCLICollective.jira-ticket-cli.yaml              # Version manifest
├── OpenCLICollective.jira-ticket-cli.installer.yaml    # Installer manifest (URLs, checksums)
├── OpenCLICollective.jira-ticket-cli.locale.en-US.yaml # Locale manifest (descriptions, tags)
└── README.md
```

## How Winget Works

Unlike Chocolatey (which hosts packages on their own feed), Winget manifests live in Microsoft's community repository [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs). Publishing requires submitting a PR to that repo.

## Template Placeholders

The manifest templates use these placeholders that are replaced during CI:

| Placeholder | Replaced with |
|-------------|--------------|
| `0.0.0` | Release version (e.g., `0.1.18`) |
| `CHECKSUM_AMD64_PLACEHOLDER` | SHA256 of the x64 zip |
| `CHECKSUM_ARM64_PLACEHOLDER` | SHA256 of the arm64 zip |

URLs contain `0.0.0` in both the tag path and filename, so the version replacement handles them automatically.

## Publishing a New Version

### Option 1: Manual PR

1. **Get release info:**
   - Download URLs: `https://github.com/open-cli-collective/atlassian-cli/releases/download/jtk-v<VERSION>/jtk_<VERSION>_windows_amd64.zip`
   - SHA256 checksums from `checksums.txt` in the release

2. **Update manifests:**
   - Replace `0.0.0` with the actual version in all three YAML files
   - Replace checksum placeholders with real SHA256 values

3. **Validate manifests:**
   ```powershell
   winget validate --manifest packaging/winget/
   ```

4. **Fork and clone** [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs)

5. **Create folder structure:**
   ```
   manifests/o/OpenCLICollective/jira-ticket-cli/<VERSION>/
   ```

6. **Copy manifests** into the folder

7. **Submit PR** to microsoft/winget-pkgs

### Option 2: Using wingetcreate submit

```powershell
# Install wingetcreate
winget install Microsoft.WingetCreate

# Copy and update templates, then submit
wingetcreate submit --path <manifest-dir> --token <PAT>
```

## Manifest Schema

These manifests use schema version 1.10.0:
- [Version manifest schema](https://aka.ms/winget-manifest.version.1.10.0.schema.json)
- [Installer manifest schema](https://aka.ms/winget-manifest.installer.1.10.0.schema.json)
- [Locale manifest schema](https://aka.ms/winget-manifest.defaultLocale.1.10.0.schema.json)

## Installer Type

This package uses:
- `InstallerType: zip` - Our releases are zip archives
- `NestedInstallerType: portable` - Contains a standalone executable
- `PortableCommandAlias: jtk` - Command users type to invoke the tool

Winget extracts the zip, places `jtk.exe` in a managed location, and creates the command alias.

## Architecture Support

| Architecture | Installer URL Pattern |
|--------------|----------------------|
| x64 | `jtk_<VERSION>_windows_amd64.zip` |
| arm64 | `jtk_<VERSION>_windows_arm64.zip` |

## After Approval

Once the PR is merged to microsoft/winget-pkgs, users can install with:
```powershell
winget install OpenCLICollective.jira-ticket-cli
```

## References

- [Winget Manifest Documentation](https://github.com/microsoft/winget-pkgs/tree/master/doc/manifest)
- [Submit packages to Windows Package Manager](https://learn.microsoft.com/en-us/windows/package-manager/package/repository)
- [wingetcreate tool](https://github.com/microsoft/winget-create)
