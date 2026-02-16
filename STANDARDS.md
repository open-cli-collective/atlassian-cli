# Go CLI Style Guide

This document catalogs coding conventions for Go CLI tools. It is intended for use as an operationalized code review prompt for AI-assisted review, but is also useful as a human reference.

When reviewing code, flag deviations from these patterns. Be pragmatic: the goal is consistency within a codebase, not pedantic enforcement. If a deviation improves readability or correctness, note it as an intentional departure rather than a defect.

### Guiding Philosophy

Prefer clarity, composability, and maintainability over cleverness. Go's strength is boringly readable code — lean into that. Use the standard library aggressively. Resist the urge to abstract prematurely or import a dependency for something you can write in 20 lines. Use judgement, not dogma.

---

## 1. Project Configuration

### Module Layout

Every tool gets its own `go.mod`. For a monorepo with shared libraries, use a top-level module with internal packages:

```
tools/
├── go.mod
├── go.sum
├── cmd/
│   ├── ingest/
│   │   └── main.go
│   ├── reconcile/
│   │   └── main.go
│   └── sync/
│       └── main.go
├── internal/
│   ├── config/
│   ├── logging/
│   └── aws/
└── pkg/          # only if genuinely intended for external consumption
```

`internal/` is the default for shared code. `pkg/` is only for packages explicitly designed as public API for other modules. When in doubt, use `internal/`.

### Build Configuration

Pin the Go version in `go.mod` and use a `.tool-versions` or `go.env` for the team:

```
go 1.24
```

Use `go.sum` for reproducible builds. Run `go mod tidy` before every commit — CI should fail if `go.mod` and `go.sum` are dirty.

### Linting

All projects use `golangci-lint` with a shared `.golangci.yml`. At minimum, enable:

```yaml
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - ineffassign
    - misspell
    - revive
    - gosec
    - errorlint    # enforce error wrapping best practices
    - exhaustive   # enforce exhaustive switch/select on enums
```

`go vet` and `staticcheck` findings are non-negotiable. Treat them as errors in CI.

### Dependency Hygiene

Order imports in three groups separated by blank lines: standard library, external dependencies, internal packages:

```go
import (
    "context"
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "go.uber.org/zap"

    "github.com/yourorg/tools/internal/config"
)
```

`goimports` enforces this automatically. Run it on save.

### Makefile

Every repo has a `Makefile` at the root. This is the answer to "I just cloned this repo, now what." CI runs the same targets developers run locally.

```makefile
.PHONY: build lint test tidy check

# Build all binaries into bin/
build:
	go build -o bin/ ./cmd/...

# Lint with golangci-lint (config in .golangci.yml)
lint:
	golangci-lint run ./...

# Run tests with race detector
test:
	go test -race ./...

# Tidy and verify modules are clean
tidy:
	go mod tidy
	git diff --exit-code go.mod go.sum

# CI gate: everything that must pass before merge
check: tidy lint test build
```

**Rules:**

- `make check` is the CI gate. It must pass before merge. Run it locally before pushing.
- `make build` outputs all binaries to `bin/`. Add `bin/` to `.gitignore`.
- `make tidy` fails if `go.mod` or `go.sum` are dirty — this catches forgotten `go mod tidy` runs.
- Add tool-specific targets as needed (`make migrate`, `make generate`, `make integration-test`), but the four core targets (`build`, `lint`, `test`, `tidy`) are non-negotiable.
- Keep targets simple. If a target exceeds ~5 lines of shell, it belongs in a script in `scripts/` that the Makefile calls.

---

## 2. Type Design

### Structs for Data, Methods for Behavior

Go doesn't have records, but the same instinct applies: separate data-carrying types from service types. Data structs should be plain, exported fields. Service types hold dependencies and attach methods:

```go
// Data: plain struct, no methods beyond serialization
type SyncResult struct {
    CompanyID  string
    Success    bool
    FailedKeys []string
    Duration   time.Duration
}

// Service: holds dependencies, has methods
type Reconciler struct {
    db     *sql.DB
    logger *slog.Logger
    clock  func() time.Time // injectable for testing
}
```

### Prefer Value Semantics for Small Types

Small data types (< ~128 bytes, no mutability needs) should be passed and returned by value, not pointer. This is Go's equivalent of preferring value types:

```go
// Good: small, immutable-ish, pass by value
type Tenant struct {
    ID   string
    Name string
}

type DateRange struct {
    Start time.Time
    End   time.Time
}

// Pointer receiver appropriate: large struct or needs mutation
type IngestionState struct {
    // ... many fields, mutated over lifetime
}

func (s *IngestionState) MarkComplete(key string) { ... }
```

### Strongly-Typed Identifiers

Wrap primitive identifiers in named types to prevent parameter confusion:

```go
type TenantID string
type CompanyID string
type BusinessID string

func GetConnection(tenant TenantID, company CompanyID) (*Connection, error) { ... }
```

