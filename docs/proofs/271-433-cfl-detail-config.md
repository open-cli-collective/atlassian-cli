# Proof: #433 `cfl` Detail and Config Presenter Migration

## Scope Delivered

- Migrated `space view` from legacy `view` rendering to presenter-owned detail
  output through `cflpresent.Emit(...)`.
- Added a config projection boundary in
  `tools/cfl/internal/config/show_projection.go` so `config show` no longer
  performs command-local env/source formatting.
- Added presenter-owned detail output in
  `tools/cfl/internal/present/detail.go` for:
  - `space view`
  - `config show`
- Preserved secret safety for `config show`: token presence/source only, never
  the token value.
- Added presenter tests, projection tests, and exact command wiring tests for
  the migrated detail/config slice.

## Verification Commands

Executed:

```bash
rtk go test ./tools/cfl/internal/present ./tools/cfl/internal/cmd/space ./tools/cfl/internal/cmd/configcmd ./tools/cfl/internal/config ./tools/cfl/internal/cmd/root ./shared/present
```

Result:

```text
Go test: 185 passed in 6 packages
```

Executed:

```bash
rtk go test ./tools/cfl/... ./shared/...
```

Result:

```text
Go test: 1599 passed in 33 packages
```

## Grep Evidence

Executed:

```bash
rtk sh -lc 'if rtk rg -n "os\.Getenv|GetEnvWithFallback|formatValueWithSource|getValueAndSource|opts\.View\(|view\.ValidateFormat|RenderKeyValue|fmt\.F(print|printf|println)" tools/cfl/internal/cmd/configcmd/show.go tools/cfl/internal/cmd/space/view.go --glob "!**/*_test.go"; then :; else printf "no legacy env/view/direct-output matches in #433 command files\n"; fi'
```

Result:

```text
no legacy env/view/direct-output matches in #433 command files
```

Executed:

```bash
rtk proxy go test ./tools/cfl/internal/present ./tools/cfl/internal/cmd/space ./tools/cfl/internal/cmd/configcmd ./tools/cfl/internal/config ./tools/cfl/internal/cmd/root ./shared/present -run 'TestSpacePresenter_PresentDetail|TestSpacePresenter_PresentDetail_Full|TestConfigShowPresenter_PresentDetail|TestRunView_Table|TestRunView_FullPlain|TestRunShow_ExactOutput|TestRunShow_UnreadableConfigNote|TestProjectShow_EnvOverridesFile|TestProjectShow_FileFallbackAndDefaults|TestProjectShow_KeyringMetadataAndUnreadableConfig' -count=1 -v
```

Result:

```text
=== RUN   TestSpacePresenter_PresentDetail
=== RUN   TestSpacePresenter_PresentDetail_Full
=== RUN   TestConfigShowPresenter_PresentDetail
--- PASS: TestSpacePresenter_PresentDetail (0.00s)
--- PASS: TestConfigShowPresenter_PresentDetail (0.00s)
--- PASS: TestSpacePresenter_PresentDetail_Full (0.00s)
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/present
=== RUN   TestRunView_Table
=== RUN   TestRunView_FullPlain
--- PASS: TestRunView_Table (0.00s)
--- PASS: TestRunView_FullPlain (0.00s)
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/cmd/space
=== RUN   TestRunShow_ExactOutput
=== RUN   TestRunShow_UnreadableConfigNote
--- PASS: TestRunShow_ExactOutput (0.00s)
--- PASS: TestRunShow_UnreadableConfigNote (0.00s)
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/cmd/configcmd
=== RUN   TestProjectShow_EnvOverridesFile
=== RUN   TestProjectShow_FileFallbackAndDefaults
=== RUN   TestProjectShow_KeyringMetadataAndUnreadableConfig
--- PASS: TestProjectShow_EnvOverridesFile (0.00s)
--- PASS: TestProjectShow_FileFallbackAndDefaults (0.00s)
--- PASS: TestProjectShow_KeyringMetadataAndUnreadableConfig (0.00s)
PASS
ok  	github.com/open-cli-collective/confluence-cli/internal/config
```

These tests prove:

- exact `OutputModel` for `space view` default/full
- exact `OutputModel` for `config show` including stderr config-path advisory
- config/env/keyring source resolution occurs outside presenters
- command stdout/stderr wiring is preserved
- token values do not leak through `config show`

## Deterministic CLI Proof

Live smoke for these commands was not recorded in this proof note because no
stable CI-safe Confluence credentials were provisioned for the repo-local run:

```bash
bin/cfl --no-color space view $CFL_SMOKE_SPACE
bin/cfl --no-color --full space view $CFL_SMOKE_SPACE
bin/cfl --no-color config show
```

Instead, deterministic httptest-backed command execution covered the same
externally visible output shapes, with the config-show path using a temporary
XDG config directory plus hermetic keyring setup:

- `space view` default stdout:
  - `Key: TEST\nName: Test Space\nID: 123456\nType: global\n`
- `space view --full -o plain` stdout:
  - `Key: TEST\nName: Test Space\nID: 123456\nType: global\nStatus: current\nDescription: A test space\n`
- `config show` stdout includes exact stable detail rows for:
  - `URL`
  - `Email`
  - `API Token`
  - `Default Space`
  - `Auth Method`
  - `Cloud ID`
  - `Keyring Ref`
  - `Keyring Backend`
- `config show` stderr exact advisory:
  - `\nConfig file: <temp-path>\n`
- unreadable config path adds:
  - `  (file not found or unreadable)\n`

## Residual Notes

- This ticket intentionally excludes `page view`; that remains the dedicated
  design/migration scope of `#434`.
- `config show` retains the existing human-readable source suffix contract
  rather than redesigning its output semantics during this slice.
