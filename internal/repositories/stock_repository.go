package repositories

import (
	"fmt"
	"math"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StockRepository interface {
	// Items
	CreateItem(item *models.StockItem) error
	GetItemByID(id string) (*models.StockItem, error)
	GetItemBySKU(sku string) (*models.StockItem, error)
	UpdateItem(item *models.StockItem) error
	DeleteItem(id string) error
	ListItems(filter models.StockItemFilter) (*models.PaginatedStockItems, error)

	// Locations
	CreateLocation(location *models.StockLocation) error
	GetLocationByID(id string) (*models.StockLocation, error)
	UpdateLocation(location *models.StockLocation) error
	DeleteLocation(id string) error
	ListLocations(filter models.StockLocationFilter) (*models.PaginatedStockLocations, error)

	// Movements (ledger - immutable)
	CreateMovement(movement *models.StockMovement) error
	GetMovementByID(id string) (*models.StockMovement, error)
	ListMovements(filter models.StockMovementFilter) (*models.PaginatedStockMovements, error)

	// Balances
	GetBalance(itemID, locationID string) (*models.StockBalance, error)
	GetBalanceForUpdate(tx *gorm.DB, itemID, locationID string) (*models.StockBalance, error)
	UpsertBalance(tx *gorm.DB, balance *models.StockBalance) error
	ListBalances(filter models.StockBalanceFilter) (*models.PaginatedStockBalances, error)

	// Transaction support
	BeginTx() *gorm.DB
	CreateMovementTx(tx *gorm.DB, movement *models.StockMovement) error
}

type stockRepository struct {
	db *gorm.DB
}

func NewStockRepository(db *gorm.DB) StockRepository {
	return &stockRepository{db: db}
}

// =============== Items ===============

func (r *stockRepository) CreateItem(item *models.StockItem) error {
	return r.db.Create(item).Error
}

func (r *stockRepository) GetItemByID(id string) (*models.StockItem, error) {
	var item models.StockItem
	err := r.db.Where("id = ?", id).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *stockRepository) GetItemBySKU(sku string) (*models.StockItem, error) {
	var item models.StockItem
	err := r.db.Where("sku = ?", sku).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *stockRepository) UpdateItem(item *models.StockItem) error {
	return r.db.Save(item).Error
}

func (r *stockRepository) DeleteItem(id string) error {
	// Soft delete by setting isActive = false
	return r.db.Model(&models.StockItem{}).Where("id = ?", id).Update("is_active", false).Error
}

func (r *stockRepository) ListItems(filter models.StockItemFilter) (*models.PaginatedStockItems, error) {
	var items []models.StockItem
	var total int64

	query := r.db.Model(&models.StockItem{})

	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR sku ILIKE ? OR description ILIKE ?", search, search, search)
	}

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	offset := (filter.Page - 1) * filter.PageSize
	err = query.Order("name ASC").Offset(offset).Limit(filter.PageSize).Find(&items).Error
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PageSize)))

	return &models.PaginatedStockItems{
		Data:       items,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// =============== Locations ===============

func (r *stockRepository) CreateLocation(location *models.StockLocation) error {
	return r.db.Create(location).Error
}

func (r *stockRepository) GetLocationByID(id string) (*models.StockLocation, error) {
	var location models.StockLocation
	err := r.db.Where("id = ?", id).First(&location).Error
	if err != nil {
		return nil, err
	}
	return &location, nil
}

func (r *stockRepository) UpdateLocation(location *models.StockLocation) error {
	return r.db.Save(location).Error
}

func (r *stockRepository) DeleteLocation(id string) error {
	return r.db.Model(&models.StockLocation{}).Where("id = ?", id).Update("is_active", false).Error
}

func (r *stockRepository) ListLocations(filter models.StockLocationFilter) (*models.PaginatedStockLocations, error) {
	var locations []models.StockLocation
	var total int64

	query := r.db.Model(&models.StockLocation{})

	if filter.ScopeID != "" {
		query = query.Where("scope_id = ?", filter.ScopeID)
	}

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ?", search)
	}

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	offset := (filter.Page - 1) * filter.PageSize
	err = query.Order("name ASC").Offset(offset).Limit(filter.PageSize).Find(&locations).Error
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PageSize)))

	return &models.PaginatedStockLocations{
		Data:       locations,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// =============== Movements ===============

func (r *stockRepository) CreateMovement(movement *models.StockMovement) error {
	return r.db.Create(movement).Error
}

func (r *stockRepository) CreateMovementTx(tx *gorm.DB, movement *models.StockMovement) error {
	return tx.Create(movement).Error
}

func (r *stockRepository) GetMovementByID(id string) (*models.StockMovement, error) {
	var movement models.StockMovement
	err := r.db.Preload("Item").Preload("FromLocation").Preload("ToLocation").Preload("Performer").
		Where("id = ?", id).First(&movement).Error
	if err != nil {
		return nil, err
	}
	return &movement, nil
}

func (r *stockRepository) ListMovements(filter models.StockMovementFilter) (*models.PaginatedStockMovements, error) {
	var movements []models.StockMovement
	var total int64

	query := r.db.Model(&models.StockMovement{})

	if filter.ScopeID != "" {
		query = query.Where("scope_id = ?", filter.ScopeID)
	}

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	if filter.ItemID != "" {
		query = query.Where("item_id = ?", filter.ItemID)
	}

	if filter.LocationID != "" {
		query = query.Where("from_location_id = ? OR to_location_id = ?", filter.LocationID, filter.LocationID)
	}

	if filter.TicketID != "" {
		query = query.Where("ticket_id = ?", filter.TicketID)
	}

	if filter.StartDate != nil {
		query = query.Where("performed_at >= ?", *filter.StartDate)
	}

	if filter.EndDate != nil {
		query = query.Where("performed_at <= ?", *filter.EndDate)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	offset := (filter.Page - 1) * filter.PageSize
	err = query.Preload("Item").Preload("FromLocation").Preload("ToLocation").
		Order("performed_at DESC").Offset(offset).Limit(filter.PageSize).Find(&movements).Error
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PageSize)))

	return &models.PaginatedStockMovements{
		Data:       movements,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// =============== Balances ===============

func (r *stockRepository) GetBalance(itemID, locationID string) (*models.StockBalance, error) {
	var balance models.StockBalance
	err := r.db.Where("item_id = ? AND location_id = ?", itemID, locationID).First(&balance).Error
	if err != nil {
		return nil, err
	}
	return &balance, nil
}

// GetBalanceForUpdate uses SELECT ... FOR UPDATE to lock the row
func (r *stockRepository) GetBalanceForUpdate(tx *gorm.DB, itemID, locationID string) (*models.StockBalance, error) {
	var balance models.StockBalance
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("item_id = ? AND location_id = ?", itemID, locationID).
		First(&balance).Error
	if err != nil {
		return nil, err
	}
	return &balance, nil
}

// UpsertBalance creates or updates the balance
func (r *stockRepository) UpsertBalance(tx *gorm.DB, balance *models.StockBalance) error {
	balance.UpdatedAt = time.Now()
	
	// Use upsert with conflict handling
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "item_id"}, {Name: "location_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"quantity", "updated_at"}),
	}).Create(balance)
	
	return result.Error
}

