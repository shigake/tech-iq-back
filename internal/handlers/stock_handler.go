package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/middleware"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/services"
)

type StockHandler struct {
	service services.StockService
}

func NewStockHandler(service services.StockService) *StockHandler {
	return &StockHandler{service: service}
}

// =============== Items ===============

// CreateItem godoc
// @Summary Create a new stock item
// @Tags Stock Items
// @Accept json
// @Produce json
// @Param request body models.CreateStockItemRequest true "Item data"
// @Success 201 {object} models.StockItem
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /stock/items [post]
func (h *StockHandler) CreateItem(c *fiber.Ctx) error {
	var req models.CreateStockItemRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "Invalid request body"})
	}

	if req.SKU == "" || req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "SKU and Name are required"})
	}

	item, err := h.service.CreateItem(req)
	if err != nil {
		if err == services.ErrItemSKUExists {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(item)
}

// GetItem godoc
// @Summary Get a stock item by ID
// @Tags Stock Items
// @Produce json
// @Param id path string true "Item ID"
// @Success 200 {object} models.StockItem
// @Failure 404 {object} ErrorResponse
// @Router /stock/items/{id} [get]
func (h *StockHandler) GetItem(c *fiber.Ctx) error {
	id := c.Params("id")
	item, err := h.service.GetItem(id)
	if err != nil {
		if err == services.ErrItemNotFound {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}
	return c.JSON(item)
}

// UpdateItem godoc
// @Summary Update a stock item
// @Tags Stock Items
// @Accept json
// @Produce json
// @Param id path string true "Item ID"
// @Param request body models.UpdateStockItemRequest true "Item data"
// @Success 200 {object} models.StockItem
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /stock/items/{id} [put]
func (h *StockHandler) UpdateItem(c *fiber.Ctx) error {
	id := c.Params("id")

	var req models.UpdateStockItemRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "Invalid request body"})
	}

	item, err := h.service.UpdateItem(id, req)
	if err != nil {
		if err == services.ErrItemNotFound {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: err.Error()})
		}
		if err == services.ErrItemSKUExists {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(item)
}

// DeleteItem godoc
// @Summary Delete a stock item (soft delete)
// @Tags Stock Items
// @Param id path string true "Item ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Router /stock/items/{id} [delete]
func (h *StockHandler) DeleteItem(c *fiber.Ctx) error {
	id := c.Params("id")
	err := h.service.DeleteItem(id)
	if err != nil {
		if err == services.ErrItemNotFound {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ListItems godoc
// @Summary List stock items with filters
// @Tags Stock Items
// @Produce json
// @Param search query string false "Search in name, SKU, description"
// @Param category query string false "Filter by category"
// @Param is_active query bool false "Filter by active status"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} models.PaginatedStockItems
// @Router /stock/items [get]
func (h *StockHandler) ListItems(c *fiber.Ctx) error {
	filter := models.StockItemFilter{
		Search:   c.Query("search"),
		Category: c.Query("category"),
		Page:     getIntQuery(c, "page", 1),
		PageSize: getIntQuery(c, "page_size", 20),
	}

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}

	result, err := h.service.ListItems(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(result)
}

// =============== Locations ===============

// CreateLocation godoc
// @Summary Create a new stock location
// @Tags Stock Locations
// @Accept json
// @Produce json
// @Param request body models.CreateStockLocationRequest true "Location data"
// @Success 201 {object} models.StockLocation
// @Failure 400 {object} ErrorResponse
// @Router /stock/locations [post]
func (h *StockHandler) CreateLocation(c *fiber.Ctx) error {
	var req models.CreateStockLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "Invalid request body"})
	}

	if req.Name == "" || req.Type == "" || req.ScopeID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "ScopeID, Type and Name are required"})
	}

	location, err := h.service.CreateLocation(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(location)
}

// GetLocation godoc
// @Summary Get a stock location by ID
// @Tags Stock Locations
// @Produce json
// @Param id path string true "Location ID"
// @Success 200 {object} models.StockLocation
// @Failure 404 {object} ErrorResponse
// @Router /stock/locations/{id} [get]
func (h *StockHandler) GetLocation(c *fiber.Ctx) error {
	id := c.Params("id")
	location, err := h.service.GetLocation(id)
	if err != nil {
		if err == services.ErrLocationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}
	return c.JSON(location)
}

// UpdateLocation godoc
// @Summary Update a stock location
// @Tags Stock Locations
// @Accept json
// @Produce json
// @Param id path string true "Location ID"
// @Param request body models.UpdateStockLocationRequest true "Location data"
// @Success 200 {object} models.StockLocation
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /stock/locations/{id} [put]
func (h *StockHandler) UpdateLocation(c *fiber.Ctx) error {
	id := c.Params("id")

	var req models.UpdateStockLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "Invalid request body"})
	}

	location, err := h.service.UpdateLocation(id, req)
	if err != nil {
		if err == services.ErrLocationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(location)
}

