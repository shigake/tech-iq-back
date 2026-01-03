package services

import (
	"errors"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type FinancialService struct {
	repo         *repositories.FinancialRepository
	categoryRepo repositories.CategoryRepository
}

func NewFinancialService(repo *repositories.FinancialRepository, categoryRepo repositories.CategoryRepository) *FinancialService {
	return &FinancialService{repo: repo, categoryRepo: categoryRepo}
}

// =============== Financial Entries ===============

// CreateEntry creates a new financial entry
func (s *FinancialService) CreateEntry(req models.CreateFinancialEntryRequest, userID string, ip string, userAgent string) (*models.FinancialEntry, error) {
	// Validate category
	if !s.ValidateCategory(req.Type, req.Category, req.Subcategory) {
		return nil, errors.New("invalid category or subcategory for the given type")
	}

	// Parse dates
	entryDate, err := time.Parse("2006-01-02", req.EntryDate)
	if err != nil {
		return nil, errors.New("invalid entry date format, expected YYYY-MM-DD")
	}

	var dueDate *time.Time
	if req.DueDate != "" {
		parsed, err := time.Parse("2006-01-02", req.DueDate)
		if err != nil {
			return nil, errors.New("invalid due date format, expected YYYY-MM-DD")
		}
		dueDate = &parsed
	}

	entry := &models.FinancialEntry{
		Type:             req.Type,
		Category:         req.Category,
		Subcategory:      req.Subcategory,
		Description:      req.Description,
		Amount:           req.Amount,
		EntryDate:        entryDate,
		DueDate:          dueDate,
		Status:           models.FinancialEntryStatusPending,
		PaymentMethod:    req.PaymentMethod,
		PaymentReference: req.PaymentReference,
		AttachmentURLs:   req.AttachmentURLs,
		CreatedBy:        userID,
		Version:          1,
	}

	// Set optional relationships
	if req.TicketID != "" {
		entry.TicketID = &req.TicketID
	}
	if req.TechnicianID != "" {
		entry.TechnicianID = &req.TechnicianID
	}
	if req.ClientID != "" {
		entry.ClientID = &req.ClientID
	}

	if err := s.repo.CreateEntry(entry); err != nil {
		return nil, err
	}

	// Audit log
	s.repo.LogChange("financial_entry", entry.ID, "create", entry, userID, ip, userAgent)

	return s.repo.GetEntryByID(entry.ID)
}

// GetEntryByID retrieves a financial entry by ID
func (s *FinancialService) GetEntryByID(id string) (*models.FinancialEntry, error) {
	return s.repo.GetEntryByID(id)
}

// UpdateEntry updates an existing financial entry
func (s *FinancialService) UpdateEntry(id string, req models.UpdateFinancialEntryRequest, userID string, ip string, userAgent string) (*models.FinancialEntry, error) {
	// Get existing entry
	existing, err := s.repo.GetEntryByID(id)
	if err != nil {
		return nil, err
	}

	// Validate category if provided
	entryType := existing.Type
	if req.Type != "" {
		entryType = req.Type
	}
	category := existing.Category
	if req.Category != "" {
		category = req.Category
	}
	subcategory := existing.Subcategory
	if req.Subcategory != "" {
		subcategory = req.Subcategory
	}
	if !s.ValidateCategory(entryType, category, subcategory) {
		return nil, errors.New("invalid category or subcategory for the given type")
	}

	// Build updated entry
	updated := *existing
	if req.Type != "" {
		updated.Type = req.Type
	}
	if req.Category != "" {
		updated.Category = req.Category
	}
	if req.Subcategory != "" {
		updated.Subcategory = req.Subcategory
	}
	if req.Description != "" {
		updated.Description = req.Description
	}
	if req.Amount > 0 {
		updated.Amount = req.Amount
	}
	if req.EntryDate != "" {
		entryDate, err := time.Parse("2006-01-02", req.EntryDate)
		if err != nil {
			return nil, errors.New("invalid entry date format")
		}
		updated.EntryDate = entryDate
	}
	if req.DueDate != "" {
		dueDate, err := time.Parse("2006-01-02", req.DueDate)
		if err != nil {
			return nil, errors.New("invalid due date format")
		}
		updated.DueDate = &dueDate
	}
	if req.TicketID != "" {
		updated.TicketID = &req.TicketID
	}
	if req.TechnicianID != "" {
		updated.TechnicianID = &req.TechnicianID
	}
	if req.ClientID != "" {
		updated.ClientID = &req.ClientID
	}
	if req.PaymentMethod != "" {
		updated.PaymentMethod = req.PaymentMethod
	}
	if req.PaymentReference != "" {
		updated.PaymentReference = req.PaymentReference
	}
	if len(req.AttachmentURLs) > 0 {
		updated.AttachmentURLs = req.AttachmentURLs
	}
	updated.UpdatedBy = &userID
	updated.Version = req.Version + 1

	if err := s.repo.UpdateEntry(&updated); err != nil {
		if errors.Is(err, errors.New("record not found")) {
			return nil, errors.New("entry was modified by another user, please refresh and try again")
		}
		return nil, err
	}

	// Audit log
	changes := map[string]interface{}{
		"before":  existing,
		"after":   updated,
		"version": req.Version,
	}
	s.repo.LogChange("financial_entry", id, "update", changes, userID, ip, userAgent)

	return s.repo.GetEntryByID(id)
}

// UpdateEntryStatus updates the status of a financial entry
func (s *FinancialService) UpdateEntryStatus(id string, req models.UpdateFinancialEntryStatusRequest, userID string, ip string, userAgent string) (*models.FinancialEntry, error) {
	existing, err := s.repo.GetEntryByID(id)
	if err != nil {
		return nil, err
	}

	var paymentDate *time.Time
	if req.PaymentDate != "" {
		parsed, err := time.Parse("2006-01-02", req.PaymentDate)
		if err != nil {
			return nil, errors.New("invalid payment date format")
		}
		paymentDate = &parsed
	}

	// If marking as paid, require payment date
	if req.Status == models.FinancialEntryStatusPaid && paymentDate == nil {
		now := time.Now()
		paymentDate = &now
	}

	if err := s.repo.UpdateEntryStatus(id, req.Status, paymentDate, req.PaymentReference, userID); err != nil {
		return nil, err
	}

	// Audit log
	changes := map[string]interface{}{
		"previousStatus": existing.Status,
		"newStatus":      req.Status,
		"paymentDate":    paymentDate,
	}
	s.repo.LogChange("financial_entry", id, "status_change", changes, userID, ip, userAgent)

	return s.repo.GetEntryByID(id)
}

// DeleteEntry soft-deletes a financial entry
func (s *FinancialService) DeleteEntry(id string, userID string, ip string, userAgent string) error {
	existing, err := s.repo.GetEntryByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteEntry(id); err != nil {
		return err
	}

	// Audit log
	s.repo.LogChange("financial_entry", id, "delete", existing, userID, ip, userAgent)

	return nil
}

// ListEntries lists financial entries with filters
func (s *FinancialService) ListEntries(filter models.FinancialEntryFilter) ([]models.FinancialEntry, int64, error) {
	return s.repo.ListEntries(filter)
}

// =============== Payment Batches ===============

// CreateBatch creates a new payment batch
func (s *FinancialService) CreateBatch(req models.CreatePaymentBatchRequest, userID string, ip string, userAgent string) (*models.PaymentBatch, error) {
	periodStart, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		return nil, errors.New("invalid period start date format")
	}
	periodEnd, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		return nil, errors.New("invalid period end date format")
	}

	if periodEnd.Before(periodStart) {
		return nil, errors.New("period end must be after period start")
	}

	batch := &models.PaymentBatch{
		Name:        req.Name,
		Description: req.Description,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Status:      models.PaymentBatchStatusDraft,
		CreatedBy:   userID,
	}

	if err := s.repo.CreateBatch(batch); err != nil {
		return nil, err
	}

	s.repo.LogChange("payment_batch", batch.ID, "create", batch, userID, ip, userAgent)

	return s.repo.GetBatchByID(batch.ID)
}

