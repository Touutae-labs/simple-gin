// Package models holds the type scaffolding shared across the product
// sub-domain. Pure shapes — no tags, no I/O.
package models

import "github.com/shopspring/decimal"

// Money is a named type over shopspring/decimal so the domain talks
// about money in its own words.
type Money = decimal.Decimal

// Product is the canonical product in the domain. The HTTP-shaped
// request DTOs live in internal/controllers/; the GORM row lives in
// internal/repositories/models.go.
type Product struct {
	ID          string
	Name        string
	Description *string
	SalePrice   *Money
	Price       Money
}

// CreateInput is the validated input for creating a product. The
// controller is responsible for turning the HTTP body into this struct.
type CreateInput struct {
	Name        string
	Description *string
	SalePrice   *Money
	Price       Money
}

// PatchInput is a partial update. nil pointer = field not supplied
// (leave untouched). *string pointing to "" is "explicitly clear".
type PatchInput struct {
	Name        *string
	Description *string
	SalePrice   *Money
	Price       *Money
}

// Result wraps the outcome of a mutation. data1 = product id,
// data2 = product name.
type Result struct {
	ProductID string
	Name      string
}

// Error is a stable, machine-readable error code.
type Error struct {
	Code    string
	Field   string
	Message string
}

// Stable error codes. Controllers and tests branch on these.
//
//	INVALID_NAME        → 422
//	INVALID_PRICE       → 422
//	PRICE_TOO_LARGE     → 422
//	PRODUCT_NOT_FOUND   → 404
//	REPOSITORY_FAILURE  → 500
const (
	CodeInvalidName       = "INVALID_NAME"
	CodeInvalidPrice      = "INVALID_PRICE"
	CodePriceTooLarge     = "PRICE_TOO_LARGE"
	CodeProductNotFound   = "PRODUCT_NOT_FOUND"
	CodeRepositoryFailure = "REPOSITORY_FAILURE"
)
