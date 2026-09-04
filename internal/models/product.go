// Package models holds the type scaffolding shared across the product
// sub-domain. Pure shapes — no tags, no I/O.
package models

import (
	"time"

	"github.com/shopspring/decimal"
)

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
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time // soft delete: nil = live, non-nil = tombstoned
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

// ListFilter is the validated query for List. A zero value means
// "no filter", which behaves the same as before this struct existed
// (return everything in created_at order).
type ListFilter struct {
	// Cursor is the id of the last product from the previous page.
	// Zero value = first page. Pagination is key-set on id, not on
	// offset, so inserts during pagination don't shift results.
	Cursor string
	// Limit caps the page size. The service layer clamps to
	// MaxListLimit if the caller asks for more.
	Limit int
	// Name is an optional case-insensitive contains-match. Empty
	// value = no name filter.
	Name string
	// MinPrice and MaxPrice are inclusive bounds. nil pointer
	// means "no bound on this side". Both can be nil; both can be
	// set; one can be set.
	MinPrice *Money
	MaxPrice *Money
}

// ListPage is the structured result of List. Items is the current
// page; NextCursor is the id to pass back as Cursor to fetch the
// next page. NextCursor == "" means "no more pages".
type ListPage struct {
	Items      []Product
	NextCursor string
}

// Result wraps the outcome of a mutation. data1 = product id,
// data2 = product name.
type Result struct {
	ProductID string
	Name      string
}

// Error is a stable, machine-readable error code. Implements the
// standard `error` interface so it can be returned from any
// function that takes `error`, but the rich fields (Code, Field,
// Message) are the contract callers branch on.
type Error struct {
	Code    string
	Field   string
	Message string
}

// Error implements the error interface. The string form is just
// the code so logs don't get noisy.
func (e *Error) Error() string { return e.Code }

// Stable error codes. Controllers and tests branch on these.
//
//	INVALID_NAME         → 422
//	INVALID_DESCRIPTION  → 422
//	INVALID_PRICE        → 422
//	INVALID_PRICE_RANGE  → 422
//	INVALID_LIMIT        → 422
//	PRICE_TOO_LARGE      → 422
//	PRODUCT_NOT_FOUND    → 404
//	REPOSITORY_FAILURE   → 500
const (
	CodeInvalidName         = "INVALID_NAME"
	CodeInvalidDescription  = "INVALID_DESCRIPTION"
	CodeInvalidPrice        = "INVALID_PRICE"
	CodeInvalidPriceRange   = "INVALID_PRICE_RANGE"
	CodeInvalidLimit        = "INVALID_LIMIT"
	CodePriceTooLarge       = "PRICE_TOO_LARGE"
	CodeProductNotFound     = "PRODUCT_NOT_FOUND"
	CodeRepositoryFailure   = "REPOSITORY_FAILURE"
)