// DeleteLocation godoc
// @Summary Delete a stock location (soft delete)
// @Tags Stock Locations
// @Param id path string true "Location ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Router /stock/locations/{id} [delete]
func (h *StockHandler) DeleteLocation(c *fiber.Ctx) error {
	id := c.Params("id")
	err := h.service.DeleteLocation(id)
	if err != nil {
		if err == services.ErrLocationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ListLocations godoc
// @Summary List stock locations with filters
// @Tags Stock Locations
// @Produce json
// @Param scope_id query string false "Filter by scope ID"
// @Param type query string false "Filter by location type"
// @Param search query string false "Search in name"
// @Param is_active query bool false "Filter by active status"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} models.PaginatedStockLocations
// @Router /stock/locations [get]
func (h *StockHandler) ListLocations(c *fiber.Ctx) error {
	filter := models.StockLocationFilter{
		ScopeID:  c.Query("scope_id"),
		Type:     c.Query("type"),
		Search:   c.Query("search"),
		Page:     getIntQuery(c, "page", 1),
		PageSize: getIntQuery(c, "page_size", 20),
	}

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}

	result, err := h.service.ListLocations(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(result)
}

// =============== Movements ===============

// CreateMovement godoc
// @Summary Create a stock movement (entry, exit, transfer)
// @Tags Stock Movements
// @Accept json
// @Produce json
// @Param request body models.CreateStockMovementRequest true "Movement data"
// @Success 201 {object} models.StockMovement
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Router /stock/movements [post]
func (h *StockHandler) CreateMovement(c *fiber.Ctx) error {
	var req models.CreateStockMovementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "Invalid request body"})
	}

	if req.ScopeID == "" || req.Type == "" || req.ItemID == "" || req.Quantity <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "ScopeID, Type, ItemID and positive Quantity are required"})
	}

	// Get user ID from JWT context
	userID := middleware.GetUserID(c)

	movement, err := h.service.CreateMovement(req, userID)
	if err != nil {
		switch err {
		case services.ErrItemNotFound, services.ErrLocationNotFound:
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: err.Error()})
		case services.ErrInsufficientStock, services.ErrInvalidMovementType,
			services.ErrMissingFromLocation, services.ErrMissingToLocation,
			services.ErrTransferSameLocation, services.ErrNegativeQuantity:
			return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorResponse{Error: err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(movement)
}

// GetMovement godoc
// @Summary Get a stock movement by ID
// @Tags Stock Movements
// @Produce json
// @Param id path string true "Movement ID"
// @Success 200 {object} models.StockMovement
// @Failure 404 {object} ErrorResponse
// @Router /stock/movements/{id} [get]
func (h *StockHandler) GetMovement(c *fiber.Ctx) error {
	id := c.Params("id")
	movement, err := h.service.GetMovement(id)
	if err != nil {
		if err == services.ErrMovementNotFound {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}
	return c.JSON(movement)
}

// ListMovements godoc
// @Summary List stock movements with filters
// @Tags Stock Movements
// @Produce json
// @Param scope_id query string false "Filter by scope ID"
// @Param type query string false "Filter by movement type"
// @Param item_id query string false "Filter by item ID"
// @Param location_id query string false "Filter by location (from or to)"
// @Param ticket_id query string false "Filter by ticket ID"
// @Param start_date query string false "Filter by start date (RFC3339)"
// @Param end_date query string false "Filter by end date (RFC3339)"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} models.PaginatedStockMovements
// @Router /stock/movements [get]
func (h *StockHandler) ListMovements(c *fiber.Ctx) error {
	filter := models.StockMovementFilter{
		ScopeID:    c.Query("scope_id"),
		Type:       c.Query("type"),
		ItemID:     c.Query("item_id"),
		LocationID: c.Query("location_id"),
		TicketID:   c.Query("ticket_id"),
		Page:       getIntQuery(c, "page", 1),
		PageSize:   getIntQuery(c, "page_size", 20),
	}

	// Parse dates if provided
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := parseTime(startDate); err == nil {
			filter.StartDate = &t
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := parseTime(endDate); err == nil {
			filter.EndDate = &t
		}
	}

	result, err := h.service.ListMovements(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(result)
}

// =============== Balances ===============

// ListBalances godoc
// @Summary List stock balances with filters
// @Tags Stock Balances
// @Produce json
// @Param scope_id query string false "Filter by scope ID"
// @Param item_id query string false "Filter by item ID"
// @Param location_id query string false "Filter by location ID"
// @Param search query string false "Search in item name, SKU, location name"
// @Param low_stock query bool false "Filter low stock items"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} models.PaginatedStockBalances
// @Router /stock/balances [get]
func (h *StockHandler) ListBalances(c *fiber.Ctx) error {
	filter := models.StockBalanceFilter{
		ScopeID:    c.Query("scope_id"),
		ItemID:     c.Query("item_id"),
		LocationID: c.Query("location_id"),
		Search:     c.Query("search"),
		LowStock:   c.Query("low_stock") == "true",
		Page:       getIntQuery(c, "page", 1),
		PageSize:   getIntQuery(c, "page_size", 20),
	}

	result, err := h.service.ListBalances(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(result)
}

// GetBalance godoc
// @Summary Get stock balance for a specific item and location
// @Tags Stock Balances
// @Produce json
// @Param item_id query string true "Item ID"
// @Param location_id query string true "Location ID"
// @Success 200 {object} models.StockBalance
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /stock/balances/single [get]
func (h *StockHandler) GetBalance(c *fiber.Ctx) error {
	itemID := c.Query("item_id")
	locationID := c.Query("location_id")

	if itemID == "" || locationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "item_id and location_id are required"})
	}

	balance, err := h.service.GetBalance(itemID, locationID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "Balance not found"})
	}

	return c.JSON(balance)
}

