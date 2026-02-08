# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `projects create`, `update`, `delete`, `restore`, `types` commands for full project management ([#106](https://github.com/open-cli-collective/atlassian-cli/pull/106))
- `automation create` command to create rules from JSON files ([#79](https://github.com/open-cli-collective/atlassian-cli/pull/79))
- `automation enable`, `disable`, `update`, `export` commands for full automation rule management ([#76](https://github.com/open-cli-collective/atlassian-cli/pull/76))
- `--full` flag on `issues get` and `comments list` to show full content without truncation ([#72](https://github.com/open-cli-collective/atlassian-cli/pull/72))
- `init` command for guided setup wizard ([#48](https://github.com/open-cli-collective/atlassian-cli/pull/48))
- `issues move` command to move issues between projects ([#51](https://github.com/open-cli-collective/atlassian-cli/pull/51))
- `attachments` commands: list, add, get, delete ([#50](https://github.com/open-cli-collective/atlassian-cli/pull/50))
- Wiki markup detection and automatic conversion to ADF ([#49](https://github.com/open-cli-collective/atlassian-cli/pull/49))
- `issues field-options` command to list allowed values for select fields ([#36](https://github.com/open-cli-collective/jira-ticket-cli/pull/36))
- `issues types` command to list valid issue types per project ([#22](https://github.com/open-cli-collective/jira-ticket-cli/pull/22))
- `users search` command for finding account IDs by name/email ([#34](https://github.com/open-cli-collective/jira-ticket-cli/pull/34))
- Show required fields for transitions in `transitions list --fields` ([#35](https://github.com/open-cli-collective/jira-ticket-cli/pull/35))
- Include custom fields in issue JSON output ([#37](https://github.com/open-cli-collective/jira-ticket-cli/pull/37))

### Changed

- Consolidated markdown-to-ADF conversion into shared package ([#74](https://github.com/open-cli-collective/atlassian-cli/pull/74))
- Improved init/config UX with huh forms and --force flag on clear ([#55](https://github.com/open-cli-collective/atlassian-cli/pull/55))
- **Binary renamed to `jtk`** - The CLI binary is now `jtk` (short for jira-ticket-cli). Install via `brew install jira-ticket-cli`, run with `jtk`. ([#41](https://github.com/open-cli-collective/jira-ticket-cli/pull/41))
- Module path migrated to `github.com/open-cli-collective/jira-ticket-cli` ([#39](https://github.com/open-cli-collective/jira-ticket-cli/pull/39))

### Fixed

- `config show -o json` no longer appends trailing plain text after JSON body ([#124](https://github.com/open-cli-collective/atlassian-cli/pull/124))
- `projects create` success message uses the input name instead of the empty API response name ([#121](https://github.com/open-cli-collective/atlassian-cli/pull/121))
- `ProjectDetail.ID` uses `json.Number` to handle numeric API responses ([#116](https://github.com/open-cli-collective/atlassian-cli/pull/116))
- Automation rule state endpoint uses correct payload format for Jira Cloud ([#110](https://github.com/open-cli-collective/atlassian-cli/pull/110))
- `automation create` strips server-assigned fields and parses `ruleUuid` correctly ([#109](https://github.com/open-cli-collective/atlassian-cli/pull/109))
- `--field` flag handles structured fields (e.g., `priority=High`) in create and update ([#107](https://github.com/open-cli-collective/atlassian-cli/pull/107))
- Validate file input before making network calls ([#86](https://github.com/open-cli-collective/atlassian-cli/pull/86))
- Automation API parsing aligned with Jira Cloud response format ([#87](https://github.com/open-cli-collective/atlassian-cli/pull/87))
- Show user display name instead of account ID in assign command output ([#33](https://github.com/open-cli-collective/jira-ticket-cli/pull/33))
- Convert number and textarea fields to correct API format when updating issues ([#32](https://github.com/open-cli-collective/jira-ticket-cli/pull/32))
