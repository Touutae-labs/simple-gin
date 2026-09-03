package di

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Touutae-labs/simple-gin/internal/configurations"
	"github.com/Touutae-labs/simple-gin/internal/controllers"
	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/repositories"
	"github.com/Touutae-labs/simple-gin/internal/server"
)

// provideDB opens the Postgres connection via pgx's database/sql
// driver and GORM's postgres driver. AutoMigrate runs when
// cfg.AutoMigrate is true.
func provideDB(cfg configurations.DatabaseConfig) (*gorm.DB, func(), error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DriverName: "pgx",
		DSN:        cfg.DSN(),
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return nil, nil, fmt.Errorf("open db: %w", err)
	}
	if cfg.AutoMigrate {
		if err := db.AutoMigrate(repositories.AllModels()...); err != nil {
			return nil, nil, fmt.Errorf("automigrate: %w", err)
		}
	}
	cleanup := func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
	return db, cleanup, nil
}

// provideProductRepository returns the GORM-backed adapter. Tests
// swap this for the in-memory adapter via testhelpers.NewProductMemory.
func provideProductRepository(db *gorm.DB) product.Repository {
	return repositories.NewProduct(db)
}

// provideServer builds the *server.Server, which constructs the
// engine, mounts middleware + Swagger, and registers every route.
func provideServer(cfg server.ServerConfig, c *controllers.Controllers) *server.Server {
	return server.NewServer(cfg, c)
}
