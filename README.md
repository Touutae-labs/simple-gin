# simple-gin

Gin + PostgreSQL product service. **Hexagonal / Ports & Adapters**,
not Clean Architecture. The four NFR test layers are met, Swagger
is at `/api-docs`, and every adapter lives in exactly one package.

## Quickstart

```bash
docker compose up -d             # postgres on :5432
make install                     # wire + swag tools
make swag && make wire           # generate docs + DI graph
make dev                         # :8080, Swagger at /api-docs/index.html
```

## Endpoints

```
POST  /product         {name, description?, sale_price?, price}    → 201 {successful, error_code, data:{data1, data2}}
PATCH /product/{id}    {name?, description?, sale_price?, price?}  → 200 {successful, error_code}
GET   /healthz                                                        → 200 {successful: true}
GET   /api-docs/index.html                                            → Swagger UI
```

Response envelope on every endpoint:

```json
{ "successful": true,  "error_code": "",    "data": { "data1": "...", "data2": "..." } }
{ "successful": false, "error_code": "...", "data": null }
```

## Test layers

| Layer | File | Run |
|---|---|---|
| Domain (function) | `internal/domains/product/core_test.go` | `go test ./internal/domains/product` |
| Use case (orchestrator) | `internal/domains/product/service_test.go` | same package |
| Component (E2E) | `cmd/server/main_test.go` | `go test ./cmd/server` |
| Repository (integration) | `internal/repositories/product_test.go` | `go test ./internal/repositories` |

Repository and component tests need a Postgres on the default DSN
(`host=localhost user=postgres password=postgres dbname=simple_gin_test
port=5432 sslmode=disable`); they skip cleanly if it's not there.

## What this is

**Hexagonal / Ports & Adapters.** A core that knows nothing about
infrastructure, ports (interfaces) declared by the core, and
adapters that implement those ports — one per environment:

| Concern | Lives in |
|---|---|
| Core (rule + port + orchestrator) | `internal/domains/product/` |
| GORM/Postgres adapter (production) | `internal/repositories/` |
| In-memory adapter (tests) | `internal/testhelpers/` |
| Gin adapter (HTTP) | `internal/controllers/` + `internal/server/` |
| Composition root | `internal/di/` |

The two rules we actually enforce:

1. **Composition** — the graph is wired in exactly one place
   (`internal/di/`). No `init()`, no global singletons. Missing
   provider → build error.
2. **Interface-based** — every port is declared by its consumer.
   Both adapters satisfy the port; `var _ Port = (*Adapter)(nil)`
   catches drift at compile time.

## What this is **not**

It is **not** Clean Architecture. Clean prescribes concentric
layers (Entities → Use Cases → Interface Adapters → Frameworks)
and a separate use-case tier. This project has one core package
per sub-domain and no concentric layering. See
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the table
that spells out the difference.

## "The tag's package is the adapter's package"

| Tag | Lives in |
|---|---|
| `gorm:"…"` | `repositories/` |
| `json:"…"` | `controllers/` |
| `// @…` swag | `controllers/` + `server/` |
| `koanf:"…"` | `configurations/` |
| `wire.NewSet` | `di/` |

Grep it. If a tag leaks, the layer is wrong.

## Layout

```
internal/
├── models/                # pure shapes (no tags)
├── domains/product/       # core: rule + port + orchestrator
├── repositories/          # GORM adapter (only place gorm lives)
├── testhelpers/           # in-memory adapter
├── controllers/           # Gin adapter (only place json: lives)
├── server/                # engine + middleware + Swagger
├── configurations/        # YAML config + DSN
└── di/                    # Wire composition root
```

## Make targets

`install` `ci` `dev` `wire` `swag` `all` `clean`

## License

MIT.
