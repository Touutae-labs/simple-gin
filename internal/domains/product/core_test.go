package product_test

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/models"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestModule_ValidateCreate(t *testing.T) {
	m := product.New()

	tests := []struct {
		name     string
		input    string
		price    string
		wantCode string
	}{
		{"valid", "Espresso Beans", "29.90", ""},
		{"empty name", "", "10", models.CodeInvalidName},
		{"whitespace name", "   ", "10", models.CodeInvalidName},
		{"name too long", string(make([]byte, product.MaxNameLength+1)), "10", models.CodeInvalidName},
		{"zero price", "x", "0", models.CodeInvalidPrice},
		{"negative price", "x", "-1", models.CodeInvalidPrice},
		{"price over cap", "x", "10000001", models.CodePriceTooLarge},
		{"price at cap", "x", product.MaxPrice.String(), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.ValidateCreate(tt.input, d(tt.price))
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
		if err := m.ValidatePatch(nil, nil); err != nil {
			t.Errorf("nil/nil should pass, got %+v", err)
		}
	})

	t.Run("invalid name rejected", func(t *testing.T) {
		bad := ""
		err := m.ValidatePatch(&bad, nil)
		if err == nil || err.Code != models.CodeInvalidName {
			t.Errorf("expected INVALID_NAME, got %+v", err)
		}
	})

	t.Run("invalid price rejected", func(t *testing.T) {
		bad := d("-1")
		err := m.ValidatePatch(nil, &bad)
		if err == nil || err.Code != models.CodeInvalidPrice {
			t.Errorf("expected INVALID_PRICE, got %+v", err)
		}
	})

	t.Run("valid name and price accepted", func(t *testing.T) {
		good := "Updated"
		price := d("19.99")
		if err := m.ValidatePatch(&good, &price); err != nil {
			t.Errorf("expected valid, got %+v", err)
		}
	})
}
