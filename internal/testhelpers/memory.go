// Package testhelpers holds non-production adapters (currently the
// in-memory product.Repository) used only by tests. Production
// adapters live in internal/repositories.
package testhelpers

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/Touutae-labs/simple-gin/internal/domains/product"
	"github.com/Touutae-labs/simple-gin/internal/models"
)

// ProductMemory is a thread-safe, in-memory implementation of
// product.Repository.
type ProductMemory struct {
	mu       sync.RWMutex
	products map[string]models.Product
	forceErr error
}

func NewProductMemory() *ProductMemory {
	return &ProductMemory{products: map[string]models.Product{}}
}

// WithError makes every subsequent call return err.
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
	m.products[id] = models.Product{
		ID:          id,
		Name:        in.Name,
		Description: in.Description,
		SalePrice:   in.SalePrice,
		Price:       in.Price,
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
	if !ok {
		return nil, product.ErrNotFound
	}
	cp := p
	return &cp, nil
}

func (m *ProductMemory) List(_ context.Context) ([]models.Product, error) {
	if err := m.fail(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]models.Product, 0, len(m.products))
	for _, p := range m.products {
		out = append(out, p)
	}
	return out, nil
}

func (m *ProductMemory) Patch(_ context.Context, id string, in models.PatchInput) (*models.Product, error) {
	if err := m.fail(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.products[id]
	if !ok {
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
	m.products[id] = p
	cp := p
	return &cp, nil
}

func (m *ProductMemory) fail() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.forceErr
}
