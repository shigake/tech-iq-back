package repositories

import (
	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
)

type DashboardConfigRepository interface {
	FindByUserID(userID string) (*models.UserDashboardConfig, error)
	Save(config *models.UserDashboardConfig) error
}

type dashboardConfigRepository struct {
	db *gorm.DB
}

func NewDashboardConfigRepository(db *gorm.DB) DashboardConfigRepository {
	return &dashboardConfigRepository{db: db}
}

func (r *dashboardConfigRepository) FindByUserID(userID string) (*models.UserDashboardConfig, error) {
	var config models.UserDashboardConfig
	err := r.db.Where("user_id = ?", userID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *dashboardConfigRepository) Save(config *models.UserDashboardConfig) error {
	var existing models.UserDashboardConfig
	err := r.db.Where("user_id = ?", config.UserID).First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.db.Create(config).Error
		}
		return err
	}
	existing.Widgets = config.Widgets
	return r.db.Save(&existing).Error
}
