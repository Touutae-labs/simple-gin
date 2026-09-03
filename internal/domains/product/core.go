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
const MaxNameLength = 255

// Module owns the business rules for products.
type Module interface {
	ValidateCreate(name string, price decimal.Decimal) *models.Error
	ValidatePatch(name *string, price *decimal.Decimal) *models.Error
}

type moduleImpl struct{}

func New() Module { return &moduleImpl{} }

func (m *moduleImpl) ValidateCreate(name string, price decimal.Decimal) *models.Error {
	if err := validateName(name); err != nil {
		return err
	}
	if err := validatePrice(price); err != nil {
		return err
	}
	return nil
}

func (m *moduleImpl) ValidatePatch(name *string, price *decimal.Decimal) *models.Error {
	if name != nil {
		if err := validateName(*name); err != nil {
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
