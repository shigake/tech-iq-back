package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"github.com/shigake/tech-iq-back/internal/services"
)

type FinancialHandler struct {
	service      *services.FinancialService
	categoryRepo repositories.CategoryRepository
}

func NewFinancialHandler(service *services.FinancialService, categoryRepo repositories.CategoryRepository) *FinancialHandler {
	return &FinancialHandler{service: service, categoryRepo: categoryRepo}
}

// =============== Financial Entries ===============

// CreateEntry creates a new financial entry
// @Summary Create financial entry
// @Tags Financial
// @Accept json
// @Produce json
// @Param body body models.CreateFinancialEntryRequest true "Entry data"
// @Success 201 {object} models.FinancialEntry
// @Router /financial/entries [post]
func (h *FinancialHandler) CreateEntry(c *fiber.Ctx) error {
	var req models.CreateFinancialEntryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Type == "" || req.Category == "" || req.Description == "" || req.Amount <= 0 || req.EntryDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing required fields: type, category, description, amount, entryDate",
		})
	}

	userID := c.Locals("userId").(string)
	ip := c.IP()
	userAgent := c.Get("User-Agent")

	entry, err := h.service.CreateEntry(req, userID, ip, userAgent)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(entry)
}

// GetEntry retrieves a financial entry by ID
// @Summary Get financial entry
// @Tags Financial
// @Produce json
// @Param id path string true "Entry ID"
// @Success 200 {object} models.FinancialEntry
// @Router /financial/entries/{id} [get]
func (h *FinancialHandler) GetEntry(c *fiber.Ctx) error {
	id := c.Params("id")

	entry, err := h.service.GetEntryByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Entry not found",
		})
	}

	return c.JSON(entry)
}

// UpdateEntry updates a financial entry
// @Summary Update financial entry
// @Tags Financial
// @Accept json
// @Produce json
// @Param id path string true "Entry ID"
// @Param body body models.UpdateFinancialEntryRequest true "Entry data"
// @Success 200 {object} models.FinancialEntry
// @Router /financial/entries/{id} [put]
func (h *FinancialHandler) UpdateEntry(c *fiber.Ctx) error {
	id := c.Params("id")

	var req models.UpdateFinancialEntryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Version <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Version is required for optimistic locking",
		})
	}

	userID := c.Locals("userId").(string)
	ip := c.IP()
	userAgent := c.Get("User-Agent")

	entry, err := h.service.UpdateEntry(id, req, userID, ip, userAgent)
	if err != nil {
		if err.Error() == "entry was modified by another user, please refresh and try again" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(entry)
}

// UpdateEntryStatus updates the status of a financial entry
// @Summary Update financial entry status
// @Tags Financial
// @Accept json
// @Produce json
// @Param id path string true "Entry ID"
// @Param body body models.UpdateFinancialEntryStatusRequest true "Status data"
// @Success 200 {object} models.FinancialEntry
// @Router /financial/entries/{id}/status [patch]
func (h *FinancialHandler) UpdateEntryStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	var req models.UpdateFinancialEntryStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Status is required",
		})
	}

	userID := c.Locals("userId").(string)
	ip := c.IP()
	userAgent := c.Get("User-Agent")

	entry, err := h.service.UpdateEntryStatus(id, req, userID, ip, userAgent)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(entry)
}

// DeleteEntry deletes a financial entry
// @Summary Delete financial entry
// @Tags Financial
// @Param id path string true "Entry ID"
// @Success 204
// @Router /financial/entries/{id} [delete]
func (h *FinancialHandler) DeleteEntry(c *fiber.Ctx) error {
	id := c.Params("id")

	userID := c.Locals("userId").(string)
	ip := c.IP()
	userAgent := c.Get("User-Agent")

	if err := h.service.DeleteEntry(id, userID, ip, userAgent); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ListEntries lists financial entries with filters
// @Summary List financial entries
// @Tags Financial
// @Produce json
// @Param type query string false "Entry type (income/expense)"
// @Param status query string false "Status (pending/paid/overdue/cancelled)"
// @Param category query string false "Category"
// @Param startDate query string false "Start date (YYYY-MM-DD)"
// @Param endDate query string false "End date (YYYY-MM-DD)"
// @Param technicianId query string false "Technician ID"
// @Param clientId query string false "Client ID"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Router /financial/entries [get]
func (h *FinancialHandler) ListEntries(c *fiber.Ctx) error {
	filter := models.FinancialEntryFilter{
		Type:         models.FinancialEntryType(c.Query("type")),
		Status:       models.FinancialEntryStatus(c.Query("status")),
		Category:     c.Query("category"),
		StartDate:    c.Query("startDate"),
		EndDate:      c.Query("endDate"),
		TechnicianID: c.Query("technicianId"),
		ClientID:     c.Query("clientId"),
		TicketID:     c.Query("ticketId"),
		Page:         c.QueryInt("page", 1),
		Limit:        c.QueryInt("limit", 20),
	}

	entries, total, err := h.service.ListEntries(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"entries": entries,
		"total":   total,
		"page":    filter.Page,
		"limit":   filter.Limit,
	})
}