// GetBatchByID retrieves a payment batch by ID
func (s *FinancialService) GetBatchByID(id string) (*models.PaymentBatch, error) {
	return s.repo.GetBatchByID(id)
}

// ListBatches lists payment batches with filters
func (s *FinancialService) ListBatches(filter models.PaymentBatchFilter) ([]models.PaymentBatch, int64, error) {
	return s.repo.ListBatches(filter)
}

// AddEntriesToBatch adds entries to a payment batch
func (s *FinancialService) AddEntriesToBatch(batchID string, req models.AddBatchEntriesRequest, userID string, ip string, userAgent string) (*models.PaymentBatch, error) {
	batch, err := s.repo.GetBatchByID(batchID)
	if err != nil {
		return nil, err
	}

	if batch.Status != models.PaymentBatchStatusDraft {
		return nil, errors.New("can only add entries to draft batches")
	}

	// Validate entries exist and are expenses
	entries, err := s.repo.GetEntriesByIDs(req.EntryIDs)
	if err != nil {
		return nil, err
	}
	if len(entries) != len(req.EntryIDs) {
		return nil, errors.New("some entries were not found")
	}
	for _, entry := range entries {
		if entry.Type != models.FinancialEntryTypeExpense {
			return nil, errors.New("only expense entries can be added to payment batches")
		}
	}

	if err := s.repo.AddEntriesToBatch(batchID, req.EntryIDs); err != nil {
		return nil, err
	}

	s.repo.LogChange("payment_batch", batchID, "add_entries", req.EntryIDs, userID, ip, userAgent)

	return s.repo.GetBatchByID(batchID)
}

