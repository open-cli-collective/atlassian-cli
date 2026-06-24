# Proof: #432 `cfl` List/Search Presenter Migration

## Scope Delivered

- Added presenter-owned list/search output models at
  `tools/cfl/internal/present/lists.go` for:
  - `space list`
  - `page list`
  - `page history list`
  - `search`
  - `attachment list`
- Routed migrated commands through `tools/cfl/internal/present.Emit(...)` so
  commands fetch data, call presenters, and emit.
- Moved `-o plain` list rendering onto the shared pure renderer by adding
  `StyleHumanPlain` handling in:
  - `shared/present/model.go`
  - `shared/present/render.go`
  - `tools/cfl/internal/cmd/root/root.go`
- Added presenter tests that assert exact `present.OutputModel` sections,
  headers, row values, empty states, pagination hints, and ID-only output.
- Added command tests that assert exact stdout/stderr for representative
  `table` and `plain` outputs in every migrated command package.

## Verification Commands

Executed:

```bash
rtk go test ./tools/cfl/internal/present ./tools/cfl/internal/cmd/space ./tools/cfl/internal/cmd/page ./tools/cfl/internal/cmd/search ./tools/cfl/internal/cmd/attachment ./tools/cfl/internal/cmd/root ./shared/present
```

Result:

```text
Go test: 367 passed in 7 packages
```

Executed:

```bash
rtk go test ./tools/cfl/... ./shared/...
```

Result:

```text
Go test: 1596 passed in 33 packages
```

Executed:

```bash
rtk git diff --check
```

Result: no output, no whitespace or patch-format errors.

## Grep Evidence

Executed:

```bash
rtk sh -lc 'if rtk rg -n "v\\.Table|v\\.RenderKeyValue|v\\.Success|view\\.ValidateFormat|opts\\.View\\(|fmt\\.F(print|printf|println)\\(" tools/cfl/internal/cmd/space/list.go tools/cfl/internal/cmd/page/list.go tools/cfl/internal/cmd/page/history.go tools/cfl/internal/cmd/search/search.go tools/cfl/internal/cmd/attachment/list.go tools/cfl/internal/present/lists.go --glob "!**/*_test.go"; then :; else printf "no legacy view/output matches in migrated #432 files\n"; fi'
```

Result:

```text
no legacy view/output matches in migrated #432 files
```

Executed:

```bash
rtk proxy go test ./tools/cfl/internal/present -run 'TestSpacePresenter_PresentList|TestSpacePresenter_PresentEmpty|TestPagePresenter_PresentList|TestPagePresenter_PresentEmpty|TestPageHistoryPresenter_PresentListAndIDs|TestSearchPresenter_PresentList|TestAttachmentPresenter_PresentListAndEmpty|TestListPresenterHelpers' -count=1 -v
```

Result:

```text
=== RUN   TestSpacePresenter_PresentList
=== RUN   TestSpacePresenter_PresentEmpty
=== RUN   TestPagePresenter_PresentList
=== RUN   TestPagePresenter_PresentEmpty
=== RUN   TestPageHistoryPresenter_PresentListAndIDs
=== RUN   TestSearchPresenter_PresentList
=== RUN   TestAttachmentPresenter_PresentListAndEmpty
=== RUN   TestListPresenterHelpers
--- PASS: TestSpacePresenter_PresentList (0.00s)
--- PASS: TestPagePresenter_PresentEmpty (0.00s)
--- PASS: TestPagePresenter_PresentList (0.00s)
--- PASS: TestSpacePresenter_PresentEmpty (0.00s)
--- PASS: TestPageHistoryPresenter_PresentListAndIDs (0.00s)
--- PASS: TestListPresenterHelpers (0.00s)
--- PASS: TestAttachmentPresenter_PresentListAndEmpty (0.00s)
--- PASS: TestSearchPresenter_PresentList (0.00s)
PASS
```

These tests assert exact presenter-owned:

- table headers and row cells
- empty-state message wording and stream routing
- fallback pagination wording when the API reports more results but no cursor is parseable
- `--full` field expansion
- pagination wording
- history `--id` stdout behavior
- helper extraction/formatting used by the commands

## Deterministic CLI Proof

Live smoke for the following user-facing commands was not recorded in this
proof note because no stable CI-safe Confluence credentials were provisioned
for the repo-local run:

```bash
bin/cfl --no-color space list --limit 2
bin/cfl --no-color -o plain space list --limit 2
bin/cfl --no-color page list --space $CFL_SMOKE_SPACE --limit 2
bin/cfl --no-color page history list $CFL_SMOKE_PAGE_ID --limit 2
bin/cfl --no-color search --type page --limit 1
bin/cfl --no-color attachment list --page $CFL_SMOKE_PAGE_ID --limit 2
```

Instead, deterministic httptest-backed command execution covered the same
externally visible output shapes:

Executed:

