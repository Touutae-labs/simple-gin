package di

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Touutae-labs/simple-gin/internal/configurations"
	"github.com/Touutae-labs/simple-gin/internal/controllers"
	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/repositories"
	"github.com/Touutae-labs/simple-gin/internal/server"
	"github.com/Touutae-labs/simple-gin/internal/testhelpers"
)

// mockDB is read once at startup. APP_MOCK_DB=1 short-circuits the
// Postgres connection and wires the in-memory product repository
// instead. Use this for local smoke testing when no DB is available.
// The Application.DB field will be nil in mock mode.
func mockDB() bool { return os.Getenv("APP_MOCK_DB") == "1" }

// provideDB opens the Postgres connection via pgx's database/sql
// driver and GORM's postgres driver. AutoMigrate runs when
// cfg.AutoMigrate is true. In mock mode (APP_MOCK_DB=1) it returns
// a nil *gorm.DB with a no-op cleanup so the rest of the wire graph
// resolves unchanged.
func provideDB(cfg configurations.DatabaseConfig) (*gorm.DB, func(), error) {
	if mockDB() {
		return nil, func() {}, nil
	}

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


// provideProductRepository returns the GORM-backed adapter. When the
// db is nil (mock mode), it returns the in-memory adapter from
// testhelpers instead. Both adapters satisfy product.Repository.
func provideProductRepository(db *gorm.DB) product.Repository {
	if db == nil {
		return testhelpers.NewProductMemory()
	}

	return repositories.NewProduct(db)
}


// provideServer builds the *server.Server, which constructs the
// engine, mounts middleware + Swagger, and registers every route.
func provideServer(cfg server.ServerConfig, c *controllers.Controllers) *server.Server {
	return server.NewServer(cfg, c)
}