// RemoveEntryFromBatch removes an entry from a payment batch
func (s *FinancialService) RemoveEntryFromBatch(batchID string, entryID string, userID string, ip string, userAgent string) error {
	batch, err := s.repo.GetBatchByID(batchID)
	if err != nil {
		return err
	}

	if batch.Status != models.PaymentBatchStatusDraft {
		return errors.New("can only remove entries from draft batches")
	}

	if err := s.repo.RemoveEntryFromBatch(batchID, entryID); err != nil {
		return err
	}

	s.repo.LogChange("payment_batch", batchID, "remove_entry", entryID, userID, ip, userAgent)

	return nil
}

// ApproveBatch approves a payment batch
func (s *FinancialService) ApproveBatch(batchID string, userID string, ip string, userAgent string) (*models.PaymentBatch, error) {
	batch, err := s.repo.GetBatchByID(batchID)
	if err != nil {
		return nil, err
	}

	if batch.Status != models.PaymentBatchStatusDraft {
		return nil, errors.New("can only approve draft batches")
	}

	if batch.EntriesCount == 0 {
		return nil, errors.New("cannot approve an empty batch")
	}

	if err := s.repo.ApproveBatch(batchID, userID); err != nil {
		return nil, err
	}

	s.repo.LogChange("payment_batch", batchID, "approve", nil, userID, ip, userAgent)

	return s.repo.GetBatchByID(batchID)
}

// PayBatch marks a batch as paid
func (s *FinancialService) PayBatch(batchID string, req models.PayBatchRequest, userID string, ip string, userAgent string) (*models.PaymentBatch, error) {
	batch, err := s.repo.GetBatchByID(batchID)
	if err != nil {
		return nil, err
	}

	if batch.Status != models.PaymentBatchStatusApproved {
		return nil, errors.New("can only pay approved batches")
	}

	if err := s.repo.PayBatch(batchID, req.PaymentReference); err != nil {
		return nil, err
	}

	s.repo.LogChange("payment_batch", batchID, "pay", req, userID, ip, userAgent)

	return s.repo.GetBatchByID(batchID)
}

// DeleteBatch deletes a payment batch
func (s *FinancialService) DeleteBatch(batchID string, userID string, ip string, userAgent string) error {
	batch, err := s.repo.GetBatchByID(batchID)
	if err != nil {
		return err
	}

	if batch.Status != models.PaymentBatchStatusDraft {
		return errors.New("can only delete draft batches")
	}

	if err := s.repo.DeleteBatch(batchID); err != nil {
		return err
	}

	s.repo.LogChange("payment_batch", batchID, "delete", batch, userID, ip, userAgent)

	return nil
}

