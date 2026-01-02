package services

import (
	"errors"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"gorm.io/gorm"
)

var (
	ErrItemNotFound           = errors.New("item not found")
	ErrLocationNotFound       = errors.New("location not found")
	ErrMovementNotFound       = errors.New("movement not found")
	ErrInsufficientStock      = errors.New("insufficient stock balance")
	ErrInvalidMovementType    = errors.New("invalid movement type")
	ErrMissingFromLocation    = errors.New("from_location_id is required for this movement type")
	ErrMissingToLocation      = errors.New("to_location_id is required for this movement type")
	ErrTransferSameLocation   = errors.New("transfer must be between different locations")
	ErrNegativeQuantity       = errors.New("quantity must be greater than zero")
	ErrItemSKUExists          = errors.New("SKU already exists")
)

type StockService interface {
	// Items
	CreateItem(req models.CreateStockItemRequest) (*models.StockItem, error)
	GetItem(id string) (*models.StockItem, error)
	UpdateItem(id string, req models.UpdateStockItemRequest) (*models.StockItem, error)
	DeleteItem(id string) error
	ListItems(filter models.StockItemFilter) (*models.PaginatedStockItems, error)

	// Locations
	CreateLocation(req models.CreateStockLocationRequest) (*models.StockLocation, error)
	GetLocation(id string) (*models.StockLocation, error)
	UpdateLocation(id string, req models.UpdateStockLocationRequest) (*models.StockLocation, error)
	DeleteLocation(id string) error
	ListLocations(filter models.StockLocationFilter) (*models.PaginatedStockLocations, error)

	// Movements with transactional balance update
	CreateMovement(req models.CreateStockMovementRequest, userID string) (*models.StockMovement, error)
	GetMovement(id string) (*models.StockMovement, error)
	ListMovements(filter models.StockMovementFilter) (*models.PaginatedStockMovements, error)

	// Balances
	GetBalance(itemID, locationID string) (*models.StockBalance, error)
	ListBalances(filter models.StockBalanceFilter) (*models.PaginatedStockBalances, error)

	// Inventory Count
	PerformInventoryCount(req models.InventoryCountRequest, userID string) (*models.InventoryCountResponse, error)
}

type stockService struct {
	repo repositories.StockRepository
}

func NewStockService(repo repositories.StockRepository) StockService {
	return &stockService{repo: repo}
}

// =============== Items ===============

func (s *stockService) CreateItem(req models.CreateStockItemRequest) (*models.StockItem, error) {
	// Check if SKU already exists
	existing, err := s.repo.GetItemBySKU(req.SKU)
	if err == nil && existing != nil {
		return nil, ErrItemSKUExists
	}

	item := &models.StockItem{
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Unit:        req.Unit,
		MinQty:      req.MinQty,
		TrackSerial: req.TrackSerial,
		IsActive:    true,
	}

	err = s.repo.CreateItem(item)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (s *stockService) GetItem(id string) (*models.StockItem, error) {
	item, err := s.repo.GetItemByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}
	return item, nil
}

func (s *stockService) UpdateItem(id string, req models.UpdateStockItemRequest) (*models.StockItem, error) {
	item, err := s.repo.GetItemByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	// Check SKU uniqueness if changed
	if req.SKU != nil && *req.SKU != item.SKU {
		existing, err := s.repo.GetItemBySKU(*req.SKU)
		if err == nil && existing != nil {
			return nil, ErrItemSKUExists
		}
		item.SKU = *req.SKU
	}

	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Category != nil {
		item.Category = *req.Category
	}
	if req.Unit != nil {
		item.Unit = *req.Unit
	}
	if req.MinQty != nil {
		item.MinQty = *req.MinQty
	}
	if req.TrackSerial != nil {
		item.TrackSerial = *req.TrackSerial
	}
	if req.IsActive != nil {
		item.IsActive = *req.IsActive
	}

	err = s.repo.UpdateItem(item)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (s *stockService) DeleteItem(id string) error {
	_, err := s.repo.GetItemByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrItemNotFound
		}
		return err
	}
	return s.repo.DeleteItem(id)
}

func (s *stockService) ListItems(filter models.StockItemFilter) (*models.PaginatedStockItems, error) {
	return s.repo.ListItems(filter)
}

// =============== Locations ===============