This makes `GetConnection(companyID, tenantID)` a compile error instead of a subtle bug. Use sparingly — only where misorderings are a real risk (multiple string parameters of the same shape).

### Constructor Functions

Use `NewX` functions when a type requires initialization, validation, or has unexported fields. Return the concrete type, not an interface:

```go
func NewReconciler(db *sql.DB, logger *slog.Logger, opts ...Option) *Reconciler {
    r := &Reconciler{
        db:     db,
        logger: logger,
        clock:  time.Now,
    }
    for _, opt := range opts {
        opt(r)
    }
    return r
}
```

For simple structs with all-exported fields, struct literals are fine — no constructor needed.

### Enums via Constants

Go lacks sum types. Use typed constants with `iota`, and always handle the zero value explicitly:

```go
type PlatformType int

const (
    PlatformUnknown    PlatformType = iota // zero value = unknown
    PlatformAccounting
    PlatformBanking
    PlatformCommerce
)

func (p PlatformType) String() string {
    switch p {
    case PlatformAccounting:
        return "Accounting"
    case PlatformBanking:
        return "Banking"
    case PlatformCommerce:
        return "Commerce"
    default:
        return fmt.Sprintf("PlatformType(%d)", p)
    }
}
```

For cases where you need exhaustiveness checking, `exhaustive` lint catches missing switch arms.

---

## 3. Interface Design

### Accept Interfaces, Return Structs

This is the single most important Go design principle. Define interfaces at the call site (consumer), not at the implementation:

```go
// Good: interface defined where it's consumed, not where it's implemented
// In reconciler.go:
type AccountFetcher interface {
    GetAccounts(ctx context.Context, tenant TenantID) ([]Account, error)
}

type Reconciler struct {
    accounts AccountFetcher
    // ...
}

// In accounts.go — no interface declared here, just a concrete type
type AccountStore struct {
    db *sql.DB
}

func (s *AccountStore) GetAccounts(ctx context.Context, tenant TenantID) ([]Account, error) { ... }
```

### Keep Interfaces Small

One to three methods is ideal. If an interface has more than five methods, it's probably doing too much. The standard library's `io.Reader` (one method) is the gold standard.

```go
// Good: focused interface
type TokenRefresher interface {
    RefreshToken(ctx context.Context, tenant TenantID) (Token, error)
}

// Suspicious: interface is a service dump
type BusinessManager interface {
    GetBusiness(ctx context.Context, id string) (*Business, error)
    CreateBusiness(ctx context.Context, b Business) error
    UpdateBusiness(ctx context.Context, b Business) error
    DeleteBusiness(ctx context.Context, id string) error
    ListBusinesses(ctx context.Context, tenant TenantID) ([]Business, error)
    GetBusinessConnections(ctx context.Context, id string) ([]Connection, error)
    // ... this is just a struct with extra steps
}
```

### The Empty Interface Smell

`any` (`interface{}`) in function signatures is almost always a design smell. It means "I gave up on types." Acceptable uses: logging arguments, JSON marshaling boundaries, generic containers. Unacceptable: core business logic parameters.

---

## 4. Error Handling

### Errors Are Values, Not Exceptions

Every function that can fail returns an `error`. Check it immediately. Never discard errors silently:

```go
// Good: check immediately, handle or propagate
result, err := store.GetAccounts(ctx, tenant)
if err != nil {
    return nil, fmt.Errorf("fetching accounts for %s: %w", tenant, err)
}
```

### Wrapping With Context

Always wrap errors with `fmt.Errorf("context: %w", err)` to build a trace. The message should describe what *this* function was trying to do, not repeat what the callee said:

```go
// Good: each layer adds its own context
func (r *Reconciler) Run(ctx context.Context, tenant TenantID) error {
    accounts, err := r.accounts.GetAccounts(ctx, tenant)
    if err != nil {
        return fmt.Errorf("reconciling tenant %s: %w", tenant, err)
    }
    // ...
}

// Bad: restating the callee's error
if err != nil {
    return fmt.Errorf("failed to get accounts: %w", err) // "failed to" is noise
}
```

### Sentinel Errors and Error Types

Define sentinel errors for conditions callers need to match on. Use custom error types when callers need structured data:

```go
var (
    ErrNotFound      = errors.New("not found")
    ErrAlreadyExists = errors.New("already exists")
    ErrRateLimited   = errors.New("rate limited")
)

// Callers check with errors.Is:
if errors.Is(err, ErrNotFound) {
    // handle missing resource
}

// Custom error type when callers need details:
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation: %s: %s", e.Field, e.Message)
}

// Callers check with errors.As:
var ve *ValidationError
if errors.As(err, &ve) {
    fmt.Printf("bad field: %s\n", ve.Field)
}
```

### Don't Panic

