---
uuid: "8b2a1ee1-fadc-5422-b693-d67aa5040ff0"
kind: "research"
product_uuid: "aa14cb8f-ba36-5afe-abc9-40d26b422f89"
portal_url: "https://joel.holmes.haus/discovery/8b2a1ee1-fadc-5422-b693-d67aa5040ff0"
file: "docs/research/archaea_extensions_design.md"
body_sha256: "74708ad7368a0d602e80cf9a33544363cf62d667f458684c13a243070dfaf29d"
source: "narwhal-catalog:joel.holmes.haus/research/archaea_extensions_design.md"
parent: "https://github.com/holmes89/narwhal/blob/main/designs/joel.holmes.haus/system.md"
---

# Archaea Extension Design

Research date: 2026-03-23
Status: Design only — no implementation until test phase is complete.

This document identifies abstractions that are copy-pasted across every generated service and belong as first-class packages in `github.com/holmes89/archaea`. The analogy is Spring Boot: the framework handles wiring, lifecycle, and cross-cutting concerns so application code only needs to express domain logic.

---

## What Archaea Currently Owns

```
archaea/
  base/   — Entity, Repository[T], Service[T], GenericService[T],
             GenericGRPCService[T], GenericConsumer[T],
             Consumer[T], Producer[T], request/response interfaces
  kafka/  — Conn, Consumer[T], Producer[T]
```

What it does NOT own, but every generated service copy-pastes verbatim:

| Pattern | Where it lives today |
|---|---|
| Database connection + retry + goose migrations | `lib/repo/conn.go` in every project |
| h2c API server bootstrap + graceful shutdown | `cmd/api/main.go` in every project |
| Worker bootstrap + SIGINT + consumer lifecycle | `cmd/worker/main.go` in every project |
| CORS middleware wiring for ConnectRPC | `withCORS()` in every `cmd/api/main.go` |
| Health check endpoint | Hardcoded `fmt.Fprintf(w, "Server is running")` in every api main |
| Env var parsing | Raw `os.Getenv()` scattered through every main |
| Error-to-Connect-status-code mapping | Absent; bare errors returned everywhere |
| `log.Printf` as only observability | Scattered throughout `base/grpc.go` and service stubs |

---

## Proposed New Packages

---

### 1. `archaea/postgres`

**Spring Boot analogy:** DataSource auto-configuration + Flyway/Liquibase integration.

**Problem today:** Every service has an identical `lib/repo/conn.go`:
- `sql.Open("postgres", ...)`
- retry loop (3 attempts, 10s sleep)
- `goose.Up` from embedded migrations
- `conn.Ping()` health signal

This is ~77 lines copy-pasted with no variation. The only per-project difference is the embedded `migrations/*.sql` filesystem — which can be passed in as a parameter.

**Proposed API:**

```go
package postgres

// Conn wraps database/sql with lifecycle management.
type Conn struct { /* unexported */ }

// Open connects to PostgreSQL with exponential retry, then runs migrations
// from the provided embed.FS. Ready to use when it returns.
func Open(dsn string, migrations embed.FS, opts ...Option) (*Conn, error)

// DB returns the underlying *sql.DB for use with query builders.
func (c *Conn) DB() *sql.DB

// Ping reports whether the database is reachable (used by health checks).
func (c *Conn) Ping(ctx context.Context) error

// Close releases the connection pool.
func (c *Conn) Close() error

// Option configures Open behavior.
type Option func(*config)

func WithRetryAttempts(n int) Option
func WithRetryDelay(d time.Duration) Option
func WithMigrationsDisabled() Option   // for workers that shouldn't migrate
```

**What moves out of every `lib/repo/conn.go`:** The entire file. Each project's `conn.go` becomes a thin type alias or is deleted entirely — projects just call `postgres.Open(dsn, embedMigrations)`.

---

### 2. `archaea/server`

**Spring Boot analogy:** Embedded Tomcat/Netty + `@SpringBootApplication` bootstrap.

**Problem today:** Every `cmd/api/main.go` is ~130 lines of identical:
1. Parse `DATABASE_URL`, `KAFKA_BROKERS` from env
2. Connect to PostgreSQL
3. Connect to Kafka
4. Create `http.ServeMux`
5. Register N handlers via `reg{Entity}()` functions (the only varying part)
6. Add `/health` endpoint
7. Wrap mux with logging middleware
8. Start `h2c.NewHandler(mux, &http2.Server{})` on `:9000`
9. Trap SIGINT
10. Log termination

**Proposed API:**