func (r *stockRepository) ListBalances(filter models.StockBalanceFilter) (*models.PaginatedStockBalances, error) {
	var total int64

	query := r.db.Model(&models.StockBalance{}).
		Joins("JOIN stock_items ON stock_items.id = stock_balances.item_id").
		Joins("JOIN stock_locations ON stock_locations.id = stock_balances.location_id")

	if filter.ScopeID != "" {
		query = query.Where("stock_balances.scope_id = ?", filter.ScopeID)
	}

	if filter.ItemID != "" {
		query = query.Where("stock_balances.item_id = ?", filter.ItemID)
	}

	if filter.LocationID != "" {
		query = query.Where("stock_balances.location_id = ?", filter.LocationID)
	}

	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("stock_items.name ILIKE ? OR stock_items.sku ILIKE ? OR stock_locations.name ILIKE ?", 
			search, search, search)
	}

	if filter.LowStock {
		query = query.Where("stock_balances.quantity <= stock_items.min_qty")
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	offset := (filter.Page - 1) * filter.PageSize

	// Select with joins
	var results []struct {
		ID           string
		ScopeID      string
		ItemID       string
		LocationID   string
		Quantity     int
		UpdatedAt    time.Time
		ItemSKU      string
		ItemName     string
		ItemUnit     string
		LocationName string
		LocationType string
	}

	err = r.db.Table("stock_balances").
		Select(`stock_balances.id, stock_balances.scope_id, stock_balances.item_id, 
				stock_balances.location_id, stock_balances.quantity, stock_balances.updated_at,
				stock_items.sku as item_sku, stock_items.name as item_name, stock_items.unit as item_unit,
				stock_locations.name as location_name, stock_locations.type as location_type`).
		Joins("JOIN stock_items ON stock_items.id = stock_balances.item_id").
		Joins("JOIN stock_locations ON stock_locations.id = stock_balances.location_id").
		Where(buildBalanceConditions(filter)).
		Order("stock_items.name ASC, stock_locations.name ASC").
		Offset(offset).Limit(filter.PageSize).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Convert to response DTOs
	data := make([]models.StockBalanceResponse, len(results))
	for i, r := range results {
		data[i] = models.StockBalanceResponse{
			ID:           r.ID,
			ScopeID:      r.ScopeID,
			ItemID:       r.ItemID,
			LocationID:   r.LocationID,
			Quantity:     r.Quantity,
			ItemSKU:      r.ItemSKU,
			ItemName:     r.ItemName,
			ItemUnit:     r.ItemUnit,
			LocationName: r.LocationName,
			LocationType: r.LocationType,
			UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PageSize)))

	return &models.PaginatedStockBalances{
		Data:       data,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func buildBalanceConditions(filter models.StockBalanceFilter) string {
	conditions := "1=1"
	if filter.ScopeID != "" {
		conditions += fmt.Sprintf(" AND stock_balances.scope_id = '%s'", filter.ScopeID)
	}
	if filter.ItemID != "" {
		conditions += fmt.Sprintf(" AND stock_balances.item_id = '%s'", filter.ItemID)
	}
	if filter.LocationID != "" {
		conditions += fmt.Sprintf(" AND stock_balances.location_id = '%s'", filter.LocationID)
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		conditions += fmt.Sprintf(" AND (stock_items.name ILIKE '%s' OR stock_items.sku ILIKE '%s' OR stock_locations.name ILIKE '%s')", 
			search, search, search)
	}
	if filter.LowStock {
		conditions += " AND stock_balances.quantity <= stock_items.min_qty"
	}
	return conditions
}

// =============== Transaction ===============

func (r *stockRepository) BeginTx() *gorm.DB {
	return r.db.Begin()
}
