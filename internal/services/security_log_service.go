package services

import (
	"math"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type SecurityLogService interface {
	Create(log *models.SecurityLog) error
	LogSecurityEvent(userID, email, action, ipAddress, userAgent, location, details string, success bool) error
	GetByID(id string) (*models.SecurityLog, error)
	GetAll(filter *models.SecurityLogFilter, page, limit int) (*models.PaginatedSecurityLogs, error)
	GetRecentLogs(limit int) ([]models.SecurityLog, error)
	GetTodayStats() (successful int64, failed int64, err error)
}

type securityLogService struct {
	repo repositories.SecurityLogRepository
}

func NewSecurityLogService(repo repositories.SecurityLogRepository) SecurityLogService {
	return &securityLogService{repo: repo}
}

func (s *securityLogService) Create(log *models.SecurityLog) error {
	return s.repo.Create(log)
}

func (s *securityLogService) LogSecurityEvent(userID, email, action, ipAddress, userAgent, location, details string, success bool) error {
	log := &models.SecurityLog{
		UserID:    userID,
		Email:     email,
		Action:    action,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Location:  location,
		Details:   details,
		Success:   success,
		CreatedAt: time.Now(),
	}
	return s.repo.Create(log)
}

func (s *securityLogService) GetByID(id string) (*models.SecurityLog, error) {
	return s.repo.FindByID(id)
}

func (s *securityLogService) GetAll(filter *models.SecurityLogFilter, page, limit int) (*models.PaginatedSecurityLogs, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	logs, total, err := s.repo.FindAll(filter, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &models.PaginatedSecurityLogs{
		Content:       logs,
		Page:          page,
		Size:          limit,
		TotalElements: total,
		TotalPages:    totalPages,
	}, nil
}

func (s *securityLogService) GetRecentLogs(limit int) ([]models.SecurityLog, error) {
	return s.repo.GetRecentLogs(limit)
}

func (s *securityLogService) GetTodayStats() (successful int64, failed int64, err error) {
	today := time.Now().Truncate(24 * time.Hour)
	
	successful, err = s.repo.CountSuccessful(today)
	if err != nil {
		return 0, 0, err
	}
	
	failed, err = s.repo.CountFailed(today)
	if err != nil {
		return 0, 0, err
	}
	
	return successful, failed, nil
}
