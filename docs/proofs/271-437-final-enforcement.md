# Proof: #437 Final `cfl` Presenter Boundary Enforcement

## Scope

This proof covers the final enforcement pass for parent `#271`.

The final state is:

- default `cfl` text output is presenter/renderer based
- commands orchestrate and call presenters
- presenters own wording, labels, ordering, stream routing, and output shape
- `present.Emit` is the intended write point for presenter-rendered output
- remaining direct-output exceptions are explicit and enforced

## Child Proof Index

- `docs/proofs/271-431-cfl-me.md`
- `docs/proofs/271-432-cfl-list-search.md`
- `docs/proofs/271-433-cfl-detail-config.md`
- `docs/proofs/271-434-cfl-page-view.md`
- `docs/proofs/271-435-cfl-mutation-success.md`
- `docs/proofs/271-436-cfl-diagnostics-advisories.md`

Temporary transcript directories named in child proofs are intentionally
ephemeral. The proof files contain durable redacted excerpts and any created or
deleted IDs needed to audit behavior after `/tmp` files are gone.

## Enforcement

Executable enforcement was added in:

```text
tools/cfl/internal/cmd/root/presenter_boundary_test.go
```

It scans all non-test production Go files under `tools/cfl/internal/cmd` using
the Go AST and fails on:

- legacy `v.Table`, `v.Success`, `v.RenderKeyValue`, `v.RenderKeyValues`,
  `v.Info`, `v.Warning`, `v.Error`, `v.Println`, or `v.Render` outside the
  `init` exception
- command-local `fmt.Fprint*` writes to `opts.Stdout`, `opts.Stderr`, `v.Out`,
  `os.Stdout`, or `os.Stderr` outside prompt/init exceptions
- bare `fmt.Print`, `fmt.Printf`, or `fmt.Println` outside `init`
- import-alias variants of `fmt`, `io`, `log`, and `os`
- `io.WriteString` or direct `.Write` calls to command output streams
- `log.Print*`, `log.Fatal*`, or `log.Panic*` output outside `init`
- `view.ValidateFormat`
- `opts.View()` outside `init`
- direct `shared/view` imports outside root/init exceptions

Allowed exceptions:

- `tools/cfl/internal/cmd/init/**`: interactive wizard and migration UX
- root `Options.View()` bridge while `init` remains on `shared/view`
- one-shot delete/config confirmation prompt text on `opts.Stderr`

## Verification Commands

Executed:

```bash
rtk go test ./tools/cfl/internal/cmd/root ./tools/cfl/internal/present ./shared/present
```

Result:

