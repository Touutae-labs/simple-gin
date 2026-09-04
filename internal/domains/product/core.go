// Package product is the core of the product sub-domain. It holds the
// rule (Module), the port (Repository), and the orchestrator
// (Service). Adapters live in internal/repositories/ (GORM) and
// internal/testhelpers/ (in-memory).
package product

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Touutae-labs/simple-gin/internal/models"
)

// MaxPrice is the sanity cap for a single product price.
var MaxPrice = decimal.RequireFromString("10000000")

// MaxNameLength bounds the name so we never persist a multi-megabyte string.
const MaxNameLength = 200

// MaxDescriptionLength bounds the description to keep query indexes happy.
const MaxDescriptionLength = 2000

// MaxListLimit caps the page size the service will return, even if the
// caller asks for more. Anything over this is a misuse, not a feature.
const MaxListLimit = 100

// DefaultListLimit is the page size used when the caller doesn't pass
// one. Chosen to match what a typical UI dashboard wants on first paint.
const DefaultListLimit = 20

// Module owns the business rules for products.
type Module interface {
	ValidateCreate(name string, description *string, price decimal.Decimal) *models.Error
	ValidatePatch(name *string, description *string, price *decimal.Decimal) *models.Error
	ValidateListFilter(f *models.ListFilter) *models.Error
}

type moduleImpl struct{}

func New() Module { return &moduleImpl{} }

func (m *moduleImpl) ValidateCreate(name string, description *string, price decimal.Decimal) *models.Error {
	if err := validateName(name); err != nil {
		return err
	}
	if err := validateDescription(description); err != nil {
		return err
	}
	if err := validatePrice(price); err != nil {
		return err
	}
	return nil
}

func (m *moduleImpl) ValidatePatch(name *string, description *string, price *decimal.Decimal) *models.Error {
	if name != nil {
		if err := validateName(*name); err != nil {
			return err
		}
	}
	if description != nil {
		if err := validateDescription(description); err != nil {
			return err
		}
	}
	if price != nil {
		if err := validatePrice(*price); err != nil {
			return err
		}
	}
	return nil
}

// ValidateListFilter clamps the page size to [1, MaxListLimit] and
// checks the price-range pair if both bounds are set. The cursor
// itself is a UUID, not validated here — a bad cursor just returns
// an empty page, which the controller surfaces as 200 with an empty
// Items slice.
func (m *moduleImpl) ValidateListFilter(f *models.ListFilter) *models.Error {
	if f == nil {
		return nil
	}
	if f.Limit < 0 {
		return &models.Error{
			Code:    models.CodeInvalidLimit,
			Field:   "limit",
			Message: "limit must be >= 0 (0 means default)",
		}
	}
	if f.Limit > MaxListLimit {
		return &models.Error{
			Code:    models.CodeInvalidLimit,
			Field:   "limit",
			Message: fmt.Sprintf("limit must be at most %d, got %d", MaxListLimit, f.Limit),
		}
	}
	if f.MinPrice != nil && f.MinPrice.IsNegative() {
		return &models.Error{
			Code:    models.CodeInvalidPriceRange,
			Field:   "min_price",
			Message: fmt.Sprintf("min_price must be >= 0, got %s", f.MinPrice.String()),
		}
	}
	if f.MaxPrice != nil && f.MaxPrice.IsNegative() {
		return &models.Error{
			Code:    models.CodeInvalidPriceRange,
			Field:   "max_price",
			Message: fmt.Sprintf("max_price must be >= 0, got %s", f.MaxPrice.String()),
		}
	}
	if f.MinPrice != nil && f.MaxPrice != nil && f.MinPrice.GreaterThan(*f.MaxPrice) {
		return &models.Error{
			Code:    models.CodeInvalidPriceRange,
			Field:   "min_price",
			Message: fmt.Sprintf("min_price %s must be <= max_price %s", f.MinPrice.String(), f.MaxPrice.String()),
		}
	}
	return nil
}

func validateName(name string) *models.Error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return &models.Error{
			Code:    models.CodeInvalidName,
			Field:   "name",
			Message: "name is required",
		}
	}
	if len(trimmed) > MaxNameLength {
		return &models.Error{
			Code:    models.CodeInvalidName,
			Field:   "name",
			Message: fmt.Sprintf("name must be at most %d characters, got %d", MaxNameLength, len(trimmed)),
		}
	}
	return nil
}

func validateDescription(d *string) *models.Error {
	if d == nil {
		return nil
	}
	if len(*d) > MaxDescriptionLength {
		return &models.Error{
			Code:    models.CodeInvalidDescription,
			Field:   "description",
			Message: fmt.Sprintf("description must be at most %d characters, got %d", MaxDescriptionLength, len(*d)),
		}
	}
	return nil
}

func validatePrice(price decimal.Decimal) *models.Error {
	if !price.IsPositive() {
		return &models.Error{
			Code:    models.CodeInvalidPrice,
			Field:   "price",
			Message: fmt.Sprintf("price must be positive, got %s", price.String()),
		}
	}
	if price.GreaterThan(MaxPrice) {
		return &models.Error{
			Code:    models.CodePriceTooLarge,
			Field:   "price",
			Message: fmt.Sprintf("price %s exceeds maximum %s", price.String(), MaxPrice.String()),
		}
	}
	return nil
}