`panic` is for programmer bugs (impossible states, violated invariants in init), never for runtime errors. A CLI tool that panics on bad input is broken. Recover from panics only at the outermost boundary (e.g., a top-level middleware in a server, or the root command's `RunE`).

### Eliminate `else` After Error Returns

Go's error handling naturally produces guard clauses. Embrace them — never nest the happy path inside `else`:

```go
// Good: guard clause, happy path is un-indented
token, err := auth.GetToken(ctx, tenant)
if err != nil {
    return fmt.Errorf("getting token: %w", err)
}
// continue with token...

// Bad: unnecessary nesting
token, err := auth.GetToken(ctx, tenant)
if err == nil {
    // happy path buried in a branch
} else {
    return err
}
```

---

## 5. Context Propagation

### Context Is Always the First Parameter

Every function that does I/O, calls other services, or could be cancelled takes `context.Context` as its first parameter. Named `ctx`:

```go
func (s *SyncService) Sync(ctx context.Context, tenant TenantID, companyID CompanyID) error
```

### Never Store Context in Structs

Context is request-scoped. Storing it in a struct means you're holding onto a cancelled context or sharing one across requests:

```go
// Bad: context outlives the request
type Worker struct {
    ctx context.Context // don't do this
}

// Good: pass per-call
func (w *Worker) Process(ctx context.Context, job Job) error { ... }
```

### Respect Cancellation

Check `ctx.Err()` or use `select` on `ctx.Done()` in loops and before expensive operations:

```go
for _, batch := range batches {
    if err := ctx.Err(); err != nil {
        return fmt.Errorf("cancelled during batch processing: %w", err)
    }
    if err := processBatch(ctx, batch); err != nil {
        return err
    }
}
```

---

## 6. CLI Patterns

### Cobra for All Tools

Use Cobra for every CLI tool, even single-command ones. The cognitive cost of "which framework did this tool use" is worse than the tiny overhead of Cobra on a simple tool. Cobra gives you consistent `--help`, flag parsing, subcommand structure, and shell completion for free. Standardize on it and stop thinking about it.

```go
func main() {
    root := &cobra.Command{
        Use:   "mytool",
        Short: "Does the thing",
        // No Run on root — forces subcommand usage
    }

    root.AddCommand(
        newSyncCmd(),
        newReconcileCmd(),
        newReportCmd(),
    )

    if err := root.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### Command Factory Functions

Each subcommand lives in its own file and returns a `*cobra.Command`. Wire dependencies in the `RunE` closure:

```go
func newSyncCmd() *cobra.Command {
    var (
        tenant  string
        dryRun  bool
        workers int
    )

    cmd := &cobra.Command{
        Use:   "sync [company-id]",
        Short: "Sync data for a company",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()
            companyID := args[0]

            cfg, err := config.Load()
            if err != nil {
                return fmt.Errorf("loading config: %w", err)
            }

            logger := logging.New(cfg.LogLevel)
            db, err := openDB(ctx, cfg.DatabaseURL)
            if err != nil {
                return fmt.Errorf("connecting to database: %w", err)
            }
            defer db.Close()

            svc := NewSyncService(db, logger)
            return svc.Run(ctx, TenantID(tenant), CompanyID(companyID), dryRun)
        },
    }

    cmd.Flags().StringVar(&tenant, "tenant", "", "tenant identifier (required)")
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without writing")
    cmd.Flags().IntVar(&workers, "workers", 4, "number of parallel workers")
    _ = cmd.MarkFlagRequired("tenant")

    return cmd
}
```

### Exit Codes

Use distinct exit codes for different failure modes. Define them as constants:

```go
const (
    ExitOK              = 0
    ExitUsageError      = 1
    ExitRuntimeError    = 2
    ExitConfigError     = 3
    ExitPartialFailure  = 4
)
```

Cobra handles exit code 1 for usage errors by default. For other cases, handle in `main`:

```go
func main() {
    if err := root.Execute(); err != nil {
        var cfgErr *config.Error
        if errors.As(err, &cfgErr) {
            os.Exit(ExitConfigError)
        }
        os.Exit(ExitRuntimeError)
    }
}
```

### Stdin/Stdout/Stderr Discipline

Standard output is for *data* (pipeable results). Standard error is for *diagnostics* (logs, progress, errors). Never mix them:

```go
// Data goes to stdout — can be piped to jq, another tool, etc.
enc := json.NewEncoder(os.Stdout)
enc.Encode(result)

// Diagnostics go to stderr
logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
```

If the tool's primary output is human-readable (not piped), stdout is fine for both, but design for the pipeable case first.

### Signal Handling

CLI tools should handle SIGINT/SIGTERM gracefully. Use `signal.NotifyContext` for cancellation:

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    if err := run(ctx); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}
```

---

## 7. Configuration

### Layered Config: Env > Flags > File > Defaults

Configuration sources, in precedence order: environment variables override flags, flags override file values, file values override defaults. Use a single config struct:

```go
type Config struct {
    DatabaseURL string        `env:"DATABASE_URL" json:"database_url"`
    LogLevel    string        `env:"LOG_LEVEL"    json:"log_level"`
    Workers     int           `env:"WORKERS"      json:"workers"`
    Timeout     time.Duration `env:"TIMEOUT"      json:"timeout"`
    DryRun      bool          // flag-only, no file/env
}

func Load() (*Config, error) {
    cfg := Config{
        LogLevel: "info",
        Workers:  4,
        Timeout:  30 * time.Second,
    }

    // Load from file if present, then overlay env vars
    // ...

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    return &cfg, nil
}
```

