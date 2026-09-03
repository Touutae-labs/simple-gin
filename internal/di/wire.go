//go:build wireinject
// +build wireinject

// Package di is the composition root. wire.go (this file) is the
// spec compiled by `wire`; wire_gen.go is the generated code
// compiled by `go build` / `go test`. Regenerate with `make wire`.
package di

import (
	"github.com/google/wire"

	"github.com/Touutae-labs/simple-gin/internal/configurations"
	"github.com/Touutae-labs/simple-gin/internal/controllers"
	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/server"
)

var serverSet = wire.NewSet(
	wire.FieldsOf(new(configurations.Config), "Server"),
	server.BuildServerConfig,
	provideServer,
)

var controllersSet = wire.NewSet(
	controllers.NewHealthController,
	controllers.NewProductController,
	controllers.NewControllers,
)

var productSet = wire.NewSet(
	product.New,
	product.NewService,
)

var repositorySet = wire.NewSet(
	provideProductRepository,
)

var dbSet = wire.NewSet(
	wire.FieldsOf(new(configurations.Config), "Database"),
	provideDB,
)

func Initialize(cfg configurations.Config, title server.ServerTitle, version server.ServerVersion) (*Application, func(), error) {
	wire.Build(
		wire.Struct(new(Application), "*"),
		dbSet,
		repositorySet,
		productSet,
		controllersSet,
		serverSet,
	)
	return nil, nil, nil
}