```text
Go test: 102 passed in 3 packages
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
rtk golangci-lint run ./tools/cfl/... ./shared/...
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

## Grep Evidence

Executed:

```bash
rtk rg -n '\bv\.(Table|Success|RenderKeyValue|RenderKeyValues|Info|Warning|Error|Println|Render)\b' tools/cfl/internal/cmd --glob '!**/*_test.go'
```

Result:

```text
tools/cfl/internal/cmd/init/init.go:117:		v.Error("Cannot resolve the shared credential store path: %v", err)
tools/cfl/internal/cmd/init/init.go:118:		v.Error("Set XDG_CONFIG_HOME to an absolute path (or unset it), then re-run cfl init.")
tools/cfl/internal/cmd/init/init.go:141:		v.Error("Could not prepare secure credential storage: %v", err)
tools/cfl/internal/cmd/init/init.go:329:			v.Error("Could not construct API client: %v", err)
tools/cfl/internal/cmd/init/init.go:337:			v.Error("Connection failed: %v", err)
tools/cfl/internal/cmd/init/init.go:338:			v.Error("Check your credentials and try again")
tools/cfl/internal/cmd/init/init.go:342:		v.Success("Connected to %s", cfg.URL)
tools/cfl/internal/cmd/init/init.go:351:			v.Info("Saving credentials affects jtk (shared default section); proceeding under --non-interactive.")
tools/cfl/internal/cmd/init/init.go:364:				v.Info("Initialization cancelled. No changes saved.")
tools/cfl/internal/cmd/init/init.go:380:		v.Error("Saved the non-secret config to %s, but could not store the API token in the keyring: %v", sharedPath, err)
tools/cfl/internal/cmd/init/init.go:381:		v.Error("Recover by storing just the token (no need to re-run init): `cfl set-credential --ref atlassian-cli/default --key api_token --stdin --overwrite` (reads stdin; use --from-env VAR for env-driven setup).")
tools/cfl/internal/cmd/init/init.go:384:	v.Success("Configuration saved to %s (token stored in the OS keyring)", sharedPath)
tools/cfl/internal/cmd/init/init.go:392:			v.Info("Skipping cleanup of %s under --non-interactive; remove manually if desired.", lp)
tools/cfl/internal/cmd/init/init.go:407:				v.Error("Could not remove %s: %v", lp, err)
tools/cfl/internal/cmd/init/init.go:409:				v.Info("Removed %s", lp)
tools/cfl/internal/cmd/init/init.go:417:		v.Println("")
tools/cfl/internal/cmd/init/init.go:423:	v.Println("")
tools/cfl/internal/cmd/init/init.go:424:	v.Println("You're all set! Try running:")
tools/cfl/internal/cmd/init/init.go:425:	v.Println("  cfl space list")
tools/cfl/internal/cmd/init/init.go:426:	v.Println("  cfl page list --space <SPACE_KEY>")
tools/cfl/internal/cmd/init/init.go:429:		v.Println("")
tools/cfl/internal/cmd/init/init.go:430:		v.Info("To switch back to basic auth later, run: cfl init --auth-method basic")
tools/cfl/internal/cmd/init/reconcile.go:57:			v.Error("Shared credential store at %s is unreadable: %v", sharedPath, relErr)
tools/cfl/internal/cmd/init/reconcile.go:58:			v.Error("Refusing to overwrite. Fix or remove the file, then re-run cfl init.")
tools/cfl/internal/cmd/init/reconcile.go:60:			v.Error("Shared credential store relocation check failed: %v", relErr)
tools/cfl/internal/cmd/init/reconcile.go:61:			v.Error("Refusing to mutate anything. Reconcile the named file(s), then re-run cfl init.")
tools/cfl/internal/cmd/init/reconcile.go:68:		v.Error("Shared credential store at %s is unreadable: %v", sharedPath, err)
tools/cfl/internal/cmd/init/reconcile.go:69:		v.Error("Refusing to overwrite. Fix or remove the file, then re-run cfl init.")
tools/cfl/internal/cmd/init/reconcile.go:77:		v.Error("Shared credential store at %s is unreadable: %v", sharedPath, err)
tools/cfl/internal/cmd/init/reconcile.go:78:		v.Error("Refusing to overwrite. Fix or remove the file, then re-run cfl init.")
tools/cfl/internal/cmd/init/reconcile.go:88:			v.Error("Legacy cfl config at %s is unreadable: %v", cflLegacyPath, cflErr)
tools/cfl/internal/cmd/init/reconcile.go:89:			v.Error("Refusing to overwrite. Fix or remove the file, then re-run cfl init.")
tools/cfl/internal/cmd/init/reconcile.go:96:		v.Info("Note: sibling jtk config at %s is unreadable; ignoring. (%v)", jtkLegacyPath, jtkErr)
tools/cfl/internal/cmd/init/reconcile.go:118:			v.Error("Could not relocate the shared credential store: %v", aerr)
tools/cfl/internal/cmd/init/reconcile.go:127:			v.Error("Shared credential store at %s is unreadable: %v", sharedPath, err)
```

All matches are in the documented `init` exception.

Executed:

```bash
rtk rg -n 'fmt\.F(print|printf|println)\((opts\.(Stdout|Stderr)|v\.Out|os\.Stderr),\s*"' tools/cfl/internal/cmd --glob '!**/*_test.go'
```

Result:

```text
tools/cfl/internal/cmd/space/delete.go:63:		_, _ = fmt.Fprintf(opts.Stderr, "About to delete space: %s (%s)\n", space.Name, space.Key)
tools/cfl/internal/cmd/space/delete.go:64:		_, _ = fmt.Fprint(opts.Stderr, "Are you sure? [y/N]: ")
tools/cfl/internal/cmd/attachment/delete.go:62:		_, _ = fmt.Fprintf(opts.Stderr, "About to delete attachment: %s (ID: %s)\n", attachment.Title, attachment.ID)
tools/cfl/internal/cmd/attachment/delete.go:63:		_, _ = fmt.Fprint(opts.Stderr, "Are you sure? [y/N]: ")
tools/cfl/internal/cmd/page/delete.go:63:		_, _ = fmt.Fprintf(opts.Stderr, "About to delete page: %s (ID: %s)\n", page.Title, page.ID)
tools/cfl/internal/cmd/page/delete.go:64:		_, _ = fmt.Fprint(opts.Stderr, "Are you sure? [y/N]: ")
```

All matches are documented confirmation prompt exceptions. The stricter
executable enforcement test also covers computed prompt messages such as
`config clear`.

Executed:

```bash
rtk rg -n 'view\.ValidateFormat|opts\.View\(|github.com/open-cli-collective/atlassian-go/view' tools/cfl/internal/cmd --glob '!**/*_test.go'
```

Result:

```text
tools/cfl/internal/cmd/root/root.go:18:	"github.com/open-cli-collective/atlassian-go/view"
tools/cfl/internal/cmd/init/init.go:105:	v := opts.View()
tools/cfl/internal/cmd/init/init.go:322:	v := opts.View()
tools/cfl/internal/cmd/init/reconcile.go:7:	"github.com/open-cli-collective/atlassian-go/view"
```

All matches are root/init transitional exceptions.

## Live Smoke

The binary was rebuilt:

```bash
rtk go build -o ./bin/cfl ./tools/cfl/cmd/cfl
```

Live smoke used `ATLASSIAN_API_TOKEN` from the local keychain. The token value
was not written to any proof file.

Transcript directory:

```text
/tmp/cfl-437-smoke.cZWWs7
```

This path is intentionally ephemeral; durable excerpts are below.

Discovered smoke inputs:

```text
space key: ~595553618
page id: 33110
scratch page id: 3555033104
```

Representative read commands:

```bash
bin/cfl --no-color me
bin/cfl --no-color space list --limit 2
bin/cfl --no-color page list --space '~595553618' --limit 2
bin/cfl --no-color search --type page --limit 1
bin/cfl --no-color page view 33110
bin/cfl --no-color attachment list --page 33110 --limit 2
```

Observed excerpts, with identity fields redacted where needed:

```text
me stdout:
<redacted account id> | <redacted name> | <redacted email>