### Validate Early, Fail Fast

Validate all configuration at startup before doing any work. A config error 30 minutes into a batch job is a waste:

```go
func (c *Config) Validate() error {
    if c.DatabaseURL == "" {
        return errors.New("DATABASE_URL is required")
    }
    if c.Workers < 1 || c.Workers > 64 {
        return fmt.Errorf("workers must be 1-64, got %d", c.Workers)
    }
    return nil
}
```

### Testable Time

Inject a clock function instead of calling `time.Now()` directly. This is the same principle as C#'s `TimeProvider`:

```go
// In production
svc := &Service{clock: time.Now}

// In tests
svc := &Service{clock: func() time.Time { return fixedTime }}
```

For more complex time needs, define a small interface:

```go
type Clock interface {
    Now() time.Time
}
```

---

## 8. Concurrency

### Start Goroutines, Manage Lifetimes

Every goroutine must have a clear shutdown path. Use `context.Context` for cancellation and `sync.WaitGroup` or `errgroup.Group` for completion:

```go
g, ctx := errgroup.WithContext(ctx)

for _, job := range jobs {
    g.Go(func() error {
        return processJob(ctx, job)
    })
}

if err := g.Wait(); err != nil {
    return fmt.Errorf("processing jobs: %w", err)
}
```

### errgroup for Parallel Tasks

`errgroup` is the default for parallel work in CLI tools. It handles cancellation on first error and waitgroup semantics in one package:

```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(workers) // bounded parallelism

for _, item := range items {
    g.Go(func() error {
        return process(ctx, item)
    })
}
return g.Wait()
```

### Channels for Pipelines, Not for Synchronization

Use channels when data flows between stages. Use `sync.WaitGroup`, `errgroup`, or `sync.Mutex` for synchronization. Don't use a `chan struct{}` when a `WaitGroup` is clearer:

```go
// Good: channel as a pipeline stage
func produce(ctx context.Context, items []Item) <-chan Item {
    ch := make(chan Item)
    go func() {
        defer close(ch)
        for _, item := range items {
            select {
            case ch <- item:
            case <-ctx.Done():
                return
            }
        }
    }()
    return ch
}
```

### Never Launch Unbounded Goroutines

Always limit concurrency for I/O-bound work. A CLI tool that launches 10,000 goroutines to hit an API will get rate-limited or OOM. Use `errgroup.SetLimit`, a semaphore channel, or a worker pool:

```go
// Semaphore pattern
sem := make(chan struct{}, maxConcurrency)
for _, item := range items {
    sem <- struct{}{}
    go func() {
        defer func() { <-sem }()
        process(ctx, item)
    }()
}
```

---

## 9. Data Access

### database/sql with pgx or lib/pq

Use `database/sql` as the interface layer. `pgx` is preferred as the driver for PostgreSQL (better performance, native types). Dapper-style explicit SQL applies equally here — write your own queries, don't hide behind an ORM:

```go
const accountsQuery = `
    WITH target AS (
        SELECT b.id AS business_id
        FROM business b
        JOIN financial_institution fi ON b.tenant_id = fi.tenant_id
        WHERE fi.fi_identifier = $1 AND b.company_id = $2
    )
    SELECT a.id, a.name, a.type
    FROM account a
    JOIN target t ON a.business_id = t.business_id
    ORDER BY a.name
`

func (s *AccountStore) GetAccounts(ctx context.Context, tenant TenantID, company CompanyID) ([]Account, error) {
    rows, err := s.db.QueryContext(ctx, accountsQuery, string(tenant), string(company))
    if err != nil {
        return nil, fmt.Errorf("querying accounts: %w", err)
    }
    defer rows.Close()

    var accounts []Account
    for rows.Next() {
        var a Account
        if err := rows.Scan(&a.ID, &a.Name, &a.Type); err != nil {
            return nil, fmt.Errorf("scanning account row: %w", err)
        }
        accounts = append(accounts, a)
    }
    return accounts, rows.Err()
}
```

### SQL Best Practices

These carry over directly from the C# guide:

- Always specify columns — avoid `SELECT *`
- Always use parameterized queries (`$1`, `$2`), never `fmt.Sprintf` into SQL
- Use CTEs over subqueries for readability
- Paginate large result sets; prefer cursor-based pagination over `OFFSET`/`LIMIT`
- Batch large `IN` clauses (100+ items) with `ANY($1::text[])` or temp tables

### Transaction Management

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return fmt.Errorf("beginning transaction: %w", err)
}
defer tx.Rollback() // no-op if committed

// ... operations using tx ...

