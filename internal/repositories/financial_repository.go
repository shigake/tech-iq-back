package repositories

import (
	"encoding/json"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
)

type FinancialRepository struct {
	db *gorm.DB
}

func NewFinancialRepository(db *gorm.DB) *FinancialRepository {
	return &FinancialRepository{db: db}
}

// =============== Financial Entries ===============

// CreateEntry creates a new financial entry
func (r *FinancialRepository) CreateEntry(entry *models.FinancialEntry) error {
	return r.db.Create(entry).Error
}

// GetEntryByID retrieves a financial entry by ID
func (r *FinancialRepository) GetEntryByID(id string) (*models.FinancialEntry, error) {
	var entry models.FinancialEntry
	err := r.db.Preload("Ticket").
		Preload("Technician").
		Preload("Client").
		Preload("CreatedByUser").
		First(&entry, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// UpdateEntry updates an existing financial entry with optimistic locking
func (r *FinancialRepository) UpdateEntry(entry *models.FinancialEntry) error {
	result := r.db.Model(entry).
		Where("id = ? AND version = ?", entry.ID, entry.Version-1).
		Updates(map[string]interface{}{
			"type":              entry.Type,
			"category":          entry.Category,
			"subcategory":       entry.Subcategory,
			"description":       entry.Description,
			"amount":            entry.Amount,
			"entry_date":        entry.EntryDate,
			"due_date":          entry.DueDate,
			"payment_date":      entry.PaymentDate,
			"status":            entry.Status,
			"ticket_id":         entry.TicketID,
			"technician_id":     entry.TechnicianID,
			"client_id":         entry.ClientID,
			"payment_method":    entry.PaymentMethod,
			"payment_reference": entry.PaymentReference,
			"attachment_urls":   entry.AttachmentURLs,
			"updated_by":        entry.UpdatedBy,
			"updated_at":        time.Now(),
			"version":           entry.Version,
		})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound // Will be interpreted as conflict
	}
	return result.Error
}

// UpdateEntryStatus updates only the status of a financial entry
func (r *FinancialRepository) UpdateEntryStatus(id string, status models.FinancialEntryStatus, paymentDate *time.Time, paymentReference string, updatedBy string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_by": updatedBy,
		"updated_at": time.Now(),
	}
	if paymentDate != nil {
		updates["payment_date"] = paymentDate
	}
	if paymentReference != "" {
		updates["payment_reference"] = paymentReference
	}
	return r.db.Model(&models.FinancialEntry{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteEntry soft-deletes a financial entry
func (r *FinancialRepository) DeleteEntry(id string) error {
	return r.db.Delete(&models.FinancialEntry{}, "id = ?", id).Error
}

// ListEntries retrieves financial entries with filters
func (r *FinancialRepository) ListEntries(filter models.FinancialEntryFilter) ([]models.FinancialEntry, int64, error) {
	var entries []models.FinancialEntry
	var total int64

	query := r.db.Model(&models.FinancialEntry{})

	// Apply filters
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.StartDate != "" {
		startDate, _ := time.Parse("2006-01-02", filter.StartDate)
		query = query.Where("entry_date >= ?", startDate)
	}
	if filter.EndDate != "" {
		endDate, _ := time.Parse("2006-01-02", filter.EndDate)
		query = query.Where("entry_date <= ?", endDate)
	}
	if filter.TechnicianID != "" {
		query = query.Where("technician_id = ?", filter.TechnicianID)
	}
	if filter.ClientID != "" {
		query = query.Where("client_id = ?", filter.ClientID)
	}
	if filter.TicketID != "" {
		query = query.Where("ticket_id = ?", filter.TicketID)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Pagination
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	// Fetch entries
	err := query.
		Preload("Technician").
		Preload("Client").
		Order("entry_date DESC, created_at DESC").
		Offset(offset).
		Limit(filter.Limit).
		Find(&entries).Error

	return entries, total, err
}

// GetEntriesByIDs retrieves multiple entries by their IDs
func (r *FinancialRepository) GetEntriesByIDs(ids []string) ([]models.FinancialEntry, error) {
	var entries []models.FinancialEntry
	err := r.db.Where("id IN ?", ids).Find(&entries).Error
	return entries, err
}

// UpdateEntriesStatus updates status for multiple entries
func (r *FinancialRepository) UpdateEntriesStatus(ids []string, status models.FinancialEntryStatus, paymentDate *time.Time) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if paymentDate != nil {
		updates["payment_date"] = paymentDate
	}
	return r.db.Model(&models.FinancialEntry{}).Where("id IN ?", ids).Updates(updates).Error
}

// MarkOverdueEntries updates entries past due date to overdue status
func (r *FinancialRepository) MarkOverdueEntries() (int64, error) {
	result := r.db.Model(&models.FinancialEntry{}).
		Where("status = ? AND due_date < ? AND due_date IS NOT NULL", models.FinancialEntryStatusPending, time.Now()).
		Update("status", models.FinancialEntryStatusOverdue)
	return result.RowsAffected, result.Error
}

// =============== Payment Batches ===============

// CreateBatch creates a new payment batch
func (r *FinancialRepository) CreateBatch(batch *models.PaymentBatch) error {
	return r.db.Create(batch).Error
}

// GetBatchByID retrieves a payment batch by ID
func (r *FinancialRepository) GetBatchByID(id string) (*models.PaymentBatch, error) {
	var batch models.PaymentBatch
	err := r.db.Preload("Entries").
		Preload("Entries.Technician").
		Preload("CreatedByUser").
		Preload("ApprovedByUser").
		First(&batch, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

// UpdateBatch updates a payment batch
func (r *FinancialRepository) UpdateBatch(batch *models.PaymentBatch) error {
	return r.db.Save(batch).Error
}

// DeleteBatch soft-deletes a payment batch
func (r *FinancialRepository) DeleteBatch(id string) error {
	return r.db.Delete(&models.PaymentBatch{}, "id = ?", id).Error
}

// ListBatches retrieves payment batches with filters
func (r *FinancialRepository) ListBatches(filter models.PaymentBatchFilter) ([]models.PaymentBatch, int64, error) {
	var batches []models.PaymentBatch
	var total int64

	query := r.db.Model(&models.PaymentBatch{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.PeriodStart != "" {
		startDate, _ := time.Parse("2006-01-02", filter.PeriodStart)
		query = query.Where("period_end >= ?", startDate)
	}
	if filter.PeriodEnd != "" {
		endDate, _ := time.Parse("2006-01-02", filter.PeriodEnd)
		query = query.Where("period_start <= ?", endDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	err := query.
		Preload("CreatedByUser").
		Order("created_at DESC").
		Offset(offset).
		Limit(filter.Limit).
		Find(&batches).Error

	return batches, total, err
}

// AddEntriesToBatch adds entries to a batch and updates totals
func (r *FinancialRepository) AddEntriesToBatch(batchID string, entryIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Get the batch
		var batch models.PaymentBatch
		if err := tx.First(&batch, "id = ?", batchID).Error; err != nil {
			return err
		}

		// Get entries
		var entries []models.FinancialEntry
		if err := tx.Where("id IN ?", entryIDs).Find(&entries).Error; err != nil {
			return err
		}

		// Add entries to batch
		if err := tx.Model(&batch).Association("Entries").Append(entries); err != nil {
			return err
		}

		// Update totals
		var totalAmount float64
		var count int64
		tx.Model(&models.FinancialEntry{}).
			Joins("JOIN payment_batch_entries ON payment_batch_entries.entry_id = financial_entries.id").
			Where("payment_batch_entries.batch_id = ?", batchID).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&totalAmount)
		tx.Model(&models.FinancialEntry{}).
			Joins("JOIN payment_batch_entries ON payment_batch_entries.entry_id = financial_entries.id").
			Where("payment_batch_entries.batch_id = ?", batchID).
			Count(&count)

		return tx.Model(&batch).Updates(map[string]interface{}{
			"total_amount":  totalAmount,
			"entries_count": count,
		}).Error
	})
}

// RemoveEntryFromBatch removes an entry from a batch
func (r *FinancialRepository) RemoveEntryFromBatch(batchID string, entryID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Remove the association
		if err := tx.Exec("DELETE FROM payment_batch_entries WHERE batch_id = ? AND entry_id = ?", batchID, entryID).Error; err != nil {
			return err
		}

		// Update totals
		var batch models.PaymentBatch
		if err := tx.First(&batch, "id = ?", batchID).Error; err != nil {
			return err
		}

		var totalAmount float64
		var count int64
		tx.Model(&models.FinancialEntry{}).
			Joins("JOIN payment_batch_entries ON payment_batch_entries.entry_id = financial_entries.id").
			Where("payment_batch_entries.batch_id = ?", batchID).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&totalAmount)
		tx.Model(&models.FinancialEntry{}).
			Joins("JOIN payment_batch_entries ON payment_batch_entries.entry_id = financial_entries.id").
			Where("payment_batch_entries.batch_id = ?", batchID).
			Count(&count)

		return tx.Model(&batch).Updates(map[string]interface{}{
			"total_amount":  totalAmount,
			"entries_count": count,
		}).Error
	})
}

// ApproveBatch approves a payment batch
func (r *FinancialRepository) ApproveBatch(batchID string, approvedBy string) error {
	now := time.Now()
	return r.db.Model(&models.PaymentBatch{}).
		Where("id = ? AND status = ?", batchID, models.PaymentBatchStatusDraft).
		Updates(map[string]interface{}{
			"status":      models.PaymentBatchStatusApproved,
			"approved_by": approvedBy,
			"approved_at": now,
		}).Error
}

// PayBatch marks a batch as paid and updates all entries
func (r *FinancialRepository) PayBatch(batchID string, paymentReference string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// Update batch status
		if err := tx.Model(&models.PaymentBatch{}).
			Where("id = ? AND status = ?", batchID, models.PaymentBatchStatusApproved).
			Updates(map[string]interface{}{
				"status":            models.PaymentBatchStatusPaid,
				"paid_at":           now,
				"payment_reference": paymentReference,
			}).Error; err != nil {
			return err
		}

		// Update all entries in the batch to paid
		return tx.Exec(`
			UPDATE financial_entries 
			SET status = ?, payment_date = ?, updated_at = ?
			WHERE id IN (
				SELECT entry_id FROM payment_batch_entries WHERE batch_id = ?
			)
		`, models.FinancialEntryStatusPaid, now, now, batchID).Error
	})
}

// =============== Dashboard & Reports ===============

// GetDashboardData retrieves dashboard statistics
func (r *FinancialRepository) GetDashboardData(startDate, endDate time.Time) (*models.FinancialDashboard, error) {
	dashboard := &models.FinancialDashboard{
		ByCategory: struct {
			Income  map[string]float64 `json:"income"`
			Expense map[string]float64 `json:"expense"`
		}{
			Income:  make(map[string]float64),
			Expense: make(map[string]float64),
		},
	}

	// Total income
	r.db.Model(&models.FinancialEntry{}).
		Where("type = ? AND entry_date BETWEEN ? AND ?", models.FinancialEntryTypeIncome, startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&dashboard.Summary.TotalIncome)

	// Total expense
	r.db.Model(&models.FinancialEntry{}).
		Where("type = ? AND entry_date BETWEEN ? AND ?", models.FinancialEntryTypeExpense, startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&dashboard.Summary.TotalExpense)

	dashboard.Summary.Balance = dashboard.Summary.TotalIncome - dashboard.Summary.TotalExpense

	// Income by category
	var incomeByCategory []struct {
		Category string
		Total    float64
	}
	r.db.Model(&models.FinancialEntry{}).
		Where("type = ? AND entry_date BETWEEN ? AND ?", models.FinancialEntryTypeIncome, startDate, endDate).
		Select("category, COALESCE(SUM(amount), 0) as total").
		Group("category").
		Scan(&incomeByCategory)
	for _, item := range incomeByCategory {
		dashboard.ByCategory.Income[item.Category] = item.Total
	}

	// Expense by category
	var expenseByCategory []struct {
		Category string
		Total    float64
	}
	r.db.Model(&models.FinancialEntry{}).
		Where("type = ? AND entry_date BETWEEN ? AND ?", models.FinancialEntryTypeExpense, startDate, endDate).
		Select("category, COALESCE(SUM(amount), 0) as total").
		Group("category").
		Scan(&expenseByCategory)
	for _, item := range expenseByCategory {
		dashboard.ByCategory.Expense[item.Category] = item.Total
	}

	// Pending payments count
	r.db.Model(&models.FinancialEntry{}).
		Where("status = ?", models.FinancialEntryStatusPending).
		Count(&dashboard.PendingPayments)

	// Overdue count
	r.db.Model(&models.FinancialEntry{}).
		Where("status = ?", models.FinancialEntryStatusOverdue).
		Count(&dashboard.OverdueCount)

	// Recent entries
	r.db.Preload("Technician").
		Preload("Client").
		Where("entry_date BETWEEN ? AND ?", startDate, endDate).
		Order("created_at DESC").
		Limit(10).
		Find(&dashboard.RecentEntries)

	return dashboard, nil
}

// GetCashFlowReport generates cash flow report grouped by period
func (r *FinancialRepository) GetCashFlowReport(startDate, endDate time.Time, groupBy string) (*models.CashFlowReport, error) {
	report := &models.CashFlowReport{
		Periods: []models.CashFlowPeriod{},
	}

	var dateFormat string
	switch groupBy {
	case "day":
		dateFormat = "YYYY-MM-DD"
	case "week":
		dateFormat = "IYYY-IW"
	case "month":
		dateFormat = "YYYY-MM"
	default:
		dateFormat = "YYYY-MM-DD"
	}

	// Query for income grouped by period
	var incomeData []struct {
		Period string
		Total  float64
	}
	r.db.Model(&models.FinancialEntry{}).
		Where("type = ? AND entry_date BETWEEN ? AND ?", models.FinancialEntryTypeIncome, startDate, endDate).
		Select("TO_CHAR(entry_date, ?) as period, COALESCE(SUM(amount), 0) as total", dateFormat).
		Group("period").
		Order("period").
		Scan(&incomeData)

	// Query for expense grouped by period
	var expenseData []struct {
		Period string
		Total  float64
	}
	r.db.Model(&models.FinancialEntry{}).
		Where("type = ? AND entry_date BETWEEN ? AND ?", models.FinancialEntryTypeExpense, startDate, endDate).
		Select("TO_CHAR(entry_date, ?) as period, COALESCE(SUM(amount), 0) as total", dateFormat).
		Group("period").
		Order("period").
		Scan(&expenseData)

	// Combine data
	periodMap := make(map[string]*models.CashFlowPeriod)
	for _, item := range incomeData {
		periodMap[item.Period] = &models.CashFlowPeriod{
			Period: item.Period,
			Income: item.Total,
		}
	}
	for _, item := range expenseData {
		if p, exists := periodMap[item.Period]; exists {
			p.Expense = item.Total
			p.Balance = p.Income - p.Expense
		} else {
			periodMap[item.Period] = &models.CashFlowPeriod{
				Period:  item.Period,
				Expense: item.Total,
				Balance: -item.Total,
			}
		}
	}

	for _, p := range periodMap {
		report.Periods = append(report.Periods, *p)
	}

	return report, nil
}

// GetTechnicianPaymentsReport generates technician payments report
func (r *FinancialRepository) GetTechnicianPaymentsReport(startDate, endDate time.Time, technicianID string) (*models.TechnicianPaymentsReport, error) {
	report := &models.TechnicianPaymentsReport{
		Technicians: []models.TechnicianPaymentReport{},
	}

	query := r.db.Model(&models.FinancialEntry{}).
		Where("category = ? AND entry_date BETWEEN ? AND ? AND technician_id IS NOT NULL",
			"technician_payment", startDate, endDate)

	if technicianID != "" {
		query = query.Where("technician_id = ?", technicianID)
	}

	var data []struct {
		TechnicianID string
		Total        float64
		Count        int
	}
	query.Select("technician_id, COALESCE(SUM(amount), 0) as total, COUNT(*) as count").
		Group("technician_id").
		Scan(&data)

	// Get technician names
	for _, item := range data {
		var tech models.Technician
		r.db.Select("full_name").First(&tech, "id = ?", item.TechnicianID)

		report.Technicians = append(report.Technicians, models.TechnicianPaymentReport{
			TechnicianID:   item.TechnicianID,
			TechnicianName: tech.FullName,
			Total:          item.Total,
			EntriesCount:   item.Count,
		})
	}

	return report, nil
}

// =============== Audit Logs ===============

// CreateAuditLog creates a new audit log entry
func (r *FinancialRepository) CreateAuditLog(log *models.FinancialAuditLog) error {
	return r.db.Create(log).Error
}

// LogChange creates an audit log for a financial change
func (r *FinancialRepository) LogChange(entityType string, entityID string, action string, changes interface{}, userID string, ip string, userAgent string) error {
	changesJSON, _ := json.Marshal(changes)
	log := &models.FinancialAuditLog{
		EntityType:  entityType,
		EntityID:    entityID,
		Action:      action,
		Changes:     string(changesJSON),
		PerformedBy: userID,
		PerformedAt: time.Now(),
		IPAddress:   ip,
		UserAgent:   userAgent,
	}
	return r.CreateAuditLog(log)
}

// GetAuditLogs retrieves audit logs for an entity
func (r *FinancialRepository) GetAuditLogs(entityType string, entityID string) ([]models.FinancialAuditLog, error) {
	var logs []models.FinancialAuditLog
	err := r.db.
		Preload("PerformedByUser").
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("performed_at DESC").
		Find(&logs).Error
	return logs, err
}
