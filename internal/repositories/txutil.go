package repositories

import (
	"context"

	"gorm.io/gorm"
)

// txContextKey is the unexported key under which a GORM transaction
// is stashed in context. Repositories use FromContext to pick up
// the in-flight tx so port interfaces don't need to know about
// *gorm.DB.
type txContextKey struct{}

// WithTx injects a GORM transaction into ctx.
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// FromContext extracts a GORM transaction from ctx, or returns the
// fallback db if no transaction is present.
func FromContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txContextKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return db
}
