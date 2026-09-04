package product_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/models"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func strPtr(s string) *string { return &s }

func TestModule_ValidateCreate(t *testing.T) {
	m := product.New()

	tests := []struct {
		name      string
		inputName string
		inputDesc *string
		price     string
		wantCode  string
	}{
		{"valid", "Espresso Beans", strPtr("good"), "29.90", ""},
		{"valid no desc", "Espresso", nil, "29.90", ""},
		{"empty name", "", nil, "10", models.CodeInvalidName},
		{"whitespace name", "   ", nil, "10", models.CodeInvalidName},
		{"name too long", string(make([]byte, product.MaxNameLength+1)), nil, "10", models.CodeInvalidName},
		{"description too long", "x", strPtr(string(make([]byte, product.MaxDescriptionLength+1))), "10", models.CodeInvalidDescription},
		{"zero price", "x", nil, "0", models.CodeInvalidPrice},
		{"negative price", "x", nil, "-1", models.CodeInvalidPrice},
		{"price over cap", "x", nil, "10000001", models.CodePriceTooLarge},
		{"price at cap", "x", nil, product.MaxPrice.String(), ""},
	}


	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.ValidateCreate(tt.inputName, tt.inputDesc, d(tt.price))
			if tt.wantCode == "" {
				require.Nil(t, err)
				return
			}

			require.NotNil(t, err, "expected code %s, got nil", tt.wantCode)
			require.Equal(t, tt.wantCode, err.Code)
		})
	}
}


func TestModule_ValidatePatch(t *testing.T) {
	m := product.New()

	t.Run("nil fields pass through", func(t *testing.T) {
		require.Nil(t, m.ValidatePatch(nil, nil, nil))
	})

	t.Run("invalid name rejected", func(t *testing.T) {
		bad := ""
		err := m.ValidatePatch(&bad, nil, nil)
		require.NotNil(t, err)
		require.Equal(t, models.CodeInvalidName, err.Code)
	})

	t.Run("invalid description rejected", func(t *testing.T) {
		bad := string(make([]byte, product.MaxDescriptionLength+1))
		err := m.ValidatePatch(nil, &bad, nil)
		require.NotNil(t, err)
		require.Equal(t, models.CodeInvalidDescription, err.Code)
	})

	t.Run("invalid price rejected", func(t *testing.T) {
		bad := d("-1")
		err := m.ValidatePatch(nil, nil, &bad)
		require.NotNil(t, err)
		require.Equal(t, models.CodeInvalidPrice, err.Code)
	})

	t.Run("valid name and price accepted", func(t *testing.T) {
		good := "Updated"
		price := d("19.99")
		require.Nil(t, m.ValidatePatch(&good, nil, &price))
	})
}


func TestModule_ValidateListFilter(t *testing.T) {
	m := product.New()

	t.Run("nil filter is ok", func(t *testing.T) {
		require.Nil(t, m.ValidateListFilter(nil))
	})

	t.Run("zero filter is ok", func(t *testing.T) {
		require.Nil(t, m.ValidateListFilter(&models.ListFilter{}))
	})

	t.Run("negative limit rejected", func(t *testing.T) {
		err := m.ValidateListFilter(&models.ListFilter{Limit: -1})
		require.NotNil(t, err)
		require.Equal(t, models.CodeInvalidLimit, err.Code)
	})

	t.Run("limit over cap rejected", func(t *testing.T) {
		err := m.ValidateListFilter(&models.ListFilter{Limit: product.MaxListLimit + 1})
		require.NotNil(t, err)
		require.Equal(t, models.CodeInvalidLimit, err.Code)
	})

	t.Run("min > max rejected", func(t *testing.T) {
		min, max := d("20"), d("10")
		err := m.ValidateListFilter(&models.ListFilter{MinPrice: &min, MaxPrice: &max})
		require.NotNil(t, err)
		require.Equal(t, models.CodeInvalidPriceRange, err.Code)
	})

	t.Run("min == max accepted", func(t *testing.T) {
		min, max := d("10"), d("10")
		require.Nil(t, m.ValidateListFilter(&models.ListFilter{MinPrice: &min, MaxPrice: &max}))
	})

	t.Run("valid range accepted", func(t *testing.T) {
		min, max := d("10"), d("100")
		require.Nil(t, m.ValidateListFilter(&models.ListFilter{MinPrice: &min, MaxPrice: &max}))
	})
}