// =============== Dashboard & Reports ===============

// GetDashboard retrieves financial dashboard data
func (s *FinancialService) GetDashboard(filter models.DashboardFilter) (*models.FinancialDashboard, error) {
	startDate, endDate := s.getPeriodDates(filter.Period, filter.StartDate, filter.EndDate)
	return s.repo.GetDashboardData(startDate, endDate)
}

// GetCashFlowReport retrieves cash flow report
func (s *FinancialService) GetCashFlowReport(filter models.CashFlowFilter) (*models.CashFlowReport, error) {
	startDate, err := time.Parse("2006-01-02", filter.StartDate)
	if err != nil {
		return nil, errors.New("invalid start date format")
	}
	endDate, err := time.Parse("2006-01-02", filter.EndDate)
	if err != nil {
		return nil, errors.New("invalid end date format")
	}

	groupBy := filter.GroupBy
	if groupBy == "" {
		groupBy = "day"
	}

	return s.repo.GetCashFlowReport(startDate, endDate, groupBy)
}

// GetTechnicianPaymentsReport retrieves technician payments report
func (s *FinancialService) GetTechnicianPaymentsReport(filter models.TechnicianPaymentsFilter) (*models.TechnicianPaymentsReport, error) {
	startDate, err := time.Parse("2006-01-02", filter.StartDate)
	if err != nil {
		return nil, errors.New("invalid start date format")
	}
	endDate, err := time.Parse("2006-01-02", filter.EndDate)
	if err != nil {
		return nil, errors.New("invalid end date format")
	}

	return s.repo.GetTechnicianPaymentsReport(startDate, endDate, filter.TechnicianID)
}

// GetCategories returns all available financial categories
func (s *FinancialService) GetCategories() []models.FinancialCategory {
	return models.GetFinancialCategories()
}

// MarkOverdueEntries marks entries past due date as overdue
func (s *FinancialService) MarkOverdueEntries() (int64, error) {
	return s.repo.MarkOverdueEntries()
}

// =============== Helpers ===============

// ValidateCategory validates if the category and subcategory are valid for the type
// Uses database categories instead of hardcoded values
func (s *FinancialService) ValidateCategory(entryType models.FinancialEntryType, category string, subcategory string) bool {
	// Convert FinancialEntryType to CategoryType
	var categoryType models.CategoryType
	if entryType == models.FinancialEntryTypeIncome {
		categoryType = models.CategoryTypeFinanceIncome
	} else {
		categoryType = models.CategoryTypeFinanceExpense
	}

	// Get categories from database
	dbCategories, err := s.categoryRepo.GetByTypeWithChildren(categoryType)
	if err != nil {
		// If error, fallback to allow (or log error)
		return false
	}

	// Find matching category by name
	for _, cat := range dbCategories {
		if cat.Name == category {
			if subcategory == "" {
				return true
			}
			// Check subcategories (children)
			for _, child := range cat.Children {
				if child.Name == subcategory {
					return true
				}
			}
			return false
		}
	}
	return false
}

// getPeriodDates converts a period string to start and end dates
func (s *FinancialService) getPeriodDates(period string, customStart string, customEnd string) (time.Time, time.Time) {
	now := time.Now()
	var startDate, endDate time.Time

	switch period {
	case "today":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 0, 6)
		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, endDate.Location())
	case "month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, -1)
		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, endDate.Location())
	case "quarter":
		quarter := (int(now.Month()) - 1) / 3
		startMonth := time.Month(quarter*3 + 1)
		startDate = time.Date(now.Year(), startMonth, 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 3, -1)
		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, endDate.Location())
	case "year":
		startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), 12, 31, 23, 59, 59, 0, now.Location())
	case "custom":
		startDate, _ = time.Parse("2006-01-02", customStart)
		endDate, _ = time.Parse("2006-01-02", customEnd)
		if endDate.IsZero() {
			endDate = now
		}
	default:
		// Default to current month
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = now
	}

	return startDate, endDate
}