```go
package server

// APIServer is an opinionated h2c ConnectRPC server with health checks,
// CORS, structured logging, and graceful shutdown built in.
type APIServer struct { /* unexported */ }

// New creates an APIServer. Call Register to mount handlers, then Run.
func New(opts ...Option) *APIServer

// Register mounts a ConnectRPC handler at its canonical path with CORS applied.
// path and handler come directly from the generated Connect handler constructor,
// e.g.: srv.Register(servicesv1connect.NewResourceServiceHandler(resourceSvc))
func (s *APIServer) Register(path string, handler http.Handler)

// Handle mounts an arbitrary http.Handler at path (for non-Connect routes).
func (s *APIServer) Handle(path string, handler http.Handler)

// AddHealthCheck registers a named check run on GET /health.
// The server returns 200 only when all checks pass.
func (s *APIServer) AddHealthCheck(name string, check health.Checker)

// Run starts the server and blocks until a signal (SIGINT/SIGTERM) is received,
// then performs graceful shutdown within the configured timeout.
func (s *APIServer) Run() error

// Option configures the server.
type Option func(*config)

func WithAddr(addr string) Option              // default ":9000"
func WithShutdownTimeout(d time.Duration) Option
func WithLogger(l *slog.Logger) Option
func WithCORSAllowedOrigins(origins []string) Option
```

**Usage in generated `cmd/api/main.go`:**

```go
// Before: ~130 lines
// After: ~20 lines
func main() {
    db, _ := postgres.Open(os.Getenv("DATABASE_URL"), embedMigrations)
    kconn := kafka.NewConn(strings.Split(os.Getenv("KAFKA_BROKERS"), ","))

    srv := server.New()
    srv.AddHealthCheck("postgres", db)

    regLabel(srv, db, kconn)
    regResource(srv, db, kconn)

    log.Fatal(srv.Run())
}
```

