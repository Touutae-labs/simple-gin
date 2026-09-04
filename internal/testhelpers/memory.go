// Package testhelpers holds non-production adapters (currently the
// in-memory product.Repository) used only by tests. Production
// adapters live in internal/repositories.
package testhelpers

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/models"
)

// ProductMemory is a thread-safe, in-memory implementation of
// product.Repository. The map is keyed on id; ordering for List
// is by id (string sort) so pagination is stable across pages.
type ProductMemory struct {
	mu       sync.RWMutex
	products map[string]models.Product
	forceErr error
}

func NewProductMemory() *ProductMemory {
	return &ProductMemory{products: map[string]models.Product{}}
}

// WithError makes every subsequent call return err. Used to exercise
// the repository-failure error path in the service.
func (m *ProductMemory) WithError(err error) *ProductMemory {
	m.mu.Lock()
	m.forceErr = err
	m.mu.Unlock()
	return m
}

var _ product.Repository = (*ProductMemory)(nil)

func (m *ProductMemory) Create(_ context.Context, in models.CreateInput) (string, error) {
	if err := m.fail(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.NewString()
	now := time.Now().UTC()
	m.products[id] = models.Product{
		ID:          id,
		Name:        in.Name,
		Description: in.Description,
		SalePrice:   in.SalePrice,
		Price:       in.Price,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return id, nil
}

func (m *ProductMemory) GetByID(_ context.Context, id string) (*models.Product, error) {
	if err := m.fail(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.products[id]
	if !ok || p.DeletedAt != nil {
		return nil, product.ErrNotFound
	}
	cp := p
	return &cp, nil
}

// List paginates over live (non-deleted) products ordered by id. The
// filter is applied in memory; tests use small N so the cost is fine.
func (m *ProductMemory) List(_ context.Context, f *models.ListFilter) (*models.ListPage, error) {
	if err := m.fail(); err != nil {
		return nil, err
	}
	if f == nil {
		f = &models.ListFilter{}
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect live ids in deterministic order.
	ids := make([]string, 0, len(m.products))
	for id, p := range m.products {
		if p.DeletedAt == nil {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	// Skip past the cursor.
	start := 0
	if f.Cursor != "" {
		for i, id := range ids {
			if id > f.Cursor {
				start = i
				break
			}
		}
	}

	// Fetch limit+1 to know if there's a next page.
	end := start + limit + 1
	if end > len(ids) {
		end = len(ids)
	}
	page := &models.ListPage{Items: make([]models.Product, 0, limit)}
	for _, id := range ids[start:end] {
		page.Items = append(page.Items, m.products[id])
	}
	if len(page.Items) > limit {
		page.NextCursor = page.Items[limit-1].ID
		page.Items = page.Items[:limit]
	}

	// Apply name + price filters (after pagination since the in-memory
	// store is small). The trade-off: page size becomes "approximate
	// up to limit" when filters are set. Acceptable for tests.
	if name := strings.TrimSpace(f.Name); name != "" {
		filtered := page.Items[:0]
		for _, p := range page.Items {
			if strings.Contains(strings.ToLower(p.Name), strings.ToLower(name)) {
				filtered = append(filtered, p)
			}
		}
		page.Items = filtered
	}
	if f.MinPrice != nil || f.MaxPrice != nil {
		filtered := page.Items[:0]
		for _, p := range page.Items {
			if f.MinPrice != nil && p.Price.LessThan(*f.MinPrice) {
				continue
			}
			if f.MaxPrice != nil && p.Price.GreaterThan(*f.MaxPrice) {
				continue
			}
			filtered = append(filtered, p)
		}
		page.Items = filtered
	}

	return page, nil
}

func (m *ProductMemory) Patch(_ context.Context, id string, in models.PatchInput) (*models.Product, error) {
	if err := m.fail(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.products[id]
	if !ok || p.DeletedAt != nil {
		return nil, product.ErrNotFound
	}
	if in.Name != nil {
		p.Name = *in.Name
	}
	if in.Description != nil {
		p.Description = in.Description
	}
	if in.SalePrice != nil {
		p.SalePrice = in.SalePrice
	}
	if in.Price != nil {
		p.Price = *in.Price
	}
	p.UpdatedAt = time.Now().UTC()
	m.products[id] = p
	cp := p
	return &cp, nil
}

// SoftDelete sets DeletedAt to now on a live row. Idempotent — if
// the row is already deleted, returns nil and leaves the timestamp
// alone.
func (m *ProductMemory) SoftDelete(_ context.Context, id string) error {
	if err := m.fail(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.products[id]
	if !ok {
		return product.ErrNotFound
	}
	if p.DeletedAt != nil {
		return nil
	}
	now := time.Now().UTC()
	p.DeletedAt = &now
	p.UpdatedAt = now
	m.products[id] = p
	return nil
}

func (m *ProductMemory) fail() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.forceErr
}
