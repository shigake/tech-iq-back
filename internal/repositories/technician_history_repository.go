package repositories

import (
	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
)

type TechnicianHistoryRepository interface {
	Create(history *models.TechnicianHistory) error
	FindByTechnicianID(technicianID string, page, size int) ([]models.TechnicianHistory, int64, error)
	FindByID(id string) (*models.TechnicianHistory, error)
}

type technicianHistoryRepository struct {
	db *gorm.DB
}

func NewTechnicianHistoryRepository(db *gorm.DB) TechnicianHistoryRepository {
	return &technicianHistoryRepository{db: db}
}

func (r *technicianHistoryRepository) Create(history *models.TechnicianHistory) error {
	return r.db.Create(history).Error
}

func (r *technicianHistoryRepository) FindByTechnicianID(technicianID string, page, size int) ([]models.TechnicianHistory, int64, error) {
	var histories []models.TechnicianHistory
	var total int64

	baseQuery := r.db.Model(&models.TechnicianHistory{}).Where("technician_id = ?", technicianID)
	baseQuery.Count(&total)

	offset := page * size
	err := baseQuery.
		Preload("Ticket").
		Preload("CreatedByUser").
		Offset(offset).
		Limit(size).
		Order("activated_at DESC").
		Find(&histories).Error
	if err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

func (r *technicianHistoryRepository) FindByID(id string) (*models.TechnicianHistory, error) {
	var history models.TechnicianHistory
	err := r.db.Preload("Ticket").Preload("CreatedByUser").Where("id = ?", id).First(&history).Error
	if err != nil {
		return nil, err
	}
	return &history, nil
}
