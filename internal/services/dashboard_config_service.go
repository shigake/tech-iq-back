package services

import (
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type DashboardConfigService interface {
	GetConfig(userID string) (*models.UserDashboardConfig, error)
	SaveConfig(userID string, req *models.SaveDashboardConfigRequest) (*models.UserDashboardConfig, error)
}

type dashboardConfigService struct {
	repo repositories.DashboardConfigRepository
}

func NewDashboardConfigService(repo repositories.DashboardConfigRepository) DashboardConfigService {
	return &dashboardConfigService{repo: repo}
}

func (s *dashboardConfigService) GetConfig(userID string) (*models.UserDashboardConfig, error) {
	config, err := s.repo.FindByUserID(userID)
	if err != nil {
		return &models.UserDashboardConfig{
			UserID:  userID,
			Widgets: models.DashboardWidgets{},
		}, nil
	}
	return config, nil
}

func (s *dashboardConfigService) SaveConfig(userID string, req *models.SaveDashboardConfigRequest) (*models.UserDashboardConfig, error) {
	config := &models.UserDashboardConfig{
		UserID:  userID,
		Widgets: req.Widgets,
	}
	if err := s.repo.Save(config); err != nil {
		return nil, err
	}
	return s.repo.FindByUserID(userID)
}
