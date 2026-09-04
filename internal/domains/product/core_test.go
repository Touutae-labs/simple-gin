package product_test

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/models"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func strPtr(s string) *string { return &s }

func TestModule_ValidateCreate(t *testing.T) {
	m := product.New()

	tests := []struct {
		name        string
		inputName   string
		inputDesc   *string
		price       string
		wantCode    string
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
				if err != nil {
					t.Errorf("expected valid, got %+v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected code %s, got nil", tt.wantCode)
			}
			if err.Code != tt.wantCode {
				t.Errorf("got code %s, want %s", err.Code, tt.wantCode)
			}
		})
	}
}

func TestModule_ValidatePatch(t *testing.T) {
	m := product.New()

	t.Run("nil fields pass through", func(t *testing.T) {
		if err := m.ValidatePatch(nil, nil, nil); err != nil {
			t.Errorf("nil/nil should pass, got %+v", err)
		}
	})

	t.Run("invalid name rejected", func(t *testing.T) {
		bad := ""
		err := m.ValidatePatch(&bad, nil, nil)
		if err == nil || err.Code != models.CodeInvalidName {
			t.Errorf("expected INVALID_NAME, got %+v", err)
		}
	})

	t.Run("invalid description rejected", func(t *testing.T) {
		bad := string(make([]byte, product.MaxDescriptionLength+1))
		err := m.ValidatePatch(nil, &bad, nil)
		if err == nil || err.Code != models.CodeInvalidDescription {
			t.Errorf("expected INVALID_DESCRIPTION, got %+v", err)
		}
	})

	t.Run("invalid price rejected", func(t *testing.T) {
		bad := d("-1")
		err := m.ValidatePatch(nil, nil, &bad)
		if err == nil || err.Code != models.CodeInvalidPrice {
			t.Errorf("expected INVALID_PRICE, got %+v", err)
		}
	})

	t.Run("valid name and price accepted", func(t *testing.T) {
		good := "Updated"
		price := d("19.99")
		if err := m.ValidatePatch(&good, nil, &price); err != nil {
			t.Errorf("expected valid, got %+v", err)
		}
	})
}

func TestModule_ValidateListFilter(t *testing.T) {
	m := product.New()

	t.Run("nil filter is ok", func(t *testing.T) {
		if err := m.ValidateListFilter(nil); err != nil {
			t.Errorf("expected nil, got %+v", err)
		}
	})

	t.Run("zero filter is ok", func(t *testing.T) {
		if err := m.ValidateListFilter(&models.ListFilter{}); err != nil {
			t.Errorf("expected nil, got %+v", err)
		}
	})

	t.Run("negative limit rejected", func(t *testing.T) {
		if err := m.ValidateListFilter(&models.ListFilter{Limit: -1}); err == nil || err.Code != models.CodeInvalidLimit {
			t.Errorf("expected INVALID_LIMIT, got %+v", err)
		}
	})

	t.Run("limit over cap rejected", func(t *testing.T) {
		if err := m.ValidateListFilter(&models.ListFilter{Limit: product.MaxListLimit + 1}); err == nil || err.Code != models.CodeInvalidLimit {
			t.Errorf("expected INVALID_LIMIT, got %+v", err)
		}
	})

	t.Run("min > max rejected", func(t *testing.T) {
		min, max := d("20"), d("10")
		if err := m.ValidateListFilter(&models.ListFilter{MinPrice: &min, MaxPrice: &max}); err == nil || err.Code != models.CodeInvalidPriceRange {
			t.Errorf("expected INVALID_PRICE_RANGE, got %+v", err)
		}
	})

	t.Run("min == max accepted", func(t *testing.T) {
		min, max := d("10"), d("10")
		if err := m.ValidateListFilter(&models.ListFilter{MinPrice: &min, MaxPrice: &max}); err != nil {
			t.Errorf("min==max should pass, got %+v", err)
		}
	})

	t.Run("valid range accepted", func(t *testing.T) {
		min, max := d("10"), d("100")
		if err := m.ValidateListFilter(&models.ListFilter{MinPrice: &min, MaxPrice: &max}); err != nil {
			t.Errorf("valid range should pass, got %+v", err)
		}
	})
}
