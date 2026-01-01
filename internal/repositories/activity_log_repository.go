package repositories

import (
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
)

type ActivityLogRepository interface {
	Create(log *models.ActivityLog) error
	FindByID(id string) (*models.ActivityLog, error)
	FindByUserID(userID string, page, limit int) ([]models.ActivityLog, int64, error)
	FindAll(filter *models.ActivityLogFilter, page, limit int) ([]models.ActivityLog, int64, error)
	Delete(id string) error
	DeleteOlderThan(date time.Time) (int64, error)
	CountByAction(action string) (int64, error)
	GetRecentLogs(limit int) ([]models.ActivityLog, error)
}

type activityLogRepository struct {
	db *gorm.DB
}

func NewActivityLogRepository(db *gorm.DB) ActivityLogRepository {
	return &activityLogRepository{db: db}
}

func (r *activityLogRepository) Create(log *models.ActivityLog) error {
	return r.db.Create(log).Error
}

func (r *activityLogRepository) FindByID(id string) (*models.ActivityLog, error) {
	var log models.ActivityLog
	err := r.db.Preload("User").Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *activityLogRepository) FindByUserID(userID string, page, limit int) ([]models.ActivityLog, int64, error) {
	var logs []models.ActivityLog
	var total int64

	offset := (page - 1) * limit

	// Count total
	if err := r.db.Model(&models.ActivityLog{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch paginated data
	err := r.db.Preload("User").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error

	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *activityLogRepository) FindAll(filter *models.ActivityLogFilter, page, limit int) ([]models.ActivityLog, int64, error) {
	var logs []models.ActivityLog
	var total int64

	offset := (page - 1) * limit
	query := r.db.Model(&models.ActivityLog{})

	// Apply filters
	if filter != nil {
		if filter.UserID != "" {
			query = query.Where("user_id = ?", filter.UserID)
		}
		if filter.Action != "" {
			query = query.Where("action = ?", filter.Action)
		}
		if filter.Resource != "" {
			query = query.Where("resource = ?", filter.Resource)
		}
		if filter.ResourceID != "" {
			query = query.Where("resource_id = ?", filter.ResourceID)
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
	err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error

	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *activityLogRepository) Delete(id string) error {
	return r.db.Delete(&models.ActivityLog{}, "id = ?", id).Error
}

func (r *activityLogRepository) DeleteOlderThan(date time.Time) (int64, error) {
	result := r.db.Where("created_at < ?", date).Delete(&models.ActivityLog{})
	return result.RowsAffected, result.Error
}

func (r *activityLogRepository) CountByAction(action string) (int64, error) {
	var count int64
	err := r.db.Model(&models.ActivityLog{}).Where("action = ?", action).Count(&count).Error
	return count, err
}

func (r *activityLogRepository) GetRecentLogs(limit int) ([]models.ActivityLog, error) {
	var logs []models.ActivityLog
	err := r.db.Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
