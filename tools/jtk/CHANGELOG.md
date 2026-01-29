# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **Binary renamed to `jtk`** - The CLI binary is now `jtk` (short for jira-ticket-cli). Install via `brew install jira-ticket-cli`, run with `jtk`. ([#41](https://github.com/open-cli-collective/jira-ticket-cli/pull/41))
- Module path migrated to `github.com/open-cli-collective/jira-ticket-cli` ([#39](https://github.com/open-cli-collective/jira-ticket-cli/pull/39))

### Added

- `jtk issues field-options` command to list allowed values for select fields ([#36](https://github.com/open-cli-collective/jira-ticket-cli/pull/36))
- `jtk issues types` command to list valid issue types per project ([#22](https://github.com/open-cli-collective/jira-ticket-cli/pull/22))
- `jtk users search` command for finding account IDs by name/email ([#34](https://github.com/open-cli-collective/jira-ticket-cli/pull/34))
- Show required fields for transitions in `jtk transitions list` ([#35](https://github.com/open-cli-collective/jira-ticket-cli/pull/35))
- Include custom fields in issue JSON output ([#37](https://github.com/open-cli-collective/jira-ticket-cli/pull/37))

### Fixed

- Show user display name instead of account ID in assign command output ([#33](https://github.com/open-cli-collective/jira-ticket-cli/pull/33))
- Convert number and textarea fields to correct API format when updating issues ([#32](https://github.com/open-cli-collective/jira-ticket-cli/pull/32))
