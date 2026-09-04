# Architecture

Composition + interface-based. The folder structure is what falls
out of the two rules below. This file is short on purpose.

## The two rules

1. **Composition.** `internal/di/` is the only place that knows
   the graph. No `init()`, no global singletons. Missing
   provider → build error.
2. **Interface-based.** Every port is declared by its consumer.
   Both adapters satisfy the port, and `var _ Port =
   (*Adapter)(nil)` makes port drift a build error.

## The tag rule

| Tag | Lives in |
|---|---|
| `gorm:"…"` | `internal/repositories/` |
| `json:"…"` | `internal/controllers/` |
| `// @…` swag | `internal/controllers/` + `internal/server/` |
| `koanf:"…"` | `internal/configurations/` |
| `wire.NewSet` | `internal/di/` |

If a tag leaks, the layer is wrong. Grep it.

## Test pyramid

| Layer | File | What it needs |
|---|---|---|
| Domain | `internal/domains/product/core_test.go` | `product.New()` |
| Use case | `internal/domains/product/service_test.go` | the in-memory adapter |
| Component | `cmd/server/main_test.go` | full wire, real PG |
| Repository | `internal/repositories/product_test.go` | real PG |
| Server middleware | `internal/server/middleware_test.go` | `httptest` |

The use case test wires the in-memory adapter directly. No
mockery, no `MockXxx`.

## Middleware order in `internal/server/server.go`

1. `requestIDMiddleware` — read or generate `X-Request-Id`, store
   on the gin context, echo back on the response.
2. `metricsMiddleware` — `http_requests_total{method,path,status}`,
   `http_request_duration_seconds{method,path}` histogram, and the
   `http_in_flight_requests` gauge. Uses `c.FullPath()` so the
   `path` label is the route pattern, not the raw URL.
3. `slogRequestLogger` — one structured line per request with
   method, path, status, duration, size, and request_id. Level
   follows status (5xx→Error, 4xx→Warn, else→Info).
4. `corsMiddleware` — no-op unless `cors_allowed_origins` is set.
5. `gin.Recovery` — last, so a panic still hits the access log.

## Request lifecycle

`cmd/server/main.go`:

1. Parse `APP_CONFIG` (default `config.yml`) and set the slog
   handler (tint for dev, JSON for `APP_LOG=json`).
2. Run Wire to build the application graph.
3. Run the HTTP server in a goroutine.
4. Block on a SIGINT/SIGTERM context.
5. On signal, build a `context.WithTimeout(parent, shutdown_timeout_sec)`,
   call `server.Shutdown(ctx)` to drain in-flight requests, release
   the DB pool via Wire's cleanup, exit 0.

`shutdown_timeout_sec` is clamped to `[1, 300]` seconds. Anything
still in flight past the deadline gets cut off; `server.shutdown.error`
lands in the log with the underlying error.

## Stable error contract

Every error response carries a stable `error_code` string. The set
lives in `internal/models/product.go`:

| code | HTTP | meaning |
|---|---|---|
| `INVALID_NAME` | 422 | empty / whitespace / > 200 chars |
| `INVALID_DESCRIPTION` | 422 | > 2000 chars |
| `INVALID_PRICE` | 422 | price ≤ 0 |
| `PRICE_TOO_LARGE` | 422 | price > 10,000,000 |
| `INVALID_PRICE_RANGE` | 422 | `min_price > max_price` |
| `INVALID_LIMIT` | 422 | limit < 0 or > 100 |
| `PRODUCT_NOT_FOUND` | 404 | id missing or soft-deleted |
| `REPOSITORY_FAILURE` | 500 | db or repo error |
| `INVALID_BODY` | 400 | JSON body fails to bind |
| `INVALID_ID` | 400 | path `:id` is empty |

`internal/domains/product/core_test.go` is the spec — every code
listed there has a unit test.

## Soft-delete semantics

`DELETE /product/{id}` is idempotent soft delete. `deleted_at` is a
nullable `TIMESTAMPTZ` column; `NULL` = live, anything else = tombstoned.
The repository filters on `deleted_at IS NULL` for every read path
(`GetByID`, `List`, `Patch`) and only flips the column on
`SoftDelete` when the row is currently live, so a second delete is
a no-op. The list returns one extra row (`limit + 1`) and uses the
last kept row's id as `next_cursor`, giving key-set pagination that
is stable across inserts.

## Proof

```bash
$ grep -rln 'gorm\|gin\|koanf\|swag\|wire' internal/domains/product/
# empty
```

The core has zero infrastructure in its import graph.

## NFR compliance

| Requirement | File |
|---|---|
| Component test (E2E) | `cmd/server/main_test.go` |
| Use case unit test | `internal/domains/product/service_test.go` |
| Domain / function test | `internal/domains/product/core_test.go` |
| Repository integration test | `internal/repositories/product_test.go` |
| Server middleware test | `internal/server/middleware_test.go` |
| Swagger at `/api-docs` | `internal/server/server.go` |
| How to start | `README.md` Quickstart |
| DI (composition root) | `internal/di/` (Wire) |
| PostgreSQL | `gorm.io/driver/postgres` + `DriverName: "pgx"` |
| Request tracing | `requestIDMiddleware` in `internal/server/middleware.go` |
| Graceful shutdown | `cmd/server/main.go` + `server.Shutdown` |
| Metrics | `metricsMiddleware` + `/metrics` in `internal/server/server.go` |
| CORS | `corsMiddleware` (only active when configured) |
