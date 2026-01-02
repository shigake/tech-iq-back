package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// =============== Enums ===============

type StockLocationType string

const (
	LocationWarehouse  StockLocationType = "WAREHOUSE"
	LocationBranch     StockLocationType = "BRANCH"
	LocationTechnician StockLocationType = "TECHNICIAN"
	LocationClient     StockLocationType = "CLIENT"
)

func (t StockLocationType) IsValid() bool {
	switch t {
	case LocationWarehouse, LocationBranch, LocationTechnician, LocationClient:
		return true
	}
	return false
}

type StockMovementType string

const (
	MovementTypeEntradaCompra    StockMovementType = "ENTRADA_COMPRA"
	MovementTypeEntradaDevolucao StockMovementType = "ENTRADA_DEVOLUCAO"
	MovementTypeTransferencia    StockMovementType = "TRANSFERENCIA"
	MovementTypeSaidaConsumoOS   StockMovementType = "SAIDA_CONSUMO_OS"
	MovementTypeSaidaPerda       StockMovementType = "SAIDA_PERDA"
	MovementTypeAjusteInventario StockMovementType = "AJUSTE_INVENTARIO"
)

func (t StockMovementType) IsValid() bool {
	switch t {
	case MovementTypeEntradaCompra, MovementTypeEntradaDevolucao, MovementTypeTransferencia,
		MovementTypeSaidaConsumoOS, MovementTypeSaidaPerda, MovementTypeAjusteInventario:
		return true
	}
	return false
}

func (t StockMovementType) IsEntry() bool {
	return t == MovementTypeEntradaCompra || t == MovementTypeEntradaDevolucao
}

func (t StockMovementType) IsExit() bool {
	return t == MovementTypeSaidaConsumoOS || t == MovementTypeSaidaPerda
}

func (t StockMovementType) IsTransfer() bool {
	return t == MovementTypeTransferencia
}

func (t StockMovementType) IsAdjustment() bool {
	return t == MovementTypeAjusteInventario
}

// =============== Models ===============

