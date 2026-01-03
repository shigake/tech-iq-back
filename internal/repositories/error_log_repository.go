package repositories

import (
	"strings"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"

	"gorm.io/gorm"
)

type ErrorLogRepository struct {
	db *gorm.DB
}

func NewErrorLogRepository(db *gorm.DB) *ErrorLogRepository {
	return &ErrorLogRepository{db: db}
}

// Create creates a new error log entry
func (r *ErrorLogRepository) Create(log *models.ErrorLog) error {
	return r.db.Create(log).Error
}

// GetByID returns an error log by ID
func (r *ErrorLogRepository) GetByID(id string) (*models.ErrorLog, error) {
	var log models.ErrorLog
	err := r.db.First(&log, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetAll returns all error logs with pagination and filtering
func (r *ErrorLogRepository) GetAll(page, pageSize int, filter *models.ErrorLogFilter) (*models.PaginatedErrorLogs, error) {
	var logs []models.ErrorLog
	var total int64

	query := r.db.Model(&models.ErrorLog{})

	if filter != nil {
		if filter.Level != "" {
			query = query.Where("level = ?", filter.Level)
		}
		if filter.Feature != "" {
			query = query.Where("feature ILIKE ?", "%"+filter.Feature+"%")
		}
		if filter.Endpoint != "" {
			query = query.Where("endpoint ILIKE ?", "%"+filter.Endpoint+"%")
		}
		if filter.ErrorCode != "" {
			query = query.Where("error_code = ?", filter.ErrorCode)
		}
		if filter.UserID != "" {
			query = query.Where("user_id = ?", filter.UserID)
		}
		if filter.StatusCode > 0 {
			query = query.Where("status_code = ?", filter.StatusCode)
		}
		if filter.Resolved != nil {
			query = query.Where("resolved = ?", *filter.Resolved)
		}
		if !filter.StartDate.IsZero() {
			query = query.Where("timestamp >= ?", filter.StartDate)
		}
		if !filter.EndDate.IsZero() {
			query = query.Where("timestamp <= ?", filter.EndDate)
		}
		if filter.Search != "" {
			searchTerm := "%" + strings.ToLower(filter.Search) + "%"
			query = query.Where(
				"LOWER(error_message) LIKE ? OR LOWER(feature) LIKE ? OR LOWER(endpoint) LIKE ?",
				searchTerm, searchTerm, searchTerm,
			)
		}
	}

	// Count total
	query.Count(&total)

	// Get paginated results
	offset := page * pageSize
	err := query.Order("timestamp DESC").Offset(offset).Limit(pageSize).Find(&logs).Error
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &models.PaginatedErrorLogs{
		Data:       logs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetStats returns statistics about error logs
func (r *ErrorLogRepository) GetStats() (*models.ErrorLogStats, error) {
	stats := &models.ErrorLogStats{
		ErrorsByLevel:    make(map[string]int64),
		ErrorsByFeature:  make(map[string]int64),
		ErrorsByEndpoint: make(map[string]int64),
	}

	// Total errors
	r.db.Model(&models.ErrorLog{}).Count(&stats.TotalErrors)

	// Unresolved errors
	r.db.Model(&models.ErrorLog{}).Where("resolved = ?", false).Count(&stats.UnresolvedErrors)

	// Errors today
	today := time.Now().Truncate(24 * time.Hour)
	r.db.Model(&models.ErrorLog{}).Where("timestamp >= ?", today).Count(&stats.ErrorsToday)

	// Errors this week
	weekAgo := time.Now().AddDate(0, 0, -7)
	r.db.Model(&models.ErrorLog{}).Where("timestamp >= ?", weekAgo).Count(&stats.ErrorsThisWeek)

	// Errors by level
	var levelCounts []struct {
		Level string
		Count int64
	}
	r.db.Model(&models.ErrorLog{}).
		Select("level, count(*) as count").
		Group("level").
		Scan(&levelCounts)
	for _, lc := range levelCounts {
		stats.ErrorsByLevel[lc.Level] = lc.Count
	}

	// Errors by feature (top 10)
	var featureCounts []struct {
		Feature string
		Count   int64
	}
	r.db.Model(&models.ErrorLog{}).
		Select("feature, count(*) as count").
		Where("feature != ''").
		Group("feature").
		Order("count DESC").
		Limit(10).
		Scan(&featureCounts)
	for _, fc := range featureCounts {
		stats.ErrorsByFeature[fc.Feature] = fc.Count
	}

	// Errors by endpoint (top 10)
	var endpointCounts []struct {
		Endpoint string
		Count    int64
	}
	r.db.Model(&models.ErrorLog{}).
		Select("endpoint, count(*) as count").
		Where("endpoint != ''").
		Group("endpoint").
		Order("count DESC").
		Limit(10).
		Scan(&endpointCounts)
	for _, ec := range endpointCounts {
		stats.ErrorsByEndpoint[ec.Endpoint] = ec.Count
	}

	return stats, nil
}

// Resolve marks an error as resolved
func (r *ErrorLogRepository) Resolve(id string, resolvedBy string, notes string) error {
	now := time.Now()
	return r.db.Model(&models.ErrorLog{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"resolved":    true,
			"resolved_at": now,
			"resolved_by": resolvedBy,
			"notes":       notes,
		}).Error
}

// BulkResolve marks multiple errors as resolved
func (r *ErrorLogRepository) BulkResolve(ids []string, resolvedBy string, notes string) error {
	now := time.Now()
	return r.db.Model(&models.ErrorLog{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"resolved":    true,
			"resolved_at": now,
			"resolved_by": resolvedBy,
			"notes":       notes,
		}).Error
}

// Delete deletes an error log
func (r *ErrorLogRepository) Delete(id string) error {
	return r.db.Delete(&models.ErrorLog{}, "id = ?", id).Error
}

// DeleteOld deletes error logs older than the specified duration
func (r *ErrorLogRepository) DeleteOld(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := r.db.Where("timestamp < ? AND resolved = ?", cutoff, true).Delete(&models.ErrorLog{})
	return result.RowsAffected, result.Error
}

// GetRecentByFeature returns recent errors for a specific feature
func (r *ErrorLogRepository) GetRecentByFeature(feature string, limit int) ([]models.ErrorLog, error) {
	var logs []models.ErrorLog
	err := r.db.Where("feature = ?", feature).
		Order("timestamp DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
