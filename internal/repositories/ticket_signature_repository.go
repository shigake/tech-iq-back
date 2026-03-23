package repositories

import (
	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
)

type TicketSignatureRepository interface {
	Create(signature *models.TicketSignature) error
	FindByTicketID(ticketID string) (*models.TicketSignature, error)
	Update(signature *models.TicketSignature) error
	Delete(ticketID string) error
}

type ticketSignatureRepository struct {
	db *gorm.DB
}

func NewTicketSignatureRepository(db *gorm.DB) TicketSignatureRepository {
	return &ticketSignatureRepository{db: db}
}

func (r *ticketSignatureRepository) Create(signature *models.TicketSignature) error {
	return r.db.Create(signature).Error
}

func (r *ticketSignatureRepository) FindByTicketID(ticketID string) (*models.TicketSignature, error) {
	var signature models.TicketSignature
	err := r.db.Where("ticket_id = ?", ticketID).First(&signature).Error
	if err != nil {
		return nil, err
	}
	return &signature, nil
}

func (r *ticketSignatureRepository) Update(signature *models.TicketSignature) error {
	return r.db.Save(signature).Error
}

func (r *ticketSignatureRepository) Delete(ticketID string) error {
	return r.db.Where("ticket_id = ?", ticketID).Delete(&models.TicketSignature{}).Error
}