// StockItem represents an inventory item
type StockItem struct {
	ID          string    `json:"id" gorm:"type:uuid;primaryKey"`
	SKU         string    `json:"sku" gorm:"type:varchar(100);uniqueIndex;not null"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null"`
	Description *string   `json:"description" gorm:"type:text"`
	Category    *string   `json:"category" gorm:"type:varchar(100)"`
	Unit        string    `json:"unit" gorm:"type:varchar(20);not null;default:'UN'"`
	MinQty      int       `json:"minQty" gorm:"default:0"`
	TrackSerial bool      `json:"trackSerial" gorm:"default:false"`
	IsActive    bool      `json:"isActive" gorm:"default:true"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (s *StockItem) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

func (StockItem) TableName() string {
	return "stock_items"
}

// StockLocation represents a storage location
type StockLocation struct {
	ID        string            `json:"id" gorm:"type:uuid;primaryKey"`
	ScopeID   string            `json:"scopeId" gorm:"type:uuid;index;not null"`
	Type      StockLocationType `json:"type" gorm:"type:varchar(50);not null"`
	Name      string            `json:"name" gorm:"type:varchar(255);not null"`
	IsActive  bool              `json:"isActive" gorm:"default:true"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

func (s *StockLocation) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

func (StockLocation) TableName() string {
	return "stock_locations"
}

// StockMovement represents an immutable ledger entry
type StockMovement struct {
	ID             string            `json:"id" gorm:"type:uuid;primaryKey"`
	ScopeID        string            `json:"scopeId" gorm:"type:uuid;index;not null"`
	Type           StockMovementType `json:"type" gorm:"type:varchar(50);not null;index"`
	ItemID         string            `json:"itemId" gorm:"type:uuid;not null;index"`
	FromLocationID *string           `json:"fromLocationId" gorm:"type:uuid;index"`
	ToLocationID   *string           `json:"toLocationId" gorm:"type:uuid;index"`
	TicketID       *string           `json:"ticketId" gorm:"type:uuid;index"`
	Quantity       int               `json:"quantity" gorm:"not null"`
	UnitCost       *decimal.Decimal  `json:"unitCost" gorm:"type:decimal(12,2)"`
	Notes          *string           `json:"notes" gorm:"type:text"`
	PerformedBy    string            `json:"performedBy" gorm:"type:uuid;not null"`
	PerformedAt    time.Time         `json:"performedAt" gorm:"not null"`
	CreatedAt      time.Time         `json:"createdAt"`

	// Relations (for eager loading)
	Item         *StockItem     `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	FromLocation *StockLocation `json:"fromLocation,omitempty" gorm:"foreignKey:FromLocationID"`
	ToLocation   *StockLocation `json:"toLocation,omitempty" gorm:"foreignKey:ToLocationID"`
	Performer    *User          `json:"performer,omitempty" gorm:"foreignKey:PerformedBy"`
}

func (s *StockMovement) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.PerformedAt.IsZero() {
		s.PerformedAt = time.Now()
	}
	return nil
}

func (StockMovement) TableName() string {
	return "stock_movements"
}

// StockBalance represents the materialized balance per item + location
type StockBalance struct {
	ID         string    `json:"id" gorm:"type:uuid;primaryKey"`
	ScopeID    string    `json:"scopeId" gorm:"type:uuid;index;not null"`
	ItemID     string    `json:"itemId" gorm:"type:uuid;not null"`
	LocationID string    `json:"locationId" gorm:"type:uuid;not null"`
	Quantity   int       `json:"quantity" gorm:"not null;default:0"`
	UpdatedAt  time.Time `json:"updatedAt"`

	// Relations (for eager loading)
	Item     *StockItem     `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	Location *StockLocation `json:"location,omitempty" gorm:"foreignKey:LocationID"`
}

func (s *StockBalance) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

func (StockBalance) TableName() string {
	return "stock_balances"
}

// =============== DTOs ===============

// CreateStockItemRequest DTO
type CreateStockItemRequest struct {
	SKU         string  `json:"sku" validate:"required,min=1,max=100"`
	Name        string  `json:"name" validate:"required,min=1,max=255"`
	Description *string `json:"description"`
	Category    *string `json:"category"`
	Unit        string  `json:"unit" validate:"required,min=1,max=20"`
	MinQty      int     `json:"minQty"`
	TrackSerial bool    `json:"trackSerial"`
}

// UpdateStockItemRequest DTO
type UpdateStockItemRequest struct {
	SKU         *string `json:"sku"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Category    *string `json:"category"`
	Unit        *string `json:"unit"`
	MinQty      *int    `json:"minQty"`
	TrackSerial *bool   `json:"trackSerial"`
	IsActive    *bool   `json:"isActive"`
}

// CreateStockLocationRequest DTO
type CreateStockLocationRequest struct {
	ScopeID string `json:"scopeId" validate:"required,uuid"`
	Type    string `json:"type" validate:"required"`
	Name    string `json:"name" validate:"required,min=1,max=255"`
}

// UpdateStockLocationRequest DTO
type UpdateStockLocationRequest struct {
	ScopeID  *string `json:"scopeId"`
	Type     *string `json:"type"`
	Name     *string `json:"name"`
	IsActive *bool   `json:"isActive"`
}

// CreateStockMovementRequest DTO
type CreateStockMovementRequest struct {
	ScopeID        string            `json:"scopeId" validate:"required,uuid"`
	Type           string            `json:"type" validate:"required"`
	ItemID         string            `json:"itemId" validate:"required,uuid"`
	FromLocationID string            `json:"fromLocationId"`
	ToLocationID   string            `json:"toLocationId"`
	TicketID       string            `json:"ticketId"`
	Quantity       int               `json:"quantity" validate:"required,gt=0"`
	UnitCost       *decimal.Decimal  `json:"unitCost"`
	Notes          string            `json:"notes"`
}

// InventoryCountRequest DTO
type InventoryCountRequest struct {
	ScopeID         string  `json:"scopeId" validate:"required,uuid"`
	LocationID      string  `json:"locationId" validate:"required,uuid"`
	ItemID          string  `json:"itemId" validate:"required,uuid"`
	CountedQuantity int     `json:"countedQuantity" validate:"gte=0"`
	Notes           *string `json:"notes"`
}

// InventoryCountResponse DTO
type InventoryCountResponse struct {
	ItemID         string `json:"itemId"`
	LocationID     string `json:"locationId"`
	PreviousQty    int    `json:"previousQty"`
	CountedQty     int    `json:"countedQty"`
	Delta          int    `json:"delta"`
	AdjustmentMade bool   `json:"adjustmentMade"`
	MovementID     string `json:"movementId,omitempty"`
}

// StockBalanceResponse with joined data
type StockBalanceResponse struct {
	ID           string  `json:"id"`
	ScopeID      string  `json:"scopeId"`
	ItemID       string  `json:"itemId"`
	LocationID   string  `json:"locationId"`
	Quantity     int     `json:"quantity"`
	ItemSKU      string  `json:"itemSku"`
	ItemName     string  `json:"itemName"`
	ItemUnit     string  `json:"itemUnit"`
	LocationName string  `json:"locationName"`
	LocationType string  `json:"locationType"`
	UpdatedAt    string  `json:"updatedAt"`
}

// =============== Filters ===============

type StockItemFilter struct {
	Search   string
	Category string
	IsActive *bool
	Page     int
	PageSize int
}

type StockLocationFilter struct {
	ScopeID  string
	Type     string
	Search   string
	IsActive *bool
	Page     int
	PageSize int
}

type StockBalanceFilter struct {
	ScopeID    string
	ItemID     string
	LocationID string
	Search     string
	LowStock   bool // quantity <= minQty
	Page       int
	PageSize   int
}

type StockMovementFilter struct {
	ScopeID    string
	Type       string
	ItemID     string
	LocationID string
	TicketID   string
	StartDate  *time.Time
	EndDate    *time.Time
	Page       int
	PageSize   int
}

// =============== Paginated Responses ===============

type PaginatedStockItems struct {
	Data       []StockItem `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

type PaginatedStockLocations struct {
	Data       []StockLocation `json:"data"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
}

type PaginatedStockBalances struct {
	Data       []StockBalanceResponse `json:"data"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"pageSize"`
	TotalPages int                    `json:"totalPages"`
}

type PaginatedStockMovements struct {
	Data       []StockMovement `json:"data"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
}
