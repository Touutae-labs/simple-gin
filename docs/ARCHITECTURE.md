# Architecture — Ports & Adapters (Hexagonal)

This project is **Hexagonal Architecture** (a.k.a. **Ports &
Adapters**). It is **not** Clean Architecture. The two solve a
similar problem — keeping the business core independent of
infrastructure — but they prescribe different internal structure
and use different vocabulary. Mixing them muddles what we're
actually building, so this doc is precise about which one we are.

## What "Hexagonal / Ports & Adapters" means here

The structure is exactly the one in the article you cited:

- **Core** — the business logic. Knows nothing about HTTP,
  Postgres, GORM, Gin, koanf, swag, or Wire. Implemented in
  `internal/domains/product/`.
- **Ports** — interfaces that the core declares for what it
  needs from the outside (e.g. `product.Repository` for storage).
  Implemented next to the core: `internal/domains/product/repository.go`.
- **Adapters** — concrete implementations of the ports. One
  per environment:
  - **GORM/Postgres** (production) → `internal/repositories/`
  - **In-memory** (tests) → `internal/testhelpers/`

That's it. The internal structure of the core is intentionally left
open — Hexagonal does not prescribe concentric layers or a separate
use-case tier. Our core is one package per sub-domain with the
rule, the port, and the orchestrator; the orchestrator is **part
of the core**, not a layer above it.

## What this is **not**

It is **not** Clean Architecture. The differences that matter:

| Clean Architecture | This project (Hexagonal) |
|---|---|
| Concentric layers: Entities → Use Cases → Interface Adapters → Frameworks | One core + ports + adapters, no concentric layering |
| Use cases are a separate layer above entities | The orchestrator (`product.Service`) is part of the core, in the same package as the rule and the port |
| "Interface Adapters" is a labelled ring | "Adapters" is just wherever the port is implemented; one per environment |
| Entities and Use Cases often get their own folders/packages | One package per sub-domain; the rule, port, and orchestrator all live there |
| "Frameworks & Drivers" is the outermost ring and can include HTTP, DB, web frameworks | Frameworks don't get their own ring; they live in the adapter that uses them (Gin in `controllers/`, GORM in `repositories/`, etc.) |

If someone points at this project and says "that's clean
architecture with fewer layers", they're partially right and
substantially wrong. It's a different architecture with a
deliberately simpler internal structure.

## The two operating rules (not principles — rules)

1. **Composition** — the graph is wired in exactly one place
   (`internal/di/`). No `init()`, no global singletons, no
   `sync.Once`. Missing provider is a build error.
2. **Interface-based** — every port is declared by its consumer.
   Both adapters satisfy the port, and `var _ Port = (*Adapter)(nil)`
   makes port drift a build error.

These are the rules we actually enforce. The fact that they
produce code that *also* satisfies Hexagonal is the point of
Hexagonal — the rules are the practice, the architecture is the
shape the practice produces.

## "The tag's package is the adapter's package" rule

The greps below should return zero leaks. If a tag leaks, the
layer is wrong.

| Tag | Should only appear in |
|---|---|
| `gorm:"…"` | `internal/repositories/` |
| `json:"…"` | `internal/controllers/` |
| `// @…` swag | `internal/controllers/` + `internal/server/` |
| `koanf:"…"` | `internal/configurations/` |
| `wire.NewSet` | `internal/di/` |

```bash
$ grep -rln 'gorm:"'   --include='*.go' .   # only repositories/
$ grep -rln 'json:"'   --include='*.go' .   # only controllers/
$ grep -rln 'koanf:"'  --include='*.go' .   # only configurations/
$ grep -rln 'wire\.'   --include='*.go' .   # only internal/di/
$ grep -rln 'gorm\|gin\|koanf\|swag\|wire' internal/domains/product/  # nothing
```

The last one is the proof: the core has zero infrastructure in
its import graph.

## The port

```go
// internal/domains/product/repository.go — the consumer declares the port
type Repository interface {
    Create(ctx, in models.CreateInput) (string, error)
    GetByID(ctx, id string) (*models.Product, error)
    Patch(ctx, id string, in models.PatchInput) (*models.Product, error)
}

// Both adapters implement it; both fail to compile if the port drifts
var _ product.Repository = (*repositories.Product)(nil)      // production
var _ product.Repository = (*testhelpers.ProductMemory)(nil)   // tests
```

## Test pyramid — no mocks

| Layer | File | What it needs | What it doesn't |
|---|---|---|---|
| **Domain** | `core_test.go` | `product.New()` | DB, HTTP, anything |
| **Use case** | `service_test.go` | `svc.New(product.New(), testhelpers.NewProductMemory())` | DB, HTTP |
| **Component** | `cmd/server/main_test.go` | `di.Initialize(cfg, …)` + real PG | Nothing — full wire |
| **Repository** | `repositories/product_test.go` | real PG + AutoMigrate | service, controllers |

The use case test does **not** use mocks. The in-memory adapter
is the test adapter for the same port that the GORM adapter
satisfies in production. No mockery, no `MockXxx`.

## Layout

```
internal/
├── models/                # pure shapes, no tags
├── domains/product/       # core: rule + port + orchestrator (no infra)
├── repositories/          # adapter: GORM (only place gorm lives)
├── testhelpers/           # adapter: in-memory (tests)
├── controllers/           # adapter: Gin (only place json: lives)
├── server/                # engine + middleware + Swagger
├── configurations/        # YAML config
└── di/                    # Wire composition root
```

The "only place X lives" annotations are how you can tell at a
glance that the hexagon is intact. The core (domains/product/)
has nothing from any adapter's package in its import graph.

## NFR compliance

| Requirement | File |
|---|---|
| Component test (E2E) | `cmd/server/main_test.go` |
| Use case unit test | `internal/domains/product/service_test.go` |
| Domain / function test | `internal/domains/product/core_test.go` |
| Repository integration test | `internal/repositories/product_test.go` |
| Swagger at `/api-docs` | `internal/server/server.go` (swag annotations + mount) |
| How to start | `README.md` Quickstart |
| Composition (DI) | `internal/di/` (Wire) |
| PostgreSQL | `gorm.io/driver/postgres` + `DriverName: "pgx"` |