```bash
rtk proxy go test ./tools/cfl/internal/cmd/space ./tools/cfl/internal/cmd/page ./tools/cfl/internal/cmd/search ./tools/cfl/internal/cmd/attachment -run 'TestRunList_PlainOutputExact|TestRunList_TableOutputExact|TestRunList_PageList_PlainFullExact|TestRunList_PageList_TableOutputExact|TestRunHistoryList_PlainOutputExact|TestRunHistoryList_TableOutputExact|TestRunSearch_PlainOutput|TestRunSearch_TableOutputExact|TestRunList_PlainFullExact|TestRunList_TableOutputExact' -count=1 -v
```

Result:

```text
=== RUN   TestRunList_PlainOutputExact
=== RUN   TestRunList_TableOutputExact
--- PASS: TestRunList_PlainOutputExact (0.00s)
--- PASS: TestRunList_TableOutputExact (0.00s)
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/cmd/space
=== RUN   TestRunHistoryList_PlainOutputExact
=== RUN   TestRunHistoryList_TableOutputExact
=== RUN   TestRunList_PageList_PlainFullExact
=== RUN   TestRunList_PageList_TableOutputExact
--- PASS: TestRunHistoryList_TableOutputExact (0.00s)
--- PASS: TestRunHistoryList_PlainOutputExact (0.00s)
--- PASS: TestRunList_PageList_PlainFullExact (0.00s)
--- PASS: TestRunList_PageList_TableOutputExact (0.00s)
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/cmd/page
=== RUN   TestRunSearch_PlainOutput
=== RUN   TestRunSearch_TableOutputExact
--- PASS: TestRunSearch_PlainOutput (0.00s)
--- PASS: TestRunSearch_TableOutputExact (0.00s)
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/cmd/search
=== RUN   TestRunList_PlainFullExact
=== RUN   TestRunList_TableOutputExact
--- PASS: TestRunList_PlainFullExact (0.00s)
--- PASS: TestRunList_TableOutputExact (0.00s)
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/cmd/attachment
```

Those tests assert these exact stdout/stderr contracts:

- `space list` table:
  - stdout: `ID      KEY  TYPE    NAME\n123456  DEV  global  Development\n`
  - stderr: empty
- `space list --full -o plain`:
  - stdout: `ID\tKEY\tTYPE\tSTATUS\tNAME\n123456\tDEV\tglobal\tcurrent\tDevelopment\n`
  - stderr: empty
- `space list -o plain`:
  - stdout: `ID\tKEY\tTYPE\tNAME\n123456\tDEV\tglobal\tDevelopment\n`
  - stderr: `Next page: cfl space list --cursor "cursor-123"\n`
- `page list --full -o plain`:
  - stdout: `ID\tTITLE\tSTATUS\tVERSION\tPARENT ID\n11111\tPage One\tcurrent\tv1\t999\n`
  - stderr: `(showing first 1 results, use --limit to see more)\n`
- `page list` table:
  - stdout: `ID     TITLE     STATUS\n11111  Page One  current\n`
  - stderr: empty
- `page history list` table:
  - stdout: `VERSION  CREATED               AUTHOR\n15       2024-01-02T03:04:05Z  author-1\n`
  - stderr: empty
- `page history list -o plain`:
  - stdout: `VERSION\tCREATED\tAUTHOR\n15\t2024-01-02T03:04:05Z\tauthor-1\n`
  - stderr: `Next page: cfl page history list 12345 --cursor "cursor-out"\n`
- `search` table:
  - stdout: `ID     TYPE  SPACE  TITLE\n12345  page  DEV    Test Page\n`
  - stderr: empty
- `search -o plain`:
  - stdout: `ID\tTYPE\tSPACE\tTITLE\n12345\tpage\tDEV\tTest Page\n`
  - stderr: `(showing 1 of 2 results, use --limit to see more)\n`
- `search --full -o plain`:
  - stdout: `ID\tTYPE\tSPACE\tTITLE\tMODIFIED\tURL\n12345\tpage\tDEV\tTest Page\t2024-02-03\t/wiki/spaces/DEV/pages/12345\n`
  - stderr: empty
- `attachment list` table:
  - stdout: `ID    TITLE    MEDIA TYPE       FILE SIZE\natt1  doc.pdf  application/pdf  1.0 KB\n`
  - stderr: empty
- `attachment list --full -o plain`:
  - stdout: `ID\tTITLE\tMEDIA TYPE\tFILE SIZE\tSTATUS\tCOMMENT\natt1\tdoc.pdf\tapplication/pdf\t1.0 KB\tcurrent\tlatest\n`
  - stderr: `(showing first 1 results, use --limit to see more)\n`

## Residual Notes

- This ticket intentionally does not cover `page view`, mutation success
  output, or remaining diagnostics/advisories. Those are split into follow-on
  child issues `#433` through `#437`.
- The `attachment list --full` columns shipped here as
  `STATUS` and `COMMENT` so `--full` is a real presenter-owned expansion rather
  than a silent no-op.