space list stdout:
ID     KEY         TYPE      NAME
33015  ~595553618  personal  Konstantin N
65554  ~531183998  personal  Brandon Rumburg

space list stderr:
Next page: cfl space list --cursor "<redacted cursor>"

page list stdout:
ID     TITLE         STATUS
33110  Konstantin    current
33111  Sample Pages  current

page list stderr:
(showing first 2 results, use --limit to see more)

search stdout:
ID      TYPE  SPACE  TITLE
753683  page  ENG    Onboarding

search stderr:
(showing 1 of 2386 results, use --limit to see more)

page view stdout:
Title: Konstantin
ID: 33110
Space: ~595553618 (ID: 33015)
Version: 3

attachment list stderr:
No attachments found.
```

Representative mutation command:

```bash
bin/cfl --no-color page create --space '~595553618' --title "CFL Final Enforcement Scratch 20260624181253" --file /tmp/cfl-437-smoke.cZWWs7/scratch.md
bin/cfl --no-color page delete 3555033104 --force
```

Observed mutation output:

```text
Created page: CFL Final Enforcement Scratch 20260624181253
ID: 3555033104
URL: https://monitproduct.atlassian.net/wiki/spaces/~595553618/pages/3555033104/CFL+Final+Enforcement+Scratch+20260624181253

Deleted page: CFL Final Enforcement Scratch 20260624181253 (ID: 3555033104)
```

Cleanup result:

- scratch page `3555033104` was deleted with `--force`
- no scratch Confluence artifacts remain from this proof run
