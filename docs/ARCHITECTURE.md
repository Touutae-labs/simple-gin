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

The use case test wires the in-memory adapter directly. No
mockery, no `MockXxx`.

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
| Swagger at `/api-docs` | `internal/server/server.go` |
| How to start | `README.md` Quickstart |
| DI (composition root) | `internal/di/` (Wire) |
| PostgreSQL | `gorm.io/driver/postgres` + `DriverName: "pgx"` |