if err := tx.Commit(); err != nil {
    return fmt.Errorf("committing transaction: %w", err)
}
```

The `defer tx.Rollback()` pattern is idiomatic — it's a no-op after a successful commit and ensures cleanup on any error path.

---

## 10. Serialization

### encoding/json from the Standard Library

Use `encoding/json` by default. For performance-sensitive paths, `json/v2` (when stable) or `github.com/goccy/go-json` are acceptable drop-in replacements.

### Struct Tags Are the Schema

```go
type SyncRequest struct {
    TenantID  string       `json:"tenant_id"`
    CompanyID string       `json:"company_id"`
    Platform  PlatformType `json:"platform"`
    Priority  int          `json:"priority,omitempty"`
}
```

Use `omitempty` deliberately — it means "omit when zero value," which may or may not be what you want. An `int` field with `omitempty` drops `0`, which may be meaningful.

### Custom Marshaling for Enums

```go
func (p PlatformType) MarshalJSON() ([]byte, error) {
    return json.Marshal(p.String())
}

func (p *PlatformType) UnmarshalJSON(data []byte) error {
    var s string
    if err := json.Unmarshal(data, &s); err != nil {
        return err
    }
    switch s {
    case "Accounting":
        *p = PlatformAccounting
    case "Banking":
        *p = PlatformBanking
    default:
        return fmt.Errorf("unknown platform type: %q", s)
    }
    return nil
}
```

---

## 11. Logging

### slog for CLIs, zap for Services

Use `log/slog` from the standard library for CLI tools. It's zero-dependency, has the right level of abstraction for console programs, and writes to stderr by default (which is what you want for CLIs — see Section 6 on stdout/stderr discipline).

For long-running web services where logging is on the hot path, `go.uber.org/zap` is acceptable — it's measurably faster due to pre-allocation and zero-reflection design. But for CLI tools, logging throughput is never the bottleneck, and slog's simplicity wins.

```go
// CLI: slog with text handler for human-readable output
handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})
logger := slog.New(handler)

// CLI: JSON handler when output will be ingested by log aggregation
handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})
logger := slog.New(handler)

// Good: structured key-value pairs
logger.Info("calculating insights",
    "tenant", tenant,
    "company_id", companyID,
    "platform", platform,
)

logger.Error("sync failed",
    "tenant", tenant,
    "company_id", companyID,
    "error", err,
    "elapsed_ms", elapsed.Milliseconds(),
)
```

### Named Fields, Not Interpolation

Same principle as the C# guide — each value must be a discrete, queryable field:

```go
// Good: each field independently queryable
logger.Info("processing transaction",
    "tenant", tenant,
    "company_id", companyID,
    "txn_id", txnID,
)

// Bad: opaque string defeats structured logging
logger.Info(fmt.Sprintf("%s::%s - processing transaction %s", tenant, companyID, txnID))
```

### Logger Propagation

Pass loggers as dependencies, not globals. Use `slog.With` to add context that applies to all messages in a scope:

```go
func (s *SyncService) Run(ctx context.Context, tenant TenantID, company CompanyID) error {
    log := s.logger.With("tenant", tenant, "company_id", company)
    log.Info("starting sync")
    // all subsequent log calls in this scope include tenant and company_id
}
```

### Log Levels

| Level | Usage |
|-------|-------|
| Info  | Start/completion of major operations, business events |
| Warn  | Retry attempts, degraded scenarios, non-critical issues |
| Error | Failures (always include the error value) |
| Debug | Detailed operational info, only enabled in dev/troubleshooting |

### Performance Timing

```go
start := time.Now()
// ... work
logger.Info("operation complete",
    "tenant", tenant,
    "elapsed_ms", time.Since(start).Milliseconds(),
)
```

### Log Security

Never log sensitive information: passwords, tokens, PII, full credit card numbers, SSNs. Be cautious with user attributes — only log what's necessary for debugging.

---

## 12. Error Handling & Result Patterns

### Guard Clauses and Early Returns

Same as C#: reject invalid states early, keep the happy path at the lowest indentation level:

```go
func (s *Service) Process(ctx context.Context, req Request) (*Result, error) {
    if req.TenantID == "" {
        return nil, &ValidationError{Field: "tenant_id", Message: "required"}
    }

    conn, err := s.getConnection(ctx, req.TenantID)
    if err != nil {
        return nil, fmt.Errorf("getting connection: %w", err)
    }

    // happy path continues un-indented...
}
```

### Multi-Value Returns for Outcome Disambiguation

Go's multiple return values serve the same role as C#'s tuple returns:

```go
// Found vs not-found vs error are three different outcomes
func (s *Store) GetAccount(ctx context.Context, id string) (account Account, found bool, err error) {
    row := s.db.QueryRowContext(ctx, query, id)
    if err := row.Scan(&account.ID, &account.Name); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return Account{}, false, nil
        }
        return Account{}, false, fmt.Errorf("scanning account: %w", err)
    }
    return account, true, nil
}
```

### ok-Pattern for Optional Results

For lookups that may miss, use the comma-ok pattern familiar from map access:

```go
val, ok := cache[key]
if !ok {
    // handle miss
}
```

### Collecting Errors in Batch Operations

For operations that process multiple items where you want partial results, collect errors rather than failing on the first one:

```go
var errs []error
for _, item := range items {
    if err := process(ctx, item); err != nil {
        errs = append(errs, fmt.Errorf("item %s: %w", item.ID, err))
        continue
    }
}
if len(errs) > 0 {
    return fmt.Errorf("partial failure (%d/%d): %w", len(errs), len(items), errors.Join(errs...))
}
```

---

## 13. Collection Patterns

### Nil Slices Over Empty Slices

In Go, a nil slice and an empty slice behave identically for `len`, `cap`, `range`, and `append`. Prefer nil (the zero value) — don't allocate when there's nothing to hold:

```go
// Good: zero value is fine
var accounts []Account
// len(accounts) == 0, range works, append works

