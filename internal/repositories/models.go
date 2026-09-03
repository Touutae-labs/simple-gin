package repositories

import (
	"time"

	"github.com/shopspring/decimal"
)

// ProductModel is the GORM row. Decimal columns are stored as TEXT
// so shopspring/decimal can round-trip without float loss.
type ProductModel struct {
	ID          string           `gorm:"primaryKey;column:id;type:uuid"`
	Name        string           `gorm:"column:name;type:text;not null"`
	Description *string          `gorm:"column:description;type:text"`
	SalePrice   *decimal.Decimal `gorm:"column:sale_price;type:numeric(20,2)"`
	Price       decimal.Decimal  `gorm:"column:price;type:numeric(20,2);not null"`
	CreatedAt   time.Time        `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time        `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (ProductModel) TableName() string { return "products" }

// AllModels returns every GORM model in the codebase. Used by
// AutoMigrate.
func AllModels() []any {
	return []any{&ProductModel{}}
}