// =============== Inventory Count ===============

// PerformInventoryCount godoc
// @Summary Perform inventory count and create adjustment if needed
// @Tags Stock Balances
// @Accept json
// @Produce json
// @Param request body models.InventoryCountRequest true "Inventory count data"
// @Success 200 {object} models.InventoryCountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Router /stock/inventory-count [post]
func (h *StockHandler) PerformInventoryCount(c *fiber.Ctx) error {
	var req models.InventoryCountRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "Invalid request body"})
	}

	if req.ScopeID == "" || req.ItemID == "" || req.LocationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "ScopeID, ItemID and LocationID are required"})
	}

	if req.CountedQuantity < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "CountedQuantity cannot be negative"})
	}

	userID := middleware.GetUserID(c)

	result, err := h.service.PerformInventoryCount(req, userID)
	if err != nil {
		if err == services.ErrInsufficientStock {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorResponse{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(result)
}

// =============== Route Registration ===============

func (h *StockHandler) RegisterRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	stock := app.Group("/api/v1/stock", authMiddleware)

	// Items - write requires ADMIN or EMPLOYEE
	items := stock.Group("/items")
	items.Get("/", h.ListItems)                                            // All authenticated users
	items.Get("/:id", h.GetItem)                                           // All authenticated users
	items.Post("/", middleware.AdminOrEmployee(), h.CreateItem)            // ADMIN/EMPLOYEE only
	items.Put("/:id", middleware.AdminOrEmployee(), h.UpdateItem)          // ADMIN/EMPLOYEE only
	items.Delete("/:id", middleware.AdminOnly(), h.DeleteItem)             // ADMIN only

	// Locations - write requires ADMIN or EMPLOYEE
	locations := stock.Group("/locations")
	locations.Get("/", h.ListLocations)                                    // All authenticated users
	locations.Get("/:id", h.GetLocation)                                   // All authenticated users
	locations.Post("/", middleware.AdminOrEmployee(), h.CreateLocation)    // ADMIN/EMPLOYEE only
	locations.Put("/:id", middleware.AdminOrEmployee(), h.UpdateLocation)  // ADMIN/EMPLOYEE only
	locations.Delete("/:id", middleware.AdminOnly(), h.DeleteLocation)     // ADMIN only

	// Movements - write requires ADMIN or EMPLOYEE
	movements := stock.Group("/movements")
	movements.Get("/", h.ListMovements)                                    // All authenticated users
	movements.Get("/:id", h.GetMovement)                                   // All authenticated users
	movements.Post("/", middleware.AdminOrEmployee(), h.CreateMovement)    // ADMIN/EMPLOYEE only

	// Balances - read only for all, inventory count for ADMIN/EMPLOYEE
	balances := stock.Group("/balances")
	balances.Get("/", h.ListBalances)                                      // All authenticated users
	balances.Get("/single", h.GetBalance)                                  // All authenticated users

	// Inventory Count - ADMIN or EMPLOYEE only
	stock.Post("/inventory-count", middleware.AdminOrEmployee(), h.PerformInventoryCount)
}

// =============== Helpers ===============

func getIntQuery(c *fiber.Ctx, key string, defaultValue int) int {
	val := c.Query(key)
	if val == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return intVal
}
