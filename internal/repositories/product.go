// Package repositories holds the GORM/Postgres production adapters
// for every sub-domain. In-memory test adapters live in
// internal/testhelpers.
package repositories

import (
	"context"
	"errors"
	"fmt"
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
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repositories.Product: get: %w", err)
	}
	return toDomain(row), nil
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
		if *in.Description == "" {
			updates["description"] = nil
		} else {
			updates["description"] = *in.Description
		}
	}
	if in.SalePrice != nil {
		updates["sale_price"] = in.SalePrice
	}
	if in.Price != nil {
		updates["price"] = *in.Price
	}

	db := FromContext(ctx, r.db)
	res := db.WithContext(ctx).
		Model(&ProductModel{}).
		Where("id = ?", id).
		Updates(updates)
	if res.Error != nil {
		return nil, fmt.Errorf("repositories.Product: patch: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func toDomain(row ProductModel) *models.Product {
	return &models.Product{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		SalePrice:   row.SalePrice,
		Price:       row.Price,
	}
}
