// Package repositories holds the GORM/Postgres production adapters
// for every sub-domain. In-memory test adapters live in
// internal/testhelpers.
package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/models"
)

var _ product.Repository = (*Product)(nil)

// Product is the GORM-backed implementation of product.Repository.
type Product struct {
	db *gorm.DB
}


func NewProduct(db *gorm.DB) *Product {
	return &Product{db: db}
}


func (r *Product) Create(ctx context.Context, in models.CreateInput) (string, error) {
	id := uuid.NewString()
	now := time.Now().UTC()
	row := ProductModel{
		ID:          id,
		Name:        in.Name,
		Description: in.Description,
		SalePrice:   in.SalePrice,
		Price:       in.Price,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return "", fmt.Errorf("repositories.Product: create: %w", err)
	}

	return id, nil
}


func (r *Product) GetByID(ctx context.Context, id string) (*models.Product, error) {
	var row ProductModel
	// Soft-deleted rows are invisible to GetByID — List and SoftDelete
	// see the same world.
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, product.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("repositories.Product: get: %w", err)
	}

	return toDomain(&row), nil
}


// List returns one page of products ordered by id (UUID sort is
// stable across pages — same as created_at order for our inserts,
// but id is the right key for cursor pagination). Soft-deleted
// rows are excluded. Filter fields are optional; nil/zero means
// "no constraint on that dimension".
func (r *Product) List(ctx context.Context, f *models.ListFilter) (*models.ListPage, error) {
	if f == nil {
		f = &models.ListFilter{}
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}

	// Fetch limit+1 so we can tell whether there's a next page
	// without a second COUNT query.
	q := r.db.WithContext(ctx).Model(&ProductModel{}).Order("id asc").Limit(limit + 1)
	if f.Cursor != "" {
		q = q.Where("id > ?", f.Cursor)
	}

	if name := strings.TrimSpace(f.Name); name != "" {
		q = q.Where("name ILIKE ?", "%"+name+"%")
	}

	if f.MinPrice != nil {
		q = q.Where("price >= ?", f.MinPrice.String())
	}

	if f.MaxPrice != nil {
		q = q.Where("price <= ?", f.MaxPrice.String())
	}

	var rows []ProductModel
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("repositories.Product: list: %w", err)
	}

	page := &models.ListPage{Items: make([]models.Product, 0, len(rows))}
	for i := range rows {
		page.Items = append(page.Items, *toDomain(&rows[i]))
	}

	if len(page.Items) > limit {
		// Drop the sentinel and use the last kept row's id as the
		// cursor for the next page.
		page.NextCursor = page.Items[limit-1].ID
		page.Items = page.Items[:limit]
	}

	return page, nil
}


func (r *Product) Patch(ctx context.Context, id string, in models.PatchInput) (*models.Product, error) {
	updates := map[string]any{
		"updated_at": time.Now().UTC(),
	}

	if in.Name != nil {
		updates["name"] = *in.Name
	}

	// *string on PatchInput carries "explicit clear" semantics:
	// pointer to "" means "set this column to NULL".
	if in.Description != nil {
		updates["description"] = nilIfEmpty(*in.Description)
	}

	if in.SalePrice != nil {
		updates["sale_price"] = in.SalePrice
	}

	if in.Price != nil {
		updates["price"] = *in.Price
	}


	res := r.db.WithContext(ctx).
		Model(&ProductModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)
	if res.Error != nil {
		return nil, fmt.Errorf("repositories.Product: patch: %w", res.Error)
	}

	if res.RowsAffected == 0 {
		return nil, product.ErrNotFound
	}

	return r.GetByID(ctx, id)
}


// SoftDelete sets deleted_at = now() on the row. The default GORM
// scope for the model doesn't include soft-delete (we don't import
// gorm.io/plugin/soft_delete), so we manage the column explicitly.
// Idempotent: if the row is already soft-deleted, this is a no-op.
func (r *Product) SoftDelete(ctx context.Context, id string) error {
	now := time.Now().UTC()
	// Only flip the tombstone if it's currently NULL — keeps the
	// operation idempotent and avoids bumping updated_at on
	// already-deleted rows.
	res := r.db.WithContext(ctx).
		Model(&ProductModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{"deleted_at": now, "updated_at": now})
	if res.Error != nil {
		return fmt.Errorf("repositories.Product: soft_delete: %w", res.Error)
	}

	if res.RowsAffected == 1 {
		return nil
	}

	// Not found = either no such id, or already deleted. Distinguish
	// via a follow-up existence check so a real miss returns
	// product.ErrNotFound while an already-deleted row is a no-op.
	var row ProductModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return product.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("repositories.Product: soft_delete.exists_check: %w", err)
	}

	return nil
}


func toDomain(row *ProductModel) *models.Product {
	return &models.Product{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		SalePrice:   row.SalePrice,
		Price:       row.Price,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		DeletedAt:   row.DeletedAt,
	}
}


// nilIfEmpty returns nil for the empty string, the value otherwise.
// Used to encode PATCH semantics where *string pointing to "" means
// "explicitly clear this column" rather than "leave it alone".
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}

	return s
}