// Unnecessary: allocating for no reason
accounts := make([]Account, 0)
accounts := []Account{}
```

Exception: JSON serialization. `json.Marshal(nil)` produces `null`, while `json.Marshal([]Account{})` produces `[]`. If the distinction matters to consumers, initialize explicitly.

### Pre-Allocate When Size Is Known

```go
results := make([]Result, 0, len(items))
for _, item := range items {
    results = append(results, transform(item))
}
```

### maps Package for Common Operations

Use `maps.Keys`, `maps.Values`, `maps.Clone` from the standard library instead of hand-rolling:

```go
import "maps"

keys := slices.Sorted(maps.Keys(accountsByID))
```

### slices Package for Transformations

Use `slices.SortFunc`, `slices.Contains`, `slices.Compact`, etc.:

```go
import "slices"

slices.SortFunc(accounts, func(a, b Account) int {
    return strings.Compare(a.Name, b.Name)
})

hasAdmin := slices.ContainsFunc(roles, func(r Role) bool {
    return r.Name == "admin"
})
```

### Chunking for Batch Operations

Same concept as C#'s `.Chunk()` — batch items for APIs with size limits:

```go
func Chunk[T any](items []T, size int) [][]T {
    var chunks [][]T
    for size < len(items) {
        items, chunks = items[size:], append(chunks, items[:size])
    }
    return append(chunks, items)
}

