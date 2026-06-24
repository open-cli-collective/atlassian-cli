# Proof: #434 `cfl` Page View Presenter Migration

## Scope Delivered

- Migrated `page view` from legacy `view` rendering to a presenter/projection
  boundary.
- Added page-view projection logic in
  `tools/cfl/internal/pageview/projection.go` for:
  - metadata preservation
  - storage vs ADF body selection
  - markdown conversion
  - raw/source-faithful body output
  - content-only mode
  - truncation
  - conversion-fallback advisory generation
- Added presenter-owned page-view output in
  `tools/cfl/internal/present/detail.go` through
  `PagePresenter.PresentView(...)`.
- Routed migrated `page view` output through `cflpresent.Emit(...)` so the
  command now fetches, projects, presents, and emits.
- Preserved `--web` as the explicit direct browser-handoff exception.
- Added exact projection tests, presenter tests, and command stdout/stderr
  tests for default, raw, content-only, no-truncate, and conversion-fallback
  paths.

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
rtk go test ./tools/cfl/... ./shared/...
```

Result:

```text
Go test: 1613 passed in 34 packages
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
rtk proxy go test ./tools/cfl/internal/pageview ./tools/cfl/internal/present ./tools/cfl/internal/cmd/page -run 'TestProject_DefaultStorageMarkdown|TestProject_ContentOnlyRawStorage|TestProject_ADFConversionFallback|TestProject_EmptyContent|TestTruncateContent|TestPagePresenter_PresentView_Default|TestPagePresenter_PresentView_ContentOnlyWithAdvisory|TestRunView_ExactOutput_Default|TestRunView_ExactOutput_ContentOnly|TestRunView_ExactOutput_Raw|TestRunView_ExactOutput_NoTruncate|TestRunView_ExactOutput_ConversionFallback|TestRunView_VersionContentOnly|TestRunView_VersionRaw' -count=1 -v
```

Result:

```text
=== RUN   TestProject_DefaultStorageMarkdown
=== RUN   TestProject_ContentOnlyRawStorage
=== RUN   TestProject_ADFConversionFallback
=== RUN   TestProject_EmptyContent
=== RUN   TestTruncateContent
--- PASS: TestProject_DefaultStorageMarkdown (0.00s)
--- PASS: TestProject_ContentOnlyRawStorage (0.00s)
--- PASS: TestProject_ADFConversionFallback (0.00s)
--- PASS: TestProject_EmptyContent (0.00s)
--- PASS: TestTruncateContent (0.00s)
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/pageview
=== RUN   TestPagePresenter_PresentView_Default
=== RUN   TestPagePresenter_PresentView_ContentOnlyWithAdvisory
--- PASS: TestPagePresenter_PresentView_Default (0.00s)
--- PASS: TestPagePresenter_PresentView_ContentOnlyWithAdvisory (0.00s)
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/present
=== RUN   TestRunView_ExactOutput_Default
=== RUN   TestRunView_ExactOutput_ContentOnly
=== RUN   TestRunView_ExactOutput_Raw
=== RUN   TestRunView_ExactOutput_NoTruncate
=== RUN   TestRunView_ExactOutput_ConversionFallback
=== RUN   TestRunView_VersionContentOnly
=== RUN   TestRunView_VersionRaw
--- PASS: TestRunView_ExactOutput_Default (0.00s)
--- PASS: TestRunView_ExactOutput_ContentOnly (0.00s)
--- PASS: TestRunView_ExactOutput_Raw (0.00s)
--- PASS: TestRunView_ExactOutput_NoTruncate (0.00s)
--- PASS: TestRunView_ExactOutput_ConversionFallback (0.00s)
--- PASS: TestRunView_VersionContentOnly (0.00s)
--- PASS: TestRunView_VersionRaw (0.00s)
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/cmd/page
```

These tests prove:

- exact projection decisions for markdown, raw, ADF fallback, empty content,
  and truncation
- exact presenter-owned `OutputModel` shape for metadata/body and
  advisory/content-only modes
- exact command stdout/stderr split for:
  - default view
  - `--content-only`
  - `--raw`
  - `--content-only --raw --no-truncate`
  - conversion fallback advisory
  - historical version content-only/raw flows

## Deterministic CLI Proof

Live smoke for these commands was not recorded in this proof note because no
stable CI-safe Confluence credentials or scratch page identifiers were
provisioned for the repo-local run:

```bash
bin/cfl --no-color page view $CFL_SMOKE_PAGE_ID
bin/cfl --no-color page view $CFL_SMOKE_PAGE_ID --content-only --no-truncate
bin/cfl --no-color page view $CFL_SMOKE_PAGE_ID --raw --no-truncate
bin/cfl --no-color page view $CFL_SMOKE_PAGE_ID --version $CFL_SMOKE_PAGE_VERSION --content-only --no-truncate
```

Instead, deterministic httptest-backed command execution covered the same
externally visible output shapes:

- default `page view` stdout:
  - `Title: Test Page\nID: 12345\nSpace: TEST (ID: 98765)\nVersion: 3\n\n<converted markdown>\n`
  - stderr: empty
- `page view --content-only` stdout:
  - `<converted markdown>\n`
  - stderr: empty
- `page view --raw` stdout:
  - `Title: Raw Page\nID: 12345\nVersion: 1\n\n<p>Raw HTML Content</p>\n`
  - stderr: empty
- `page view --content-only --raw --no-truncate` stdout:
  - full untruncated raw body plus trailing newline
  - stderr: empty
- conversion-fallback path stdout/stderr:
  - stdout: `{not-json\n`
  - stderr: `(Failed to convert ADF to markdown, showing raw ADF)\n`
- historical version content-only/raw:
  - stdout contains only historical body content with no metadata headers
  - stderr: empty

## Residual Notes

- `--web` remains the explicit direct-output exception for browser handoff.
- This ticket intentionally does not migrate page mutation success output;
  those flows remain in `#435`.
