package product_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/models"
	"github.com/Touutae-labs/simple-gin/internal/testhelpers"
)

func newService(t *testing.T) (product.Service, *testhelpers.ProductMemory) {
	t.Helper()
	repo := testhelpers.NewProductMemory()
	return product.NewService(product.New(), repo), repo
}

func TestService_Create_HappyPath(t *testing.T) {
	s, _ := newService(t)
	got, err := s.Create(context.Background(), models.CreateInput{
		Name:  "Espresso",
		Price: d("29.90"),
	})
	require.Nil(t, err)
	require.NotEmpty(t, got.ProductID)
	require.Equal(t, "Espresso", got.Name)
}

func TestService_Create_InvalidName(t *testing.T) {
	s, _ := newService(t)
	_, err := s.Create(context.Background(), models.CreateInput{
		Name:  "",
		Price: d("10"),
	})
	require.NotNil(t, err)
	require.Equal(t, models.CodeInvalidName, err.Code)
}

func TestService_Create_InvalidPrice(t *testing.T) {
	s, _ := newService(t)
	_, err := s.Create(context.Background(), models.CreateInput{
		Name:  "x",
		Price: d("-1"),
	})
	require.NotNil(t, err)
	require.Equal(t, models.CodeInvalidPrice, err.Code)
}

func TestService_Create_RepoError(t *testing.T) {
	s, repo := newService(t)
	repo.WithError(errors.New("db down"))
	_, err := s.Create(context.Background(), models.CreateInput{
		Name:  "x",
		Price: d("10"),
	})
	require.NotNil(t, err)
	require.Equal(t, models.CodeRepositoryFailure, err.Code)
}

func TestService_Patch_HappyPath(t *testing.T) {
	s, _ := newService(t)
	created, _ := s.Create(context.Background(), models.CreateInput{
		Name: "Mug", Price: d("9.99"),
	})
	newPrice := d("12.50")
	got, err := s.Patch(context.Background(), created.ProductID, models.PatchInput{
		Price: &newPrice,
	})
	require.Nil(t, err)
	require.Equal(t, created.ProductID, got.ProductID)
	require.Equal(t, "Mug", got.Name)
}

func TestService_Patch_NotFound(t *testing.T) {
	s, _ := newService(t)
	price := d("1")
	_, err := s.Patch(context.Background(), "does-not-exist", models.PatchInput{
		Price: &price,
	})
	require.NotNil(t, err)
	require.Equal(t, models.CodeProductNotFound, err.Code)
}

func TestService_Patch_InvalidPrice(t *testing.T) {
	s, _ := newService(t)
	created, _ := s.Create(context.Background(), models.CreateInput{
		Name: "x", Price: d("1"),
	})
	bad := d("-1")
	_, err := s.Patch(context.Background(), created.ProductID, models.PatchInput{
		Price: &bad,
	})
	require.NotNil(t, err)
	require.Equal(t, models.CodeInvalidPrice, err.Code)
}
