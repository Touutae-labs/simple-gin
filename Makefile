.PHONY: install ci test dev wire swag mockery arch all clean verify

# Default target: bootstrap, generate, build, test, run.
# `make` with no args is the same as `make verify` so a fresh clone
# on a new machine does the right thing automatically.
.DEFAULT_GOAL := verify

# ---- Tools ----------------------------------------------------------------
# Install the CLIs the codegen steps need. Idempotent.
install:
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/vektra/mockery/v3@latest
	go install github.com/fe3dback/go-arch-lint@v1.14.0
	go mod download

# ---- CI -------------------------------------------------------------------
ci: lint test

lint:
	go vet ./...

test: verify

# ---- Codegen --------------------------------------------------------------
# Generate docs/docs.go (swagger), internal/di/wire_gen.go (wire).
# Both are git-ignored — they MUST run before build or test on a
# fresh clone. Anything that compiles or tests the service depends
# on these.
swag:
	swag init -g internal/server/server.go -o ./docs --parseDependency --parseInternal

wire:
	wire ./internal/di

# Generate testify mocks for every interface under
# internal/domains and internal/repositories. Config in
# .mockery.yaml matches the POS project layout. Output goes
# to internal/mocks/<package>/<interface>_mock.go and is
# committed to the repo (these are the test doubles, not
# throwaway build artifacts).
mockery:
	mockery

# Architecture check. Two layers of enforcement:
#   1. .go-arch-lint.yml → import boundary graph (run by
#      cmd/archcheck/main_test.go, which fails the build on
#      any violation).
#   2. cmd/archcheck/tagrule_test.go → struct-tag location
#      (gorm only in repositories, json only in controllers,
#      koanf only in configurations, wire only in di).
# Both run as part of `go test ./cmd/archcheck/...` which is
# picked up by the `test` target below.
arch:
	go test -count=1 ./cmd/archcheck/...

# ---- Dev ------------------------------------------------------------------
# dev depends on swag + wire so `make dev` on a fresh clone just
# works. The build target depends on the same so `go build` isn't
# called before the generated files exist.
dev: swag wire mockery
	go run ./cmd/server

# ---- Full from-scratch verification --------------------------------------
# Everything: install tools, generate, lint, test, build, run a
# dry-run smoke against /healthz. Safe to run on a clean checkout
# with no Postgres — DB tests will skip with a clear message.
verify: install swag wire mockery arch lint build smoke

build: swag wire mockery arch
	go build -o /tmp/simple-gin-build-check ./cmd/server
	@rm -f /tmp/simple-gin-build-check
	@echo "build OK"

# Smoke test: build the binary, start it briefly, hit /healthz, kill it.
# Requires Postgres (skip if absent).
smoke:
	@if [ -z "$$(command -v pg_isready 2>/dev/null)" ] || ! pg_isready -h localhost -p 5432 >/dev/null 2>&1; then \
		echo "smoke: Postgres not reachable on localhost:5432, skipping live smoke"; \
	else \
		set -e; \
		trap 'kill $$(cat /tmp/simple-gin-smoke.pid 2>/dev/null) 2>/dev/null || true' EXIT; \
		APP_CONFIG=config.yml go run ./cmd/server & echo $$! > /tmp/simple-gin-smoke.pid; \
		for i in $$(seq 1 30); do \
			if curl -fsS http://localhost:8080/healthz >/dev/null 2>&1; then \
				echo "smoke OK: /healthz returned 200"; \
				exit 0; \
			fi; \
			sleep 0.5; \
		done; \
		echo "smoke FAIL: /healthz did not return 200 within 15s"; \
		exit 1; \
	fi

# ---- Cleanup --------------------------------------------------------------
clean:
	rm -f docs/docs.go docs/swagger.json docs/swagger.yaml
	rm -f internal/di/wire_gen.go
	rm -rf internal/mocks
	rm -f /tmp/simple-gin-build-check
	rm -f /tmp/simple-gin-smoke.pid
