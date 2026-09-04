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

func TestService_Create_InvalidDescription(t *testing.T) {
	s, _ := newService(t)
	huge := string(make([]byte, product.MaxDescriptionLength+1))
	_, err := s.Create(context.Background(), models.CreateInput{
		Name:        "x",
		Description: &huge,
		Price:       d("10"),
	})
	require.NotNil(t, err)
	require.Equal(t, models.CodeInvalidDescription, err.Code)
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

func TestService_Delete_HappyPath(t *testing.T) {
	s, _ := newService(t)
	created, _ := s.Create(context.Background(), models.CreateInput{
		Name: "Mug", Price: d("9.99"),
	})
	require.Nil(t, s.Delete(context.Background(), created.ProductID))

	// After delete, Get should return PRODUCT_NOT_FOUND.
	_, err := s.Get(context.Background(), created.ProductID)
	require.NotNil(t, err)
	require.Equal(t, models.CodeProductNotFound, err.Code)
}

func TestService_Delete_NotFound(t *testing.T) {
	s, _ := newService(t)
	err := s.Delete(context.Background(), "does-not-exist")
	require.NotNil(t, err)
	require.Equal(t, models.CodeProductNotFound, err.Code)
}

func TestService_Delete_Idempotent(t *testing.T) {
	s, _ := newService(t)
	created, _ := s.Create(context.Background(), models.CreateInput{
		Name: "Mug", Price: d("9.99"),
	})
	require.Nil(t, s.Delete(context.Background(), created.ProductID))
	// Second delete is a no-op, not an error.
	require.Nil(t, s.Delete(context.Background(), created.ProductID))
}

func TestService_List_NoFilter(t *testing.T) {
	s, _ := newService(t)
	_, _ = s.Create(context.Background(), models.CreateInput{Name: "A", Price: d("1")})
	_, _ = s.Create(context.Background(), models.CreateInput{Name: "B", Price: d("2")})
	page, err := s.List(context.Background(), nil)
	require.Nil(t, err)
	require.Len(t, page.Items, 2)
}

func TestService_List_FilterByName(t *testing.T) {
	s, _ := newService(t)
	_, _ = s.Create(context.Background(), models.CreateInput{Name: "Espresso Beans", Price: d("10")})
	_, _ = s.Create(context.Background(), models.CreateInput{Name: "Filter Basket", Price: d("20")})
	page, err := s.List(context.Background(), &models.ListFilter{Name: "basket"})
	require.Nil(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "Filter Basket", page.Items[0].Name)
}

func TestService_List_FilterByPriceRange(t *testing.T) {
	s, _ := newService(t)
	_, _ = s.Create(context.Background(), models.CreateInput{Name: "Cheap", Price: d("5")})
	_, _ = s.Create(context.Background(), models.CreateInput{Name: "Mid", Price: d("50")})
	_, _ = s.Create(context.Background(), models.CreateInput{Name: "Pricey", Price: d("500")})
	min, max := d("10"), d("100")
	page, err := s.List(context.Background(), &models.ListFilter{MinPrice: &min, MaxPrice: &max})
	require.Nil(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "Mid", page.Items[0].Name)
}

func TestService_List_Pagination(t *testing.T) {
	s, _ := newService(t)
	// Insert 5 products, list with limit=2, expect next_cursor set.
	for _, n := range []string{"A", "B", "C", "D", "E"} {
		_, _ = s.Create(context.Background(), models.CreateInput{Name: n, Price: d("1")})
	}
	page, err := s.List(context.Background(), &models.ListFilter{Limit: 2})
	require.Nil(t, err)
	require.Len(t, page.Items, 2)
	require.NotEmpty(t, page.NextCursor)

	// Fetch the next page using the cursor.
	page2, err := s.List(context.Background(), &models.ListFilter{Limit: 2, Cursor: page.NextCursor})
	require.Nil(t, err)
	require.Len(t, page2.Items, 2)
	require.NotEmpty(t, page2.NextCursor)
}

func TestService_List_ExcludesSoftDeleted(t *testing.T) {
	s, _ := newService(t)
	a, _ := s.Create(context.Background(), models.CreateInput{Name: "Keep", Price: d("1")})
	b, _ := s.Create(context.Background(), models.CreateInput{Name: "Delete", Price: d("1")})
	require.Nil(t, s.Delete(context.Background(), b.ProductID))

	page, err := s.List(context.Background(), nil)
	require.Nil(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, a.ProductID, page.Items[0].ID)
}

func TestService_List_InvalidLimit(t *testing.T) {
	s, _ := newService(t)
	_, err := s.List(context.Background(), &models.ListFilter{Limit: -1})
	require.NotNil(t, err)
	require.Equal(t, models.CodeInvalidLimit, err.Code)
}

func TestService_List_InvalidPriceRange(t *testing.T) {
	s, _ := newService(t)
	min, max := d("100"), d("10")
	_, err := s.List(context.Background(), &models.ListFilter{MinPrice: &min, MaxPrice: &max})
	require.NotNil(t, err)
	require.Equal(t, models.CodeInvalidPriceRange, err.Code)
}
