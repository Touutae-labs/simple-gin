// Integration test for the GORM-backed product repository. Needs a
// real PostgreSQL: set TEST_DATABASE_URL or rely on the default.
// Skips cleanly if unreachable.
package repositories_test

import (
	"context"
	"os"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/models"
	"github.com/Touutae-labs/simple-gin/internal/repositories"
)

func defaultDSN() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return "host=localhost user=postgres password=postgres dbname=simple_gin_test port=5432 sslmode=disable"
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{
		DriverName: "pgx",
		DSN:        defaultDSN(),
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("skipping integration test: cannot connect to Postgres: %v", err)
	}
	if err := db.AutoMigrate(repositories.AllModels()...); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	require.NoError(t, db.Exec("TRUNCATE products").Error)
	return db
}

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestProduct_CreateGet(t *testing.T) {
	db := newTestDB(t)
	repo := repositories.NewProduct(db)

	desc := "Single-origin roast"
	id, err := repo.Create(context.Background(), models.CreateInput{
		Name:        "Espresso",
		Description: &desc,
		SalePrice:   nil,
		Price:       d("29.90"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, id)

	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "Espresso", got.Name)
	require.Equal(t, "Single-origin roast", *got.Description)
	require.True(t, got.Price.Equal(d("29.90")))
}

func TestProduct_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := repositories.NewProduct(db)
	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestProduct_Patch_Partial(t *testing.T) {
	db := newTestDB(t)
	repo := repositories.NewProduct(db)

	desc := "old"
	id, err := repo.Create(context.Background(), models.CreateInput{
		Name: "Mug", Description: &desc, Price: d("9.99"),
	})
	require.NoError(t, err)

	got, err := repo.Patch(context.Background(), id, models.PatchInput{
		Price: decimalPtr(d("12.50")),
	})
	require.NoError(t, err)
	require.Equal(t, "Mug", got.Name)
	require.Equal(t, "old", *got.Description)
	require.True(t, got.Price.Equal(d("12.50")))
}

func TestProduct_Patch_NullDescription(t *testing.T) {
	db := newTestDB(t)
	repo := repositories.NewProduct(db)

	desc := "to be cleared"
	id, err := repo.Create(context.Background(), models.CreateInput{
		Name: "Pen", Description: &desc, Price: d("1.50"),
	})
	require.NoError(t, err)

	empty := ""
	got, err := repo.Patch(context.Background(), id, models.PatchInput{
		Description: &empty,
	})
	require.NoError(t, err)
	require.Nil(t, got.Description)
}

func TestProduct_Patch_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := repositories.NewProduct(db)
	_, err := repo.Patch(context.Background(), "00000000-0000-0000-0000-000000000000", models.PatchInput{
		Price: decimalPtr(d("1")),
	})
	require.ErrorIs(t, err, repositories.ErrNotFound)
}

func decimalPtr(v decimal.Decimal) *decimal.Decimal { return &v }

// Reference product.ErrNotFound so the test file fails the build if
// the sentinel is renamed.
var _ = product.ErrNotFound