// =============== Payment Batches ===============

// CreateBatch creates a new payment batch
// @Summary Create payment batch
// @Tags Financial
// @Accept json
// @Produce json
// @Param body body models.CreatePaymentBatchRequest true "Batch data"
// @Success 201 {object} models.PaymentBatch
// @Router /financial/batches [post]
func (h *FinancialHandler) CreateBatch(c *fiber.Ctx) error {
	var req models.CreatePaymentBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" || req.PeriodStart == "" || req.PeriodEnd == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing required fields: name, periodStart, periodEnd",
		})
	}

	userID := c.Locals("userId").(string)
	ip := c.IP()
	userAgent := c.Get("User-Agent")

	batch, err := h.service.CreateBatch(req, userID, ip, userAgent)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(batch)
}

// GetBatch retrieves a payment batch by ID
// @Summary Get payment batch
// @Tags Financial
// @Produce json
// @Param id path string true "Batch ID"
// @Success 200 {object} models.PaymentBatch
// @Router /financial/batches/{id} [get]
func (h *FinancialHandler) GetBatch(c *fiber.Ctx) error {
	id := c.Params("id")

	batch, err := h.service.GetBatchByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Batch not found",
		})
	}

	return c.JSON(batch)
}

// ListBatches lists payment batches with filters
// @Summary List payment batches
// @Tags Financial
// @Produce json
// @Param status query string false "Status"
// @Param periodStart query string false "Period start (YYYY-MM-DD)"
// @Param periodEnd query string false "Period end (YYYY-MM-DD)"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Router /financial/batches [get]
func (h *FinancialHandler) ListBatches(c *fiber.Ctx) error {
	filter := models.PaymentBatchFilter{
		Status:      models.PaymentBatchStatus(c.Query("status")),
		PeriodStart: c.Query("periodStart"),
		PeriodEnd:   c.Query("periodEnd"),
		Page:        c.QueryInt("page", 1),
		Limit:       c.QueryInt("limit", 20),
	}

	batches, total, err := h.service.ListBatches(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"batches": batches,
		"total":   total,
		"page":    filter.Page,
		"limit":   filter.Limit,
	})
}

// AddEntriesToBatch adds entries to a payment batch
// @Summary Add entries to batch
// @Tags Financial
// @Accept json
// @Produce json
// @Param id path string true "Batch ID"
// @Param body body models.AddBatchEntriesRequest true "Entry IDs"
// @Success 200 {object} models.PaymentBatch
// @Router /financial/batches/{id}/entries [post]
func (h *FinancialHandler) AddEntriesToBatch(c *fiber.Ctx) error {
	id := c.Params("id")

	var req models.AddBatchEntriesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.EntryIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one entry ID is required",
		})
	}

	userID := c.Locals("userId").(string)
	ip := c.IP()
	userAgent := c.Get("User-Agent")

	batch, err := h.service.AddEntriesToBatch(id, req, userID, ip, userAgent)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(batch)
}

