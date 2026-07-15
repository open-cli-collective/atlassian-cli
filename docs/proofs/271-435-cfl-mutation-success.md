# Proof: #435 `cfl` Mutation Success Presenter Migration

## Scope

This proof covers presenter-owned mutation success output for:

- `space create`, `space update`, `space delete`
- `page create`, `page edit`, `page copy`, `page delete`
- `attachment upload`, `attachment download`, `attachment delete`

The migrated commands now:

- orchestrate API/config/prompt flow only
- emit presenter-owned success/cancellation/warning models through
  `tools/cfl/internal/present/mutation.go`
- preserve direct prompt text as an explicit `stderr` exception
- keep follow-up handles and size formatting in presenters, not command bodies

## Verification Commands

Executed:

```bash
rtk go test ./tools/cfl/internal/present ./tools/cfl/internal/cmd/space ./tools/cfl/internal/cmd/page ./tools/cfl/internal/cmd/attachment
```

Result:

```text
Go test: 292 passed in 4 packages
```

Executed:

```bash
rtk go test ./tools/cfl/... ./shared/...
```

Result:

```text
Go test: 1622 passed in 34 packages
```

Executed:

```bash
rtk golangci-lint run ./tools/cfl/internal/present ./tools/cfl/internal/cmd/space ./tools/cfl/internal/cmd/page ./tools/cfl/internal/cmd/attachment
```

Result:

```text
golangci-lint: No issues found
```

Executed:

```bash
rtk proxy git diff --check
```

Result:

```text
<no output>
```

## Grep Gates

Executed:

```bash
rtk rg -n '\bv\.(Success|RenderKeyValue|RenderKeyValues)\b' tools/cfl/internal/cmd/space tools/cfl/internal/cmd/page tools/cfl/internal/cmd/attachment --glob '!**/*_test.go'
rtk rg -n '\bv\.Warning\b' tools/cfl/internal/cmd/page/edit.go --glob '!**/*_test.go'
rtk rg -n 'formatFileSize\(' tools/cfl/internal/cmd/space tools/cfl/internal/cmd/page tools/cfl/internal/cmd/attachment tools/cfl/internal/present --glob '!**/*_test.go'
rtk rg -n 'view\.ValidateFormat|opts\.View\(|github.com/open-cli-collective/atlassian-go/view' tools/cfl/internal/cmd/space tools/cfl/internal/cmd/page tools/cfl/internal/cmd/attachment --glob '!**/*_test.go'
```

Result:

```text
<no matches>
```

These greps prove the migrated mutation files no longer own:

- legacy `v.Success` / `v.RenderKeyValue` success output
- legacy `v.Warning` in `page edit`
- command-local attachment size formatting
- legacy per-command `view.ValidateFormat` / `opts.View()` plumbing

## Test Evidence

Presenter tests in `tools/cfl/internal/present/mutation_test.go` assert exact
`OutputModel` shapes for:

- space create/update/delete
- page create/edit/copy/delete
- attachment upload/download/delete
- deletion cancellation

Command tests now assert exact stdout/stderr for representative migrated paths,
including:

- `space create` success
- `space update` success
- `space delete --force` success
- `space delete` interactive cancel and accept
- `page create` success
- `page copy` success pinned to `OUTPUT_SPEC.md`
- `page edit` success
- `page edit --legacy` warning on `stderr` plus success on `stdout`
- `page delete --force` success
- `page delete` interactive accept/cancel
- `attachment upload` success and local-size fallback
- `attachment download` success and overwrite success
- `attachment delete --force` success
- `attachment delete` interactive accept/cancel

## Live CLI Transcript

Live smoke was recorded against real Confluence credentials using:

```bash
TOKEN=$(security find-generic-password -s atlassian-cli -a default/api_token -w)
```

Transcript directory:

```text
/tmp/cfl-435-live.XaDFat
```

Created and removed identifiers:

- space key: `CFLM165420`
- created space name: `CFL Mutation Scratch 2026-06-24T16:54:20`
- updated space name: `CFL Mutation Scratch 2026-06-24T16:54:20 Updated`
- page id: `3554967565`
- copied page id: `3554738181`
- attachment id: `att3555033098`
- attachment byte count written locally: `21`
- attachment byte count downloaded: `21`

Commands executed:

```bash
./bin/cfl --no-color space create --key CFLM165420 --name "CFL Mutation Scratch 2026-06-24T16:54:20"
./bin/cfl --no-color space update CFLM165420 --name "CFL Mutation Scratch 2026-06-24T16:54:20 Updated"
./bin/cfl --no-color page create --space CFLM165420 --title "Scratch Mutation Page" --file /tmp/cfl-435-live.XaDFat/page-create.md
./bin/cfl --no-color page edit 3554967565 --file /tmp/cfl-435-live.XaDFat/page-edit.md
./bin/cfl --no-color page copy 3554967565 --title "Scratch Mutation Page Copy" --space CFLM165420
./bin/cfl --no-color attachment upload --page 3554967565 --file /tmp/cfl-435-live.XaDFat/attachment.txt
./bin/cfl --no-color attachment download att3555033098 --output-file /tmp/cfl-435-live.XaDFat/downloaded-attachment.txt --force
./bin/cfl --no-color attachment delete att3555033098 --force
./bin/cfl --no-color page delete 3554738181 --force
./bin/cfl --no-color page delete 3554967565 --force
./bin/cfl --no-color space delete CFLM165420 --force
```

Observed stdout/stderr excerpts:

- `space create` stdout:

```text
Created space: CFL Mutation Scratch 2026-06-24T16:54:20
Key: CFLM165420
URL: https://monitproduct.atlassian.net/wiki/spaces/CFLM165420
```

- `space update` stdout:

```text
Updated space: CFL Mutation Scratch 2026-06-24T16:54:20 Updated (CFLM165420)
```

- `page create` stdout:

```text
Created page: Scratch Mutation Page
ID: 3554967565
URL: https://monitproduct.atlassian.net/wiki/spaces/CFLM165420/pages/3554967565/Scratch+Mutation+Page
```

- `page edit` stdout:

```text
Updated page: Scratch Mutation Page
ID: 3554967565
Version: 2
URL: https://monitproduct.atlassian.net/wiki/spaces/CFLM165420/pages/3554967565/Scratch+Mutation+Page
```

- `page copy` stdout:

```text
Copied page: Scratch Mutation Page Copy
ID: 3554738181
Space: CFLM165420
Version: 1
```

- `attachment upload` stdout:

```text
Uploaded: attachment.txt
ID: att3555033098
Title: attachment.txt
Size: 21 B
```

- `attachment download` stdout:

```text
Downloaded: /tmp/cfl-435-live.XaDFat/downloaded-attachment.txt
Size: 21 B
```

- local byte verification:

```text
attachment.txt: 21 bytes
downloaded-attachment.txt: 21 bytes
```

- cleanup stdout:

```text
Deleted attachment: attachment.txt (ID: att3555033098)
Deleted page: Scratch Mutation Page Copy (ID: 3554738181)
Deleted page: Scratch Mutation Page (ID: 3554967565)
Deleted space: CFL Mutation Scratch 2026-06-24T16:54:20 Updated (CFLM165420)
```

Cleanup result:

- scratch attachment removed
- scratch pages removed
- scratch space removed
- no temporary Confluence artifacts remain from the live proof run
