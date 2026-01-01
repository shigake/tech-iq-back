package repositories

import (
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
)

type SecurityLogRepository interface {
	Create(log *models.SecurityLog) error
	FindByID(id string) (*models.SecurityLog, error)
	FindAll(filter *models.SecurityLogFilter, page, limit int) ([]models.SecurityLog, int64, error)
	Delete(id string) error
	DeleteOlderThan(date time.Time) (int64, error)
	CountByAction(action string, since time.Time) (int64, error)
	CountSuccessful(since time.Time) (int64, error)
	CountFailed(since time.Time) (int64, error)
	GetRecentLogs(limit int) ([]models.SecurityLog, error)
}

type securityLogRepository struct {
	db *gorm.DB
}

func NewSecurityLogRepository(db *gorm.DB) SecurityLogRepository {
	return &securityLogRepository{db: db}
}

func (r *securityLogRepository) Create(log *models.SecurityLog) error {
	return r.db.Create(log).Error
}

func (r *securityLogRepository) FindByID(id string) (*models.SecurityLog, error) {
	var log models.SecurityLog
	err := r.db.Preload("User").Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *securityLogRepository) FindAll(filter *models.SecurityLogFilter, page, limit int) ([]models.SecurityLog, int64, error) {
	var logs []models.SecurityLog
	var total int64

	offset := (page - 1) * limit
	query := r.db.Model(&models.SecurityLog{})

	// Apply filters
	if filter != nil {
		if filter.UserID != "" {
			query = query.Where("user_id = ?", filter.UserID)
		}
		if filter.Email != "" {
			query = query.Where("email ILIKE ?", "%"+filter.Email+"%")
		}
		if filter.Action != "" {
			query = query.Where("action = ?", filter.Action)
		}
		if filter.Success != nil {
			query = query.Where("success = ?", *filter.Success)
		}
		if filter.IPAddress != "" {
			query = query.Where("ip_address = ?", filter.IPAddress)
		}
		if !filter.StartDate.IsZero() {
			query = query.Where("created_at >= ?", filter.StartDate)
		}
		if !filter.EndDate.IsZero() {
			query = query.Where("created_at <= ?", filter.EndDate)
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch paginated data
	err := r.db.Model(&models.SecurityLog{}).
		Preload("User").
		Scopes(func(db *gorm.DB) *gorm.DB {
			// Re-apply filters for actual query
			if filter != nil {
				if filter.UserID != "" {
					db = db.Where("user_id = ?", filter.UserID)
				}
				if filter.Email != "" {
					db = db.Where("email ILIKE ?", "%"+filter.Email+"%")
				}
				if filter.Action != "" {
					db = db.Where("action = ?", filter.Action)
				}
				if filter.Success != nil {
					db = db.Where("success = ?", *filter.Success)
				}
				if filter.IPAddress != "" {
					db = db.Where("ip_address = ?", filter.IPAddress)
				}
				if !filter.StartDate.IsZero() {
					db = db.Where("created_at >= ?", filter.StartDate)
				}
				if !filter.EndDate.IsZero() {
					db = db.Where("created_at <= ?", filter.EndDate)
				}
			}
			return db
		}).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error

	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *securityLogRepository) Delete(id string) error {
	return r.db.Delete(&models.SecurityLog{}, "id = ?", id).Error
}

func (r *securityLogRepository) DeleteOlderThan(date time.Time) (int64, error) {
	result := r.db.Delete(&models.SecurityLog{}, "created_at < ?", date)
	return result.RowsAffected, result.Error
}

func (r *securityLogRepository) CountByAction(action string, since time.Time) (int64, error) {
	var count int64
	query := r.db.Model(&models.SecurityLog{}).Where("action = ?", action)
	if !since.IsZero() {
		query = query.Where("created_at >= ?", since)
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *securityLogRepository) CountSuccessful(since time.Time) (int64, error) {
	var count int64
	query := r.db.Model(&models.SecurityLog{}).Where("success = ?", true)
	if !since.IsZero() {
		query = query.Where("created_at >= ?", since)
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *securityLogRepository) CountFailed(since time.Time) (int64, error) {
	var count int64
	query := r.db.Model(&models.SecurityLog{}).Where("success = ?", false)
	if !since.IsZero() {
		query = query.Where("created_at >= ?", since)
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *securityLogRepository) GetRecentLogs(limit int) ([]models.SecurityLog, error) {
	var logs []models.SecurityLog
	err := r.db.Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
