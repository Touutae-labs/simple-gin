# simple-gin

Gin + PostgreSQL service. Composition + interface-based.

## Quickstart

```bash
docker compose up -d             # postgres on :5432, pgweb on :8081
make install                     # wire + swag
make swag && make wire
make dev                         # :8080
```

Open in browser:

- `http://localhost:8080` — redirects to Swagger
- `http://localhost:8080/api-docs/index.html` — Swagger UI
- `http://localhost:8080/metrics` — Prometheus metrics
- `http://localhost:8081` — pgweb (Postgres admin)

## Endpoints

```
POST   /product         {name, description?, sale_price?, price}                  → 201 envelope
GET    /product         ?cursor&limit&name&min_price&max_price                    → 200 {items, next_cursor, limit}
GET    /product/{id}                                                          → 200 product
PATCH  /product/{id}    {name?, description?, sale_price?, price?}                → 200 envelope
DELETE /product/{id}                                                          → 204 (idempotent soft delete)
GET    /healthz                                                                → 200 envelope
GET    /metrics                                                                → 200 text/plain (Prometheus)
GET    /api-docs/index.html                                                    → Swagger UI
```

### Pagination

Cursor-based. Pass `?limit=20` for page size, `?cursor=<id>` to fetch
the next page after the last one. `next_cursor == ""` in the response
means "no more pages". Default limit is 20, hard cap is 100
(`?limit=999` returns 422 `INVALID_LIMIT`).

### Filters

All optional, combined with `AND`:

- `?name=espresso` — case-insensitive `ILIKE '%name%'`
- `?min_price=10&max_price=100` — inclusive bounds; `min > max` returns 422 `INVALID_PRICE_RANGE`

### Soft delete

`DELETE /product/{id}` sets `deleted_at = now()`. Subsequent
`GET /product/{id}` and `PATCH /product/{id}` return 404
`PRODUCT_NOT_FOUND`. List excludes them. Deleting an
already-deleted row is a no-op (returns 204).

### Request tracing

Every request gets an `X-Request-Id` header. The middleware echoes
whatever you send and generates a UUID if you don't. The id appears
in:

- the response header (so curl can correlate)
- the structured access log (one line per request, level follows status)
- the `/metrics` per-request labels (when instrumentation is on)

### Response envelope

Mutation endpoints (POST, PATCH) and Healthz return:

```json
{ "successful": true,  "error_code": "",    "data": { "data1": "...", "data2": "..." } }
{ "successful": false, "error_code": "...", "data": null }
```

List endpoint returns its own paginated shape:

```json
{ "items": [ProductResponse, ...], "next_cursor": "uuid-or-empty", "limit": 20 }
```

### Error codes

Stable machine-readable strings on every error response:

| code | HTTP | meaning |
|---|---|---|
| `INVALID_NAME` | 422 | name empty, whitespace-only, or > 200 chars |
| `INVALID_DESCRIPTION` | 422 | description > 2000 chars |
| `INVALID_PRICE` | 422 | price ≤ 0 |
| `PRICE_TOO_LARGE` | 422 | price > 10,000,000 |
| `INVALID_PRICE_RANGE` | 422 | min_price > max_price |
| `INVALID_LIMIT` | 422 | limit < 0 or > 100 |
| `PRODUCT_NOT_FOUND` | 404 | id doesn't exist or is soft-deleted |
| `REPOSITORY_FAILURE` | 500 | db or repo error |
| `INVALID_BODY` | 400 | JSON body can't be parsed |
| `INVALID_ID` | 400 | path :id is empty |

### Configuration

`config.yml` keys (read via `APP_CONFIG=path/to/file.yml`):

```yaml
server:
  port: "8080"
  max_payload_size_kb: 4096
  timeout_seconds: 30
  base_url: "http://localhost:8080"
  shutdown_timeout_sec: 25     # drain in-flight requests on SIGTERM
  cors_allowed_origins: ""     # comma-separated; empty = no CORS headers
  metrics_enabled: true

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  dbname: "simple_gin"
  sslmode: "disable"
  auto_migrate: true
```

## The two rules

1. **Composition.** `internal/di/` is the only place that knows
   the graph. No `init()`, no global singletons, no `sync.Once`.
   Missing provider → build error.
2. **Interface-based.** Every port is declared by its consumer.
   Both adapters satisfy the port, and `var _ Port = (*Adapter)(nil)`
   catches drift at compile time.

## The tag rule

```
gorm:"…"         → only in internal/repositories/
json:"…"         → only in internal/controllers/
koanf:"…"        → only in internal/configurations/
wire.NewSet      → only in internal/di/
```

If a tag leaks, the layer is wrong. Grep it.

## Middleware stack (in order)

1. `requestIDMiddleware` — X-Request-Id propagation
2. `metricsMiddleware` — Prometheus counters + histograms + in-flight gauge
3. `slogRequestLogger` — one structured log line per request with method, path, status, duration, size, request_id
4. `corsMiddleware` — only sets headers if `cors_allowed_origins` is non-empty
5. `gin.Recovery` — last, so a panic still hits the access log

`/metrics` exposes the registry built in step 2 (disabled when
`metrics_enabled: false`). All three collectors are local to the
server (not the global default registry), so re-running tests
doesn't accumulate duplicates.

## Graceful shutdown

On SIGINT or SIGTERM, `cmd/server/main.go` calls `server.Shutdown`
with a deadline of `shutdown_timeout_sec` (default 25s, max 300s).
In-flight requests are drained, the engine is closed, the DB pool
is released, and the process exits 0. Anything still running past
the deadline is cut off — log shows `server.shutdown.error` if
that happens.

## Test layers (no mocks)

| Layer | File | What it needs |
|---|---|---|
| Domain | `internal/domains/product/core_test.go` | `product.New()` |
| Use case | `internal/domains/product/service_test.go` | the in-memory adapter |
| Component | `cmd/server/main_test.go` | full wire, real PG |
| Repository | `internal/repositories/product_test.go` | real PG |
| Server middleware | `internal/server/middleware_test.go` | httptest |

The use case test wires the in-memory adapter directly. No
mockery, no `MockXxx`. Repository and component tests skip
cleanly if Postgres is unreachable.

## Layout

```
internal/
├── models/                # shapes
├── domains/product/       # core: rule + port + orchestrator
├── repositories/          # GORM adapter
├── testhelpers/           # in-memory adapter
├── controllers/           # Gin adapter
├── server/                # engine + middleware + Swagger
├── configurations/        # YAML config + DSN
└── di/                    # Wire composition root
```

`grep gorm internal/domains/product/` returns nothing. The core
has zero infrastructure in its import graph.

## Make targets

`install` `ci` `dev` `wire` `swag` `all` `clean`

## License

AGPL-3.0 — see `LICENSE`. `NOTICE.md` declares an AI/ML training
ban as an additional term under AGPL §7.
