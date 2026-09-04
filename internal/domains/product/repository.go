package product

import (
	"context"
	"errors"

	"github.com/Touutae-labs/simple-gin/internal/models"
)

// ErrNotFound is returned by a Repository when a lookup misses. The
// service layer maps this to models.CodeProductNotFound.
var ErrNotFound = errors.New("product not found")

// Repository is the storage port. Adapters (GORM, in-memory) implement
// it. The consumer (this package) declares the port.
type Repository interface {
	Create(ctx context.Context, in models.CreateInput) (string, error)
	GetByID(ctx context.Context, id string) (*models.Product, error)
	List(ctx context.Context, f *models.ListFilter) (*models.ListPage, error)
	Patch(ctx context.Context, id string, in models.PatchInput) (*models.Product, error)
	// SoftDelete sets deleted_at = now() on the row. Idempotent:
	// deleting an already-deleted product is a no-op (returns nil).
	SoftDelete(ctx context.Context, id string) error
}
