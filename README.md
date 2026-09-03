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
- `http://localhost:8081` — pgweb (Postgres admin)

## Endpoints

```
POST  /product         {name, description?, sale_price?, price}    → 201 {successful, error_code, data:{data1, data2}}
PATCH /product/{id}    {name?, description?, sale_price?, price?}  → 200 {successful, error_code}
GET   /healthz         → 200 {successful: true}
GET   /api-docs/index.html
```

Response envelope on every endpoint:

```json
{ "successful": true,  "error_code": "",    "data": { "data1": "...", "data2": "..." } }
{ "successful": false, "error_code": "...", "data": null }
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

## Test layers (no mocks)

| Layer | File | What it needs |
|---|---|---|
| Domain | `internal/domains/product/core_test.go` | `product.New()` |
| Use case | `internal/domains/product/service_test.go` | the in-memory adapter |
| Component | `cmd/server/main_test.go` | full wire, real PG |
| Repository | `internal/repositories/product_test.go` | real PG |

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

MIT.
