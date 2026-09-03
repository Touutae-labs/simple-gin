package repositories

import "github.com/Touutae-labs/simple-gin/internal/domains/product"

// ErrNotFound is re-exported from product.ErrNotFound so callers
// can do errors.Is(err, repositories.ErrNotFound) without pulling
// the domain package.
var ErrNotFound = product.ErrNotFound