The `reg{Entity}()` functions remain (they're the domain-specific part), but they receive `*server.APIServer` instead of `*http.ServeMux`, and call `srv.Register(...)` instead of `mux.Handle(withCORS(...))`.

---

### 3. `archaea/worker`

**Spring Boot analogy:** `@Scheduled` + `SmartLifecycle` + application context shutdown hooks.

**Problem today:** Every `cmd/worker/main.go` is ~60 lines of identical:
1. Connect to PostgreSQL (often with `migrate=false`)
2. Connect to Kafka
3. Instantiate N services
4. Start N consumers
5. Trap SIGINT via `signal.Notify`
6. On signal: iterate consumers and call `.Close()`

**Proposed API:**

```go
package worker

// Worker coordinates the lifecycle of one or more background processes.
type Worker struct { /* unexported */ }

// New creates a Worker.
func New(opts ...Option) *Worker

// Register adds a Runnable to the worker. All registered runnables are
// started when Run is called. They are stopped in reverse registration
// order when a shutdown signal is received.
type Runnable interface {
    Close()
}

func (w *Worker) Register(name string, r Runnable)

// Run starts all registered runnables and blocks until SIGINT/SIGTERM.
// Returns after all runnables have been stopped.
func (w *Worker) Run() error

type Option func(*config)

func WithShutdownTimeout(d time.Duration) Option
func WithLogger(l *slog.Logger) Option
func WithSignals(sigs ...os.Signal) Option
```

**What this replaces:** The signal-trap + consumer-close loop that is copy-pasted verbatim in every worker main. The consumer instances returned by `kafka.NewConsumer(...)` already satisfy `Runnable` (they have `Close()`), so no changes to existing consumer code.

---

### 4. `archaea/middleware`

**Spring Boot analogy:** `OncePerRequestFilter` chain / `WebMvcConfigurer`.

**Problem today:** `withCORS()` is defined identically in every `cmd/api/main.go`:

```go
func withCORS(h http.Handler) http.Handler {
    middleware := cors.New(cors.Options{
        AllowedOrigins: []string{"*"},
        AllowedMethods: connectcors.AllowedMethods(),
        AllowedHeaders: connectcors.AllowedHeaders(),
        ExposedHeaders: connectcors.ExposedHeaders(),
    })
    return middleware.Handler(h)
}
```

This is folded into `server.APIServer` automatically when using `Register()`, so most services never need to import `middleware` directly. But it should exist as a standalone package for projects that manage their own mux (e.g., pogona with Traefik, ibis with Nginx in front).

**Proposed API:**

```go
package middleware

// ConnectCORS returns a handler that applies the correct CORS headers for
// ConnectRPC / gRPC-Web browser clients.
func ConnectCORS(h http.Handler, opts ...CORSOption) http.Handler

// RequestLogger returns a handler that logs method, path, and duration
// using the provided slog.Logger.
func RequestLogger(h http.Handler, logger *slog.Logger) http.Handler

type CORSOption func(*cors.Options)

func WithAllowedOrigins(origins ...string) CORSOption
```

The `server` package uses these internally. Direct use is only needed for unusual server setups.

---

### 5. `archaea/health`

**Spring Boot analogy:** Spring Actuator `/actuator/health` with pluggable `HealthIndicator` beans.

**Problem today:** Every service has:
```go
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Server is running")
})
```

This is useless for real health probes — it always returns 200 regardless of DB or Kafka state.

**Proposed API:**

```go
package health

// Checker is implemented by any dependency that can report its own health.
// postgres.Conn and kafka.Conn would implement this.
type Checker interface {
    Ping(ctx context.Context) error
}

// Handler returns an http.Handler for GET /health.
// Returns 200 + JSON when all checks pass; 503 + JSON with failing checks otherwise.
func Handler(checks map[string]Checker) http.Handler

// Response is the JSON shape returned by Handler.
type Response struct {
    Status string            `json:"status"` // "ok" or "degraded"
    Checks map[string]string `json:"checks"` // name → "ok" or error message
}
```

`postgres.Conn` and `kafka.Conn` each implement `health.Checker` by adding a `Ping(ctx) error` method. The `server.APIServer.AddHealthCheck(name, checker)` method delegates to this package.

---

### 6. `archaea/config`

**Spring Boot analogy:** `@ConfigurationProperties` + `@Value` + `application.properties`.

**Problem today:** Every `main()` has raw `os.Getenv()` calls with no defaults, no validation, and no documentation of what's required. The pattern is identical across all 12 services.

**Proposed API:**

```go
package config

// Base holds the env vars that every generated service needs.
// Embed this in service-specific config structs.
type Base struct {
    DatabaseURL  string `env:"DATABASE_URL,required"`
    KafkaBrokers string `env:"KAFKA_BROKERS,required"`
    Addr         string `env:"ADDR" envDefault:":9000"`
}

// Brokers splits KafkaBrokers on commas and returns a []string.
func (b Base) Brokers() []string

// Load populates T from environment variables using struct tags.
// Returns an error listing all missing required vars.
func Load[T any](dst *T) error
```

Services define their own config struct:

```go
// cmd/api/main.go in weevil
type Config struct {
    config.Base
    GoogleBooksAPIKey string `env:"GOOGLE_BOOKS_API_KEY"`
    TemporalHost      string `env:"TEMPORAL_HOST" envDefault:"localhost:7233"`
    MinioEndpoint     string `env:"MINIO_ENDPOINT"`
}
```

This replaces scattered `os.Getenv()` calls with a single `config.Load(&cfg)` call that fails fast on startup if required vars are missing, and documents all config in one place per binary.

Implementation note: this can wrap `github.com/caarlos0/env` (a lightweight, well-tested library) rather than rolling a custom parser.

---

### 7. `archaea/errors`

**Spring Boot analogy:** `@ExceptionHandler` + `ProblemDetail` (RFC 9457).

**Problem today:** `GenericGRPCService` returns bare `error` values. ConnectRPC maps unrecognized errors to `codes.Unknown` (500). Common cases like "not found" or "already exists" silently become 500s.

**Proposed API:**

```go
package errors

// Sentinel error types that map to well-known Connect/gRPC status codes.

var NotFound = &Error{code: connect.CodeNotFound}
var AlreadyExists = &Error{code: connect.CodeAlreadyExists}
var InvalidArgument = &Error{code: connect.CodeInvalidArgument}
var Unauthenticated = &Error{code: connect.CodeUnauthenticated}
var PermissionDenied = &Error{code: connect.CodePermissionDenied}
var Internal = &Error{code: connect.CodeInternal}

// Wrap annotates a sentinel with a message.
func Wrap(base *Error, msg string, args ...any) error
// e.g.: errors.Wrap(errors.NotFound, "book %s not found", id)

// ToConnect converts any error to a *connect.Error with the appropriate code.
// If the error is already a *connect.Error, it is returned as-is.
// If it wraps an archaea errors.Error, the code is used.
// Otherwise codes.Internal is used.
func ToConnect(err error) *connect.Error

// IsNotFound, IsAlreadyExists, etc. for use in tests.
func IsNotFound(err error) bool
func IsAlreadyExists(err error) bool
```

`GenericGRPCService` uses `errors.ToConnect(err)` before returning to ConnectRPC. Repository implementations use `errors.Wrap(errors.NotFound, ...)` when `sql.ErrNoRows` is encountered. This gives every service correct HTTP status codes from day one.

---

### 8. `archaea/testing`

**Spring Boot analogy:** `@SpringBootTest`, `@MockBean`, `TestContainers` integration.

**Problem today:** No test infrastructure exists anywhere in the ecosystem. Every test that gets written will need the same setup: mock repositories, a real Postgres container, fake Kafka.

**Proposed API (design sketch — lowest priority, but highest leverage when test phase starts):**

```go
package testing

// MockRepository is a generic in-memory Repository[T] for unit tests.
// Pre-populated via WithEntities; records calls for assertion.
type MockRepository[T base.Entity] struct { /* ... */ }

func NewMockRepository[T base.Entity](entities ...T) *MockRepository[T]

// Recorded calls
func (m *MockRepository[T]) CreateCalls() []T
func (m *MockRepository[T]) GetCalls() []string      // ids
func (m *MockRepository[T]) DeleteCalls() []string

// MockProducer records Publish calls without hitting Kafka.
type MockProducer[T base.Entity] struct { /* ... */ }

func NewMockProducer[T base.Entity]() *MockProducer[T]
func (p *MockProducer[T]) Published() []T

// PostgresContainer starts a real Postgres instance via Testcontainers.
// Returns a DSN ready to pass to postgres.Open.
// Cleans up automatically when the test ends.
func PostgresContainer(t *testing.T) string
```

This package intentionally has no production dependencies on Testcontainers — it's only imported in `_test.go` files. The `MockRepository` covers unit tests; `PostgresContainer` covers integration tests for the repo layer.

---

## Revised Archaea Package Layout

```
archaea/
  base/       — Entity, Repository[T], Service[T], GenericService[T],
                GenericGRPCService[T], GenericConsumer[T],
                Consumer[T], Producer[T], request/response interfaces
                [existing — no changes]

  kafka/      — Conn, Consumer[T], Producer[T]
                [existing — Conn gains Ping(ctx) error to satisfy health.Checker]

  postgres/   — NEW: Conn wrapping database/sql + goose migrations + retry
  server/     — NEW: APIServer (h2c, CORS, health, graceful shutdown)
  worker/     — NEW: Worker (Runnable lifecycle + signal handling)
  middleware/ — NEW: ConnectCORS, RequestLogger
  health/     — NEW: Checker interface, Handler, Response
  config/     — NEW: Base struct, Load[T]
  errors/     — NEW: typed errors, ToConnect, sentinel values
  testing/    — NEW: MockRepository[T], MockProducer[T], PostgresContainer
```

---

## Impact on Beaver Templates

Once archaea gains these packages, beaver's generated code simplifies significantly:

| Template | Before | After |
|---|---|---|
| `cmd/api/main.go` | ~130 lines, copy-pasted | ~20 lines using `server.New()` |
| `cmd/worker/main.go` | ~60 lines, copy-pasted | ~15 lines using `worker.New()` |
| `lib/repo/conn.go` | ~77 lines per project | Deleted; projects call `postgres.Open()` |
| `withCORS()` | Copy-pasted in every api main | Removed; built into `server.Register()` |
| `/health` handler | Useless string response | Real dependency checks via `health.Checker` |
| Error handling | Absent | `errors.ToConnect()` in grpc service template |
| Env parsing | `os.Getenv()` scattered | Single `config.Load(&cfg)` call |

The net effect is that beaver generates less code, the generated code is shorter, and the shared behavior (retry logic, CORS headers, shutdown sequencing, error codes) is maintained in one place rather than 12.

---

## Implementation Order (when ready)

1. `postgres` — self-contained, no new dependencies, immediately removes the most duplicated file
2. `errors` — self-contained, improves every service's HTTP status codes immediately
3. `health` — small, composes with `postgres` and `kafka` cleanly
4. `config` — small, add `caarlos0/env` dependency
5. `middleware` — trivial extraction of existing `withCORS()`
6. `server` — composes `middleware`, `health`; replaces the largest boilerplate block
7. `worker` — composes with `server`, replaces the second largest boilerplate block
8. `testing` — last, because it requires test phase to be complete first anyway

Each step is independently shippable. Steps 1–5 can be adopted service-by-service without touching beaver. Steps 6–7 require beaver template updates to see the full benefit.
