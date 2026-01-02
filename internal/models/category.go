package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CategoryType represents the type of category
type CategoryType string

const (
	CategoryTypeTicket         CategoryType = "ticket"          // Categorias de chamados
	CategoryTypeFinanceIncome  CategoryType = "finance_income"  // Categorias de receitas
	CategoryTypeFinanceExpense CategoryType = "finance_expense" // Categorias de despesas
)

type Category struct {
	ID          string         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string         `json:"name" gorm:"not null" validate:"required"`
	Description string         `json:"description"`
	Color       string         `json:"color"`
	Icon        string         `json:"icon"`
	Type        CategoryType   `json:"type" gorm:"type:varchar(20);not null;default:ticket;index"`
	ParentID    *string        `json:"parentId" gorm:"type:uuid;index"`
	Parent      *Category      `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children    []Category     `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	Active      bool           `json:"active" gorm:"default:true"`
	SortOrder   int            `json:"sortOrder" gorm:"default:0"`
	Timestamp   time.Time      `json:"timestamp" gorm:"autoCreateTime"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.Type == "" {
		c.Type = CategoryTypeTicket
	}
	return nil
}