func (s *stockService) CreateLocation(req models.CreateStockLocationRequest) (*models.StockLocation, error) {
	location := &models.StockLocation{
		ScopeID:  req.ScopeID,
		Type:     models.StockLocationType(req.Type),
		Name:     req.Name,
		IsActive: true,
	}

	err := s.repo.CreateLocation(location)
	if err != nil {
		return nil, err
	}

	return location, nil
}

func (s *stockService) GetLocation(id string) (*models.StockLocation, error) {
	location, err := s.repo.GetLocationByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLocationNotFound
		}
		return nil, err
	}
	return location, nil
}

func (s *stockService) UpdateLocation(id string, req models.UpdateStockLocationRequest) (*models.StockLocation, error) {
	location, err := s.repo.GetLocationByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLocationNotFound
		}
		return nil, err
	}

	if req.Name != nil {
		location.Name = *req.Name
	}
	if req.Type != nil {
		location.Type = models.StockLocationType(*req.Type)
	}
	if req.IsActive != nil {
		location.IsActive = *req.IsActive
	}

	err = s.repo.UpdateLocation(location)
	if err != nil {
		return nil, err
	}

	return location, nil
}

func (s *stockService) DeleteLocation(id string) error {
	_, err := s.repo.GetLocationByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLocationNotFound
		}
		return err
	}
	return s.repo.DeleteLocation(id)
}

func (s *stockService) ListLocations(filter models.StockLocationFilter) (*models.PaginatedStockLocations, error) {
	return s.repo.ListLocations(filter)
}

// =============== Movements ===============

func (s *stockService) CreateMovement(req models.CreateStockMovementRequest, userID string) (*models.StockMovement, error) {
	// Validate quantity
	if req.Quantity <= 0 {
		return nil, ErrNegativeQuantity
	}

	// Validate movement type and required locations
	movementType := models.StockMovementType(req.Type)
	if err := s.validateMovementLocations(movementType, req.FromLocationID, req.ToLocationID); err != nil {
		return nil, err
	}

	// Validate item exists
	_, err := s.repo.GetItemByID(req.ItemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	// Validate locations exist
	if req.FromLocationID != "" {
		_, err := s.repo.GetLocationByID(req.FromLocationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrLocationNotFound
			}
			return nil, err
		}
	}

	if req.ToLocationID != "" {
		_, err := s.repo.GetLocationByID(req.ToLocationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrLocationNotFound
			}
			return nil, err
		}
	}

	// Begin transaction
	tx := s.repo.BeginTx()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update balances based on movement type
	switch movementType {
	case models.MovementTypeEntradaCompra, models.MovementTypeEntradaDevolucao:
		// Entry: increase balance at toLocation
		if err := s.increaseBalance(tx, req.ScopeID, req.ItemID, req.ToLocationID, req.Quantity); err != nil {
			tx.Rollback()
			return nil, err
		}

	case models.MovementTypeSaidaConsumoOS, models.MovementTypeSaidaPerda:
		// Exit: decrease balance at fromLocation
		if err := s.decreaseBalance(tx, req.ScopeID, req.ItemID, req.FromLocationID, req.Quantity); err != nil {
			tx.Rollback()
			return nil, err
		}

	case models.MovementTypeTransferencia:
		// Transfer: decrease from source, increase at destination
		if err := s.decreaseBalance(tx, req.ScopeID, req.ItemID, req.FromLocationID, req.Quantity); err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := s.increaseBalance(tx, req.ScopeID, req.ItemID, req.ToLocationID, req.Quantity); err != nil {
			tx.Rollback()
			return nil, err
		}

	case models.MovementTypeAjusteInventario:
		// Adjustment can be positive (to) or negative (from)
		if req.ToLocationID != "" {
			if err := s.increaseBalance(tx, req.ScopeID, req.ItemID, req.ToLocationID, req.Quantity); err != nil {
				tx.Rollback()
				return nil, err
			}
		} else if req.FromLocationID != "" {
			if err := s.decreaseBalance(tx, req.ScopeID, req.ItemID, req.FromLocationID, req.Quantity); err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}

	// Create the movement record (immutable ledger entry)
	movement := &models.StockMovement{
		ScopeID:        req.ScopeID,
		Type:           movementType,
		ItemID:         req.ItemID,
		FromLocationID: req.FromLocationID,
		ToLocationID:   req.ToLocationID,
		TicketID:       req.TicketID,
		Quantity:       req.Quantity,
		UnitCost:       req.UnitCost,
		Notes:          req.Notes,
		PerformedBy:    userID,
		PerformedAt:    time.Now(),
	}

	if err := s.repo.CreateMovementTx(tx, movement); err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Reload with relations
	return s.repo.GetMovementByID(movement.ID)
}

