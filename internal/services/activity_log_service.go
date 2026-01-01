package services

import (
	"math"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type ActivityLogService interface {
	Create(log *models.ActivityLog) error
	CreateFromRequest(userID string, req *models.CreateActivityLogRequest) (*models.ActivityLog, error)
	GetByID(id string) (*models.ActivityLog, error)
	GetByUserID(userID string, page, limit int) (*models.PaginatedActivityLogs, error)
	GetAll(filter *models.ActivityLogFilter, page, limit int) (*models.PaginatedActivityLogs, error)
	GetRecentLogs(limit int) ([]models.ActivityLog, error)
	LogAction(userID, action, resource, resourceID, description, ipAddress, userAgent string) error
}

type activityLogService struct {
	repo repositories.ActivityLogRepository
}

func NewActivityLogService(repo repositories.ActivityLogRepository) ActivityLogService {
	return &activityLogService{repo: repo}
}

func (s *activityLogService) Create(log *models.ActivityLog) error {
	return s.repo.Create(log)
}

func (s *activityLogService) CreateFromRequest(userID string, req *models.CreateActivityLogRequest) (*models.ActivityLog, error) {
	log := &models.ActivityLog{
		UserID:      userID,
		Action:      req.Action,
		Resource:    req.Resource,
		ResourceID:  req.ResourceID,
		Description: req.Description,
		IPAddress:   req.IPAddress,
		UserAgent:   req.UserAgent,
		Metadata:    req.Metadata,
	}

	if err := s.repo.Create(log); err != nil {
		return nil, err
	}

	return log, nil
}

func (s *activityLogService) GetByID(id string) (*models.ActivityLog, error) {
	return s.repo.FindByID(id)
}

func (s *activityLogService) GetByUserID(userID string, page, limit int) (*models.PaginatedActivityLogs, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	logs, total, err := s.repo.FindByUserID(userID, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &models.PaginatedActivityLogs{
		Data:       logs,
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	}, nil
}

func (s *activityLogService) GetAll(filter *models.ActivityLogFilter, page, limit int) (*models.PaginatedActivityLogs, error) {
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

	return &models.PaginatedActivityLogs{
		Data:       logs,
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	}, nil
}

func (s *activityLogService) GetRecentLogs(limit int) ([]models.ActivityLog, error) {
	if limit < 1 {
		limit = 10
	}
	return s.repo.GetRecentLogs(limit)
}

func (s *activityLogService) LogAction(userID, action, resource, resourceID, description, ipAddress, userAgent string) error {
	log := &models.ActivityLog{
		UserID:      userID,
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		Description: description,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	}
	return s.repo.Create(log)
}
