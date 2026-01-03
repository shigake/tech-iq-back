package services

import (
	"time"

	"tech-erp/internal/models"
	"tech-erp/internal/repositories"
)

type ErrorLogService struct {
	repo *repositories.ErrorLogRepository
}

func NewErrorLogService(repo *repositories.ErrorLogRepository) *ErrorLogService {
	return &ErrorLogService{repo: repo}
}

// LogError creates a new error log entry
func (s *ErrorLogService) LogError(log *models.ErrorLog) error {
	if log.Level == "" {
		log.Level = "ERROR"
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}
	
	// Set feature name based on endpoint if not set
	if log.Feature == "" && log.Endpoint != "" {
		log.Feature = models.GetFeatureName(log.Method, log.Endpoint)
	}
	
	return s.repo.Create(log)
}

// GetByID returns an error log by ID
func (s *ErrorLogService) GetByID(id string) (*models.ErrorLog, error) {
	return s.repo.GetByID(id)
}

// GetAll returns all error logs with pagination
func (s *ErrorLogService) GetAll(page, pageSize int, filter *models.ErrorLogFilter) (*models.PaginatedErrorLogs, error) {
	return s.repo.GetAll(page, pageSize, filter)
}

// GetStats returns statistics about error logs
func (s *ErrorLogService) GetStats() (*models.ErrorLogStats, error) {
	return s.repo.GetStats()
}

// Resolve marks an error as resolved
func (s *ErrorLogService) Resolve(id string, resolvedBy string, notes string) error {
	return s.repo.Resolve(id, resolvedBy, notes)
}

// BulkResolve marks multiple errors as resolved
func (s *ErrorLogService) BulkResolve(ids []string, resolvedBy string, notes string) error {
	return s.repo.BulkResolve(ids, resolvedBy, notes)
}

// Delete deletes an error log
func (s *ErrorLogService) Delete(id string) error {
	return s.repo.Delete(id)
}

// CleanupOldLogs deletes resolved error logs older than 30 days
func (s *ErrorLogService) CleanupOldLogs() (int64, error) {
	return s.repo.DeleteOld(30 * 24 * time.Hour)
}

// GetRecentByFeature returns recent errors for a specific feature
func (s *ErrorLogService) GetRecentByFeature(feature string, limit int) ([]models.ErrorLog, error) {
	return s.repo.GetRecentByFeature(feature, limit)
}