// Usage: DynamoDB BatchWriteItem supports max 25 items
for _, batch := range Chunk(writeRequests, 25) {
    if err := writeBatch(ctx, batch); err != nil {
        return err
    }
}
```

Or use `slices.Chunk` if available on your Go version.

---

## 14. Testing

### Framework: Standard Library Only

Use `testing` from the standard library. No testify, no gomega, no ginkgo. Table-driven tests and `t.Helper()` cover nearly everything. If you need mocks, write them by hand or use a small code generator — a mock framework dependency is almost never worth it.

### Table-Driven Tests

The default test structure. Each case is a named struct in a slice:

```go
func TestGetPrimaryKeyName(t *testing.T) {
    tests := []struct {
        name     string
        platform PlatformType
        want     string
        wantErr  bool
    }{
        {name: "accounting", platform: PlatformAccounting, want: "companyAndInsight"},
        {name: "banking", platform: PlatformBanking, want: "extCompanyId"},
        {name: "unknown panics", platform: PlatformUnknown, wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := GetPrimaryKeyName(tt.platform)
            if tt.wantErr {
                if err == nil {
                    t.Fatal("expected error, got nil")
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if got != tt.want {
                t.Errorf("GetPrimaryKeyName(%v) = %q, want %q", tt.platform, got, tt.want)
            }
        })
    }
}
```

### Test Helpers

Use `t.Helper()` for functions that report failures on behalf of the caller:

```go
func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func assertEqual[T comparable](t *testing.T, got, want T) {
    t.Helper()
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
}
```

### Test Fixtures and Golden Files

For complex test data, use `testdata/` directories. For output comparison, use golden files:

```go
func TestRenderReport(t *testing.T) {
    got := renderReport(testInput)

    golden := filepath.Join("testdata", t.Name()+".golden")
    if *update {
        os.WriteFile(golden, []byte(got), 0644)
    }

    want, _ := os.ReadFile(golden)
    if got != string(want) {
        t.Errorf("output mismatch; run with -update to refresh golden files")
    }
}
```

### Fake Implementations Over Mocks

Write simple fake structs that satisfy interfaces. They're more readable and more maintainable than mock framework magic:

```go
type fakeAccountStore struct {
    accounts []Account
    err      error
}

func (f *fakeAccountStore) GetAccounts(ctx context.Context, tenant TenantID) ([]Account, error) {
    return f.accounts, f.err
}

// In test:
store := &fakeAccountStore{
    accounts: []Account{{ID: "1", Name: "Test"}},
}
svc := NewReconciler(store, slog.Default())
```

### Test Naming

Pattern: `TestFunctionName_Scenario` using sub-tests for cases:

```go
func TestReconciler_Run(t *testing.T) {
    t.Run("empty accounts returns early", func(t *testing.T) { ... })
    t.Run("mismatched totals returns error", func(t *testing.T) { ... })
    t.Run("successful reconciliation", func(t *testing.T) { ... })
}
```

### Parallel Tests

Mark tests as parallel when they don't share mutable state:

```go
func TestExpensiveComputation(t *testing.T) {
    t.Parallel()
    // ...
}
```

For table-driven tests, capture the loop variable and run subtests in parallel:

```go
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel()
        // ...
    })
}
```

---

## 15. Formatting & Layout

### gofmt Is Non-Negotiable

All code is formatted with `gofmt`. No exceptions, no arguments, no custom settings. Use `goimports` as a superset (handles import ordering too). Configure your editor to run it on save.

### Line Length

Go has no official line limit, but target ~100-120 characters for readability. Wrap function signatures and long expressions:

```go
func (s *SyncService) ProcessBatch(
    ctx context.Context,
    tenant TenantID,
    companyID CompanyID,
    items []SyncItem,
    opts ProcessOptions,
) (*BatchResult, error) {
```

### File Organization

Within a file, order declarations:

1. Package-level constants and variables
2. Types (structs, interfaces)
3. Constructor functions (`NewX`)
4. Methods grouped by receiver type
5. Package-level functions (helpers, utilities)

### One Type Per File (Usually)

Major types get their own file. Small related types (a struct and its constructor, an interface and a helper) can share a file. If a file exceeds ~400 lines, consider splitting.

### Comments: Focus on "Why"

Same as C#: comments explain *why*, not *what*. If the what/how isn't clear, improve the name:

```go
// Bad: restating the code
// Check if rate is greater than zero
if rate > 0 { ... }

// Good: explaining domain knowledge
// Fed data uses VEB (bolivar fuerte) instead of the current ISO code VES (bolivar soberano).
if strings.EqualFold(code, "VES") {
    return "VEB"
}
```

### Package Comments

Every package should have a doc comment in a `doc.go` or at the top of the primary file:

```go
// Package reconcile provides tools for reconciling account data
// between external platforms and the internal ledger.
package reconcile
```

### Exported Function Documentation

All exported functions, types, and methods have doc comments. Start with the name of the thing:

```go
// GetAccounts returns all accounts for the given tenant and company.
// It returns an empty slice (not nil) if no accounts are found.
func (s *Store) GetAccounts(ctx context.Context, tenant TenantID, company CompanyID) ([]Account, error) {
```

---

## 16. Naming

### Go Naming Conventions

These are non-negotiable — they're enforced by the compiler and community norms:

- **Exported** identifiers are `PascalCase`: `GetAccounts`, `SyncResult`, `ErrNotFound`
- **Unexported** identifiers are `camelCase`: `processItem`, `accountStore`, `defaultTimeout`
- **Acronyms** are all-caps: `ID`, `URL`, `HTTP`, `API` — not `Id`, `Url`, `Http`
- **Package names** are lowercase, single word when possible: `config`, `sync`, `ledger` — not `ledgerUtils`, `sync_helpers`
- **Interface names**: single-method interfaces use the `-er` suffix: `Reader`, `Writer`, `Closer`, `Fetcher`. Multi-method interfaces describe the role: `AccountStore`, `TokenProvider`

### Receiver Names

Use one or two letter abbreviations, consistent across all methods on a type. Never `self` or `this`:

```go
func (r *Reconciler) Run(ctx context.Context) error { ... }
func (r *Reconciler) validate() error { ... }

func (s *Store) GetAccounts(ctx context.Context) ([]Account, error) { ... }
```

### Variable Names

Short names for short scopes, descriptive names for long scopes:

```go
// Good: short scope, short name
for i, a := range accounts { ... }

// Good: longer scope, descriptive name
var activeAccounts []Account
for _, account := range allAccounts {
    if account.IsActive {
        activeAccounts = append(activeAccounts, account)
    }
}
```

### Don't Stutter

Package names qualify their exports. Don't repeat the package name in the type name:

```go
// Bad: config.ConfigOptions stutters
package config
type ConfigOptions struct { ... }

// Good: config.Options reads naturally
package config
type Options struct { ... }
```

---

## 17. Dependency Management

### Standard Library First

Before reaching for a third-party package, check if the standard library covers it. Go's stdlib is unusually comprehensive. Common cases where teams reach for dependencies unnecessarily:

- HTTP clients: `net/http` is excellent. You rarely need a wrapper.
- JSON: `encoding/json` covers most cases. Only reach for alternatives on hot paths with benchmarks.
- Logging: `log/slog` for CLI tools (see Section 11). `zap` is acceptable for web services.
- Testing: `testing` + table-driven tests covers 95% of needs.

### Acceptable Common Dependencies

These are fine to pull in without justification:

- `github.com/spf13/cobra` — CLI framework (mandatory for all tools, see Section 6)
- `github.com/aws/aws-sdk-go-v2` — AWS API access
- `github.com/jackc/pgx/v5` — PostgreSQL driver
- `golang.org/x/sync/errgroup` — parallel goroutine management
- `go.uber.org/zap` — structured logging for web services (not CLIs)

Everything else needs a reason. "It's popular" is not a reason.

### Keeping Dependencies Updated

Run `go get -u ./...` and `go mod tidy` regularly. Pin major versions in `go.mod`. Review changelogs for security patches.

---

## 18. Zero Values and Nullability

### Embrace the Zero Value

Go's zero values (`""`, `0`, `false`, `nil` for pointers/slices/maps) are part of the type system. Design types so that the zero value is useful:

```go
// Good: zero value is a valid, empty state
type BatchResult struct {
    Processed int
    Failed    int
    Errors    []error // nil = no errors
}

// r := BatchResult{} is already valid, means "nothing processed, no errors"
```

### Pointer Fields Mean "Optional"

Use pointer fields when the zero value is meaningful and you need to distinguish "unset" from "zero":

```go
type UpdateRequest struct {
    Name    *string  // nil = don't update, "" = set to empty
    Workers *int     // nil = don't update, 0 = valid value
    Active  *bool    // nil = don't update, false = valid value
}
```

### Validate at Boundaries

Same philosophy as the C# guide: eliminate nil/zero concerns at the edges. Inside the domain, types should carry only valid state:

```go
// Boundary: validate and reject
func (h *Handler) HandleSync(req *http.Request) error {
    var input SyncRequest
    if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
        return fmt.Errorf("invalid request body: %w", err)
    }
    if input.TenantID == "" {
        return &ValidationError{Field: "tenant_id", Message: "required"}
    }
    // domain functions receive validated, non-zero data
    return h.svc.Sync(req.Context(), TenantID(input.TenantID), CompanyID(input.CompanyID))
}
```

### Empty Collections Over Nil in JSON

When serializing for external consumers, initialize slices if `null` vs `[]` matters:

```go
type Response struct {
    Items []Item `json:"items"`
}

// If Items might be nil, initialize before marshaling:
if resp.Items == nil {
    resp.Items = []Item{}
}
```

---

## 19. AWS SDK Patterns

### Use SDK v2

All new code uses `aws-sdk-go-v2`. Do not use v1 (`aws-sdk-go`).

### SQS Long Polling

```go
out, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
    QueueUrl:            &queueURL,
    WaitTimeSeconds:     20,
    MaxNumberOfMessages: 10,
})
```

### S3 Pagination

Use the paginator helpers from SDK v2:

```go
paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
    Bucket: &bucket,
    Prefix: &prefix,
})

