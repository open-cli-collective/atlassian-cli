# Proof: #434 `cfl` Page View Presenter Migration

## Scope Delivered

- Migrated `page view` from legacy `view` rendering to a presenter/projection
  boundary.
- Added page-view projection logic in
  `tools/cfl/internal/pageview/projection.go` for:
  - metadata preservation
  - storage vs ADF body selection
  - markdown conversion
  - raw/source-faithful body selection
  - typed conversion-fallback facts
  - typed truncation facts
  - content-only mode
- Moved empty-content wording, conversion-fallback wording, and truncation
  trailer wording into `PagePresenter.PresentView(...)` in
  `tools/cfl/internal/present/detail.go`.
- Routed migrated `page view` output through `cflpresent.Emit(...)` so the
  command now fetches, projects, presents, and emits.
- Preserved `--web` as the explicit direct browser-handoff exception.
- Added exact projection tests, presenter tests, command stdout/stderr tests,
  and live CLI transcript proof for default, raw, content-only, truncation,
  conversion-fallback, and historical version paths.

## Verification Commands

Executed:

```bash
rtk go test ./tools/cfl/internal/present ./tools/cfl/internal/cmd/page ./tools/cfl/internal/pageview ./tools/cfl/internal/cmd/root ./shared/present
```

Result:

```text
Go test: 256 passed in 5 packages
```

Executed:

```bash
rtk go test ./tools/cfl/internal/pageview ./tools/cfl/internal/present ./tools/cfl/internal/cmd/page
```

Result:

```text
Go test: 198 passed in 3 packages
```

Executed:

```bash
rtk go test ./tools/cfl/... ./shared/...
```

Result:

```text
Go test: 1613 passed in 34 packages
```

Executed:

```bash
rtk golangci-lint run ./tools/cfl/...
```

Result:

```text
golangci-lint: No issues found
```

Executed:

```bash
rtk proxy git diff --check
```

Result: no output, no whitespace or patch-format errors.

## Grep Evidence

Executed:

```bash
rtk sh -lc 'if rtk rg -n "view\.ValidateFormat|opts\.View\(|RenderKeyValue|fmt\.F(print|printf|println)\(v\.Out|fmt\.F(print|printf|println)\(opts\.(Stdout|Stderr)" tools/cfl/internal/cmd/page/view.go --glob "!**/*_test.go"; then :; else printf "no legacy view/direct-output matches in page/view.go\n"; fi'
```

Result:

```text
no legacy view/direct-output matches in page/view.go
```

Executed:

```bash
rtk proxy go test ./tools/cfl/internal/pageview ./tools/cfl/internal/present ./tools/cfl/internal/cmd/page -run 'TestProject_DefaultStorageMarkdown|TestProject_ContentOnlyRawStorage|TestProject_ADFConversionFallback|TestProject_StorageConversionFallback|TestProject_EmptyContent|TestTruncateContent|TestPagePresenter_PresentView_Default|TestPagePresenter_PresentView_ContentOnlyWithAdvisory|TestPagePresenter_PresentView_EmptyAndTruncated|TestRunView_ExactOutput_Default|TestRunView_ExactOutput_ContentOnly|TestRunView_ExactOutput_Raw|TestRunView_ExactOutput_RawContentOnly_NoTruncate|TestRunView_ExactOutput_DefaultMarkdown_NoTruncate|TestRunView_ExactOutput_ConversionFallback|TestRunView_ExactOutput_StorageConversionFallback_Default|TestRunView_ExactOutput_VersionDefault|TestRunView_VersionContentOnly|TestRunView_VersionRaw|TestRunView_EmptyContent|TestRunView_ContentOnly_EmptyBody|TestRunView_ShowMacros' -count=1 -v
```

Result:

```text
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/pageview
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/present
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/cmd/page
```

These tests prove:

- projection returns typed facts for markdown/raw selection, fallback kind,
  content presence, and truncation
- presenter owns empty-state wording, conversion-fallback wording, and the
  truncation trailer
- exact command stdout/stderr split for:
  - default view
  - `--content-only`
  - `--raw`
  - `--content-only --raw --no-truncate`
  - default markdown `--no-truncate`
  - ADF conversion fallback
  - storage conversion fallback
  - empty-content default and content-only output
  - `--show-macros`
  - historical version default view
  - historical version content-only
  - historical version raw

## Live CLI Proof

Live smoke was recorded with real Confluence credentials on this machine using
environment-variable auth to avoid interactive keyring access:

```bash
TOKEN=$(security find-generic-password -s atlassian-cli -a default/api_token -w)
ATLASSIAN_API_TOKEN="$TOKEN" ./bin/cfl --no-color page list --space MONIT --limit 5
ATLASSIAN_API_TOKEN="$TOKEN" ./bin/cfl --no-color page view 2237596283
ATLASSIAN_API_TOKEN="$TOKEN" ./bin/cfl --no-color page view 2237596283 --content-only --no-truncate
ATLASSIAN_API_TOKEN="$TOKEN" ./bin/cfl --no-color page view 2237596283 --raw --no-truncate
ATLASSIAN_API_TOKEN="$TOKEN" ./bin/cfl --no-color page view 2237596283 --version 1 --content-only --no-truncate
```

Page selection:

- `page list --space MONIT --limit 5` returned page `2237596283` (`2021 Recap`)
- that page currently renders as `Version: 2`, so it provides both current and
  historical content without creating scratch content

Cleanup:

- no temporary Confluence pages were created
- no cleanup was required because the proof reused an existing page

Observed output shape:

- `page list --space MONIT --limit 5` stdout:

```text
ID          TITLE                           STATUS
163989      Monit                           current
2219376652  2021-Q4 Direction Setting       current
2237596283  2021 Recap                      current
2238054401  2022-Q1 Direction Setting       current
2239627329  Product Sales Mock Ups Desired  current
```

- `page list --space MONIT --limit 5` stderr:

```text
(showing first 5 results, use --limit to see more)
```

- `page view 2237596283` stdout begins:

```text
Title: 2021 Recap
ID: 2237596283
Space: MONIT (ID: 163988)
Version: 2

Monit Investors: as we close out an exciting year, I wanted to take a moment to cover recent highlights and look forward to what Monit has on tap for 2022.
```

- `page view 2237596283` stderr: empty

- `page view 2237596283 --content-only --no-truncate` stdout begins directly
  with body content and has no metadata header:

```text
Monit Investors: as we close out an exciting year, I wanted to take a moment to cover recent highlights and look forward to what Monit has on tap for 2022.
```

- `page view 2237596283 --content-only --no-truncate` stderr: empty

- `page view 2237596283 --raw --no-truncate` stdout begins:

```text
Title: 2021 Recap
ID: 2237596283
Space: MONIT (ID: 163988)
Version: 2

<p>Monit Investors: as we close out an exciting year, I wanted to take a moment to cover recent highlights and look forward to what Monit has on tap for 2022.</p>
```

- `page view 2237596283 --raw --no-truncate` stderr: empty

- `page view 2237596283 --version 1 --content-only --no-truncate` stdout begins
  directly with historical body content and differs from the current version
  body:

```text
Monit Investors: as we close out an exciting year, I wanted to take a moment to cover recent highlights and look forward to what Monit has on tap for 2022.
```

Observed historical difference in the live transcript:

- version 1 body says `The team now includes 11 employees`
- current version body says `The team now includes 12 employees`

- `page view 2237596283 --version 1 --content-only --no-truncate` stderr: empty

## Residual Notes

- `--web` remains the explicit direct-output exception for browser handoff.
- This ticket intentionally does not migrate page mutation success output;
  those flows remain in `#435`.
