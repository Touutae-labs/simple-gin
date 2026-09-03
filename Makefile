.PHONY: install ci test dev wire swag all clean

# ---- Tools ----------------------------------------------------------------
install:
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go mod download

# ---- CI -------------------------------------------------------------------
ci: lint test

lint:
	go vet ./...

test:
	go test -race -count=1 ./...

# ---- Dev ------------------------------------------------------------------
dev:
	go run ./cmd/server

# ---- Codegen --------------------------------------------------------------
wire:
	wire ./internal/di

swag:
	swag init -g internal/server/server.go -o ./docs --parseDependency --parseInternal

# ---- Full from-scratch verification --------------------------------------
all: install swag wire ci

clean:
	rm -f docs/docs.go docs/swagger.json docs/swagger.yaml
	rm -f internal/di/wire_gen.go