for paginator.HasMorePages() {
    page, err := paginator.NextPage(ctx)
    if err != nil {
        return fmt.Errorf("listing objects: %w", err)
    }
    for _, obj := range page.Contents {
        // process
    }
}
```

### DynamoDB Batch Operations

```go
// BatchWriteItem supports max 25 items
for _, batch := range Chunk(writeRequests, 25) {
    _, err := client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
        RequestItems: map[string][]types.WriteRequest{
            tableName: batch,
        },
    })
    if err != nil {
        return fmt.Errorf("batch write: %w", err)
    }
}
```

---

## 20. Patterns to Maintain

### Prefer Standard Library Over Hand-Rolled

Same principle as the C# guide: check whether Go's stdlib already provides the functionality before writing a utility. In particular:

- `slices` and `maps` packages replace many hand-rolled loops (Go 1.21+)
- `slog` replaces custom logging for CLI tools (Go 1.21+); `zap` remains appropriate for services
- `errors.Join` replaces custom multi-error types (Go 1.20+)
- `sync.OnceValue` replaces lazy initialization patterns (Go 1.21+)
- `http.NewServeMux` pattern matching replaces many router libraries (Go 1.22+)

### decimal for Money

Go's `float64` has the same problems as C#'s `double`. Use `shopspring/decimal` or a similar arbitrary-precision library for monetary calculations. Never `float64` for money:

```go
import "github.com/shopspring/decimal"

rate := decimal.NewFromString("0.0425")
monthly := rate.Div(decimal.NewFromInt(12))
```

### time.Time, Not int64

Represent timestamps as `time.Time`, not Unix epoch integers. Convert at boundaries (JSON serialization, database storage), not in domain logic.

### Performance Behind Good Names

Same as C#: a function named `GetExchangeRate` can use `unsafe.Pointer` arithmetic internally if profiling demands it. The caller sees a clean API. Optimize hot paths, not cold paths. Profile before optimizing.

### Composition Over Inheritance

Go doesn't have inheritance. Use embedding for shared structure, interfaces for shared behavior:

```go
// Embedding: shared fields
type BaseJob struct {
    TenantID  TenantID
    CompanyID CompanyID
    CreatedAt time.Time
}

type SyncJob struct {
    BaseJob
    Platform PlatformType
    Priority int
}

// Interface: shared behavior
type Processor interface {
    Process(ctx context.Context, job Job) error
}
```