func (s *stockService) validateMovementLocations(movementType models.StockMovementType, fromLocationID, toLocationID string) error {
	switch movementType {
	case models.MovementTypeEntradaCompra, models.MovementTypeEntradaDevolucao:
		if toLocationID == "" {
			return ErrMissingToLocation
		}
	case models.MovementTypeSaidaConsumoOS, models.MovementTypeSaidaPerda:
		if fromLocationID == "" {
			return ErrMissingFromLocation
		}
	case models.MovementTypeTransferencia:
		if fromLocationID == "" {
			return ErrMissingFromLocation
		}
		if toLocationID == "" {
			return ErrMissingToLocation
		}
		if fromLocationID == toLocationID {
			return ErrTransferSameLocation
		}
	case models.MovementTypeAjusteInventario:
		if fromLocationID == "" && toLocationID == "" {
			return ErrMissingToLocation
		}
	default:
		return ErrInvalidMovementType
	}
	return nil
}

func (s *stockService) increaseBalance(tx *gorm.DB, scopeID, itemID, locationID string, quantity int) error {
	balance, err := s.repo.GetBalanceForUpdate(tx, itemID, locationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new balance
			balance = &models.StockBalance{
				ScopeID:    scopeID,
				ItemID:     itemID,
				LocationID: locationID,
				Quantity:   quantity,
			}
		} else {
			return err
		}
	} else {
		balance.Quantity += quantity
	}

	return s.repo.UpsertBalance(tx, balance)
}

func (s *stockService) decreaseBalance(tx *gorm.DB, scopeID, itemID, locationID string, quantity int) error {
	balance, err := s.repo.GetBalanceForUpdate(tx, itemID, locationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInsufficientStock
		}
		return err
	}

	newQuantity := balance.Quantity - quantity
	if newQuantity < 0 {
		return ErrInsufficientStock
	}

	balance.Quantity = newQuantity
	return s.repo.UpsertBalance(tx, balance)
}

func (s *stockService) GetMovement(id string) (*models.StockMovement, error) {
	movement, err := s.repo.GetMovementByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMovementNotFound
		}
		return nil, err
	}
	return movement, nil
}

func (s *stockService) ListMovements(filter models.StockMovementFilter) (*models.PaginatedStockMovements, error) {
	return s.repo.ListMovements(filter)
}

// =============== Balances ===============

func (s *stockService) GetBalance(itemID, locationID string) (*models.StockBalance, error) {
	return s.repo.GetBalance(itemID, locationID)
}

func (s *stockService) ListBalances(filter models.StockBalanceFilter) (*models.PaginatedStockBalances, error) {
	return s.repo.ListBalances(filter)
}

// =============== Inventory Count ===============

func (s *stockService) PerformInventoryCount(req models.InventoryCountRequest, userID string) (*models.InventoryCountResponse, error) {
	// Get current balance
	currentQty := 0
	balance, err := s.repo.GetBalance(req.ItemID, req.LocationID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if balance != nil {
		currentQty = balance.Quantity
	}

	delta := req.CountedQuantity - currentQty
	
	// If no change, return response without creating movement
	if delta == 0 {
		return &models.InventoryCountResponse{
			ItemID:          req.ItemID,
			LocationID:      req.LocationID,
			PreviousQty:     currentQty,
			CountedQty:      req.CountedQuantity,
			Delta:           0,
			AdjustmentMade:  false,
		}, nil
	}

	// Create adjustment movement
	movementReq := models.CreateStockMovementRequest{
		ScopeID:  req.ScopeID,
		Type:     string(models.MovementTypeAjusteInventario),
		ItemID:   req.ItemID,
		Quantity: abs(delta),
		Notes:    req.Notes,
	}

	if delta > 0 {
		// Positive adjustment (add stock)
		movementReq.ToLocationID = req.LocationID
	} else {
		// Negative adjustment (remove stock)
		movementReq.FromLocationID = req.LocationID
	}

	movement, err := s.CreateMovement(movementReq, userID)
	if err != nil {
		return nil, err
	}

	return &models.InventoryCountResponse{
		ItemID:          req.ItemID,
		LocationID:      req.LocationID,
		PreviousQty:     currentQty,
		CountedQty:      req.CountedQuantity,
		Delta:           delta,
		AdjustmentMade:  true,
		MovementID:      movement.ID,
	}, nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