// RemoveEntryFromBatch removes an entry from a payment batch
// @Summary Remove entry from batch
// @Tags Financial
// @Param id path string true "Batch ID"
// @Param entryId path string true "Entry ID"
// @Success 204
// @Router /financial/batches/{id}/entries/{entryId} [delete]
func (h *FinancialHandler) RemoveEntryFromBatch(c *fiber.Ctx) error {
	batchID := c.Params("id")
	entryID := c.Params("entryId")

	userID := c.Locals("userId").(string)
	ip := c.IP()
	userAgent := c.Get("User-Agent")

	if err := h.service.RemoveEntryFromBatch(batchID, entryID, userID, ip, userAgent); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ApproveBatch approves a payment batch
// @Summary Approve payment batch
// @Tags Financial
// @Produce json
// @Param id path string true "Batch ID"
// @Success 200 {object} models.PaymentBatch
// @Router /financial/batches/{id}/approve [patch]
func (h *FinancialHandler) ApproveBatch(c *fiber.Ctx) error {
	id := c.Params("id")

	userID := c.Locals("userId").(string)
	ip := c.IP()
	userAgent := c.Get("User-Agent")

	batch, err := h.service.ApproveBatch(id, userID, ip, userAgent)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(batch)
}

// PayBatch marks a batch as paid
// @Summary Pay payment batch
// @Tags Financial
// @Accept json
// @Produce json
// @Param id path string true "Batch ID"
// @Param body body models.PayBatchRequest true "Payment data"
// @Success 200 {object} models.PaymentBatch
// @Router /financial/batches/{id}/pay [patch]
func (h *FinancialHandler) PayBatch(c *fiber.Ctx) error {
	id := c.Params("id")

	var req models.PayBatchRequest
	c.BodyParser(&req) // Optional body

	userID := c.Locals("userId").(string)
	ip := c.IP()
	userAgent := c.Get("User-Agent")

	batch, err := h.service.PayBatch(id, req, userID, ip, userAgent)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(batch)
}

// DeleteBatch deletes a payment batch
// @Summary Delete payment batch
// @Tags Financial
// @Param id path string true "Batch ID"
// @Success 204
// @Router /financial/batches/{id} [delete]
func (h *FinancialHandler) DeleteBatch(c *fiber.Ctx) error {
	id := c.Params("id")

	userID := c.Locals("userId").(string)
	ip := c.IP()
	userAgent := c.Get("User-Agent")

	if err := h.service.DeleteBatch(id, userID, ip, userAgent); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// =============== Dashboard & Reports ===============

// GetDashboard retrieves financial dashboard data
// @Summary Get financial dashboard
// @Tags Financial
// @Produce json
// @Param period query string false "Period (today/week/month/quarter/year/custom)"
// @Param startDate query string false "Start date for custom period"
// @Param endDate query string false "End date for custom period"
// @Success 200 {object} models.FinancialDashboard
// @Router /financial/dashboard [get]
func (h *FinancialHandler) GetDashboard(c *fiber.Ctx) error {
	filter := models.DashboardFilter{
		Period:    c.Query("period", "month"),
		StartDate: c.Query("startDate"),
		EndDate:   c.Query("endDate"),
	}

	dashboard, err := h.service.GetDashboard(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(dashboard)
}

// GetCashFlowReport retrieves cash flow report
// @Summary Get cash flow report
// @Tags Financial
// @Produce json
// @Param startDate query string true "Start date (YYYY-MM-DD)"
// @Param endDate query string true "End date (YYYY-MM-DD)"
// @Param groupBy query string false "Group by (day/week/month)"
// @Success 200 {object} models.CashFlowReport
// @Router /financial/reports/cash-flow [get]
func (h *FinancialHandler) GetCashFlowReport(c *fiber.Ctx) error {
	filter := models.CashFlowFilter{
		StartDate: c.Query("startDate"),
		EndDate:   c.Query("endDate"),
		GroupBy:   c.Query("groupBy", "day"),
	}

	if filter.StartDate == "" || filter.EndDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "startDate and endDate are required",
		})
	}

	report, err := h.service.GetCashFlowReport(filter)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(report)
}

// GetTechnicianPaymentsReport retrieves technician payments report
// @Summary Get technician payments report
// @Tags Financial
// @Produce json
// @Param startDate query string true "Start date (YYYY-MM-DD)"
// @Param endDate query string true "End date (YYYY-MM-DD)"
// @Param technicianId query string false "Filter by technician ID"
// @Success 200 {object} models.TechnicianPaymentsReport
// @Router /financial/reports/technician-payments [get]
func (h *FinancialHandler) GetTechnicianPaymentsReport(c *fiber.Ctx) error {
	filter := models.TechnicianPaymentsFilter{
		StartDate:    c.Query("startDate"),
		EndDate:      c.Query("endDate"),
		TechnicianID: c.Query("technicianId"),
	}

	if filter.StartDate == "" || filter.EndDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "startDate and endDate are required",
		})
	}

	report, err := h.service.GetTechnicianPaymentsReport(filter)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(report)
}

// GetCategories returns available financial categories from the database
// @Summary Get financial categories
// @Tags Financial
// @Produce json
// @Param type query string false "Filter by type: finance_income or finance_expense"
// @Success 200 {array} models.Category
// @Router /financial/categories [get]
func (h *FinancialHandler) GetCategories(c *fiber.Ctx) error {
	categoryType := c.Query("type")
	
	if categoryType != "" {
		categories, err := h.categoryRepo.GetByTypeWithChildren(models.CategoryType(categoryType))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch categories",
			})
		}
		return c.JSON(categories)
	}
	
	// Return both income and expense categories
	incomeCategories, err := h.categoryRepo.GetByTypeWithChildren(models.CategoryTypeFinanceIncome)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch income categories",
		})
	}
	
	expenseCategories, err := h.categoryRepo.GetByTypeWithChildren(models.CategoryTypeFinanceExpense)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch expense categories",
		})
	}
	
	// Combine both lists
	allCategories := append(incomeCategories, expenseCategories...)
	return c.JSON(allCategories)
}
