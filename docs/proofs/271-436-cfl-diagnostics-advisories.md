# Proof: #436 `cfl` Diagnostics and Advisory Presenter Migration

## Scope

This proof covers presenter-owned diagnostic/advisory/status output for:

- `config test`
- `config clear`

The migrated commands now:

- orchestrate config, API, keyring, and prompt control flow only
- emit presenter-owned diagnostic/status models through
  `tools/cfl/internal/present/config.go`
- preserve `config test` progress timing through an explicit no-newline
  presenter message section
- keep one-shot confirmation prompt text as an explicit direct `stderr`
  exception
- keep `cfl init` wizard output outside this ticket

Pagination advisories and the `page edit --legacy` warning were already moved
to presenters in earlier child tickets. `docs/proofs/271-435-cfl-mutation-success.md`
contains the exact command and live proof for the legacy-editor advisory.

## Verification Commands

Executed:

```bash
rtk go test ./tools/cfl/internal/cmd/configcmd ./tools/cfl/internal/present
```

Result:

```text
Go test: 57 passed in 2 packages
```

Executed:

```bash
rtk go test ./tools/cfl/internal/present ./tools/cfl/internal/cmd/configcmd ./tools/cfl/internal/cmd/page ./tools/cfl/internal/cmd/root ./shared/present
```

Result:

```text
Go test: 287 passed in 5 packages
```

Executed:

```bash
rtk golangci-lint run ./tools/cfl/internal/present ./tools/cfl/internal/cmd/configcmd ./tools/cfl/internal/cmd/page ./tools/cfl/internal/cmd/root
```

Result:

```text
golangci-lint: No issues found
```

Executed:

```bash
rtk go test ./tools/cfl/... ./shared/...
```

Result:

```text
Go test: 1637 passed in 34 packages
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
rtk rg -n 'fmt\.Fprint|fmt\.Fprintf|fmt\.Fprintln' tools/cfl/internal/cmd/configcmd/test.go tools/cfl/internal/cmd/configcmd/clear.go
```

Result:

```text
tools/cfl/internal/cmd/configcmd/clear.go:87:			_, _ = fmt.Fprint(opts.Stderr, promptText+" [y/N]: ")
```

This is the documented prompt exception.

Executed:

```bash
rtk rg -n 'Testing connection|Troubleshooting|Authenticated as|No stored API token|Cancelled\. Nothing was cleared|Removed key|Removed the shared keyring bundle|Warning: this is the SHARED token|Note: .*environment' tools/cfl/internal/cmd/configcmd/test.go tools/cfl/internal/cmd/configcmd/clear.go
```

Result:

```text
tools/cfl/internal/cmd/configcmd/clear.go:43:Note: CFL_API_TOKEN / ATLASSIAN_API_TOKEN environment variables still
```

This remaining match is command help text, not runtime diagnostic/status
output.

## Test Evidence

Presenter tests in `tools/cfl/internal/present/config_test.go` assert exact
`OutputModel` messages and stderr routing for:

- config-test success with user details
- config-test no-newline progress
- config-test fallback success when user lookup fails after connectivity works
- config-test failure/troubleshooting
- clear planned default action
- clear planned `--all` action, including plaintext cleanup and keyring
  unavailable notes
- clear cancelled
- clear success
- clear `--all` success
- no stored token
- environment override notes

Command tests now assert exact stdout/stderr for:

- `config test` success
- `config test` fallback success
- `config test` connection failure
- `config clear` no-op
- `config clear` no-op with an environment override note, without exposing the
  token value
- `config clear --force` default deletion
- `config clear --force` default deletion with an environment override note,
  without exposing the token value
- interactive `config clear` confirm/cancel prompt behavior
- `config clear --non-interactive` without `--force`, proving no preview or
  warning leaks before `prompt.ErrConfirmationRequired`
- `config clear --non-interactive --force`
- `config clear --all`
- `config clear --all` when keyring planning fails but plaintext cleanup can
  still proceed
- an executable source gate that permits only the prompt `fmt.Fprint` exception
  and rejects migrated diagnostic/status strings in `configcmd/test.go` and
  `configcmd/clear.go`

## Live CLI Transcript

The binary was rebuilt:

```bash
rtk go build -o ./bin/cfl ./tools/cfl/cmd/cfl
```

Transcript directory:

```text
/tmp/cfl-436-proof.u3byyu
```

This path was an intentionally ephemeral local capture directory. The redacted
durable excerpts needed for review are included below.

Executed:

```bash
./bin/cfl --no-color config test
```

Result:

```text
exit status: 0
stdout bytes: 0
stderr bytes: 169
```

Observed stderr, with identity fields redacted:

```text
Testing connection... success!

Authentication successful
API access verified

Authenticated as: <redacted name> (<redacted email>)
Account ID: <redacted account id>
```

Executed:

```bash
./bin/cfl --no-color --non-interactive config clear
```

Result:

```text
exit status: 1
stdout bytes: 0
stderr bytes: 80
```

Observed stderr:

```text
Error: --non-interactive: confirmation required; re-run with --force to proceed
```

This proves the safe non-destructive case fails loudly before prompt preview,
shared-token warning, keyring deletion, or any token value exposure.
