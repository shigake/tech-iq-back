package repositories

import (
	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
)

type TicketRepository interface {
	Create(ticket *models.Ticket) error
	FindAll(page, size int, filters *models.TicketFilters) ([]models.Ticket, int64, error)
	FindByID(id string) (*models.Ticket, error)
	Update(ticket *models.Ticket) error
	Delete(id string) error
	CountByStatus(status string) (int64, error)
	CountAll() (int64, error)
	GroupByStatus() ([]models.TicketsByStatus, error)
	UpdateStatus(id string, status string) error
	AssignTechnicians(id string, technicians []models.Technician) error
	GetRecent(limit int) ([]models.Ticket, error)
}

type ticketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) TicketRepository {
	return &ticketRepository{db: db}
}

func (r *ticketRepository) Create(ticket *models.Ticket) error {
	return r.db.Create(ticket).Error
}

func (r *ticketRepository) FindAll(page, size int, filters *models.TicketFilters) ([]models.Ticket, int64, error) {
	var tickets []models.Ticket
	var total int64

	query := r.db.Model(&models.Ticket{})

	// Apply filters
	if filters != nil {
		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}
		if filters.Priority != "" {
			query = query.Where("priority = ?", filters.Priority)
		}
		if filters.ClientID != "" {
			query = query.Where("client_id = ?", filters.ClientID)
		}
		if filters.CategoryID != "" {
			query = query.Where("category_id = ?", filters.CategoryID)
		}
		if filters.TechnicianID != "" {
			query = query.Joins("JOIN ticket_technicians tt ON tt.ticket_id = tickets.id").
				Where("tt.technician_id = ?", filters.TechnicianID)
		}
		if filters.Search != "" {
			search := "%" + filters.Search + "%"
			query = query.Where("error_description ILIKE ? OR serial_number ILIKE ? OR computer_brand ILIKE ? OR computer_model ILIKE ?", 
				search, search, search, search)
		}
		if filters.DateFrom != "" {
			query = query.Where("created_at >= ?", filters.DateFrom)
		}
		if filters.DateTo != "" {
			query = query.Where("created_at <= ?", filters.DateTo)
		}
	}

	// Count total with filters applied
	query.Count(&total)

	// Fetch paginated results
	offset := page * size
	err := query.
		Preload("Client").
		Preload("Category").
		Preload("Technicians").
		Offset(offset).
		Limit(size).
		Order("created_at DESC").
		Find(&tickets).Error
	if err != nil {
		return nil, 0, err
	}

	return tickets, total, nil
}

func (r *ticketRepository) FindByID(id string) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.
		Preload("Client").
		Preload("Category").
		Preload("Technicians").
		Preload("Files").
		Where("id = ?", id).
		First(&ticket).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *ticketRepository) Update(ticket *models.Ticket) error {
	return r.db.Save(ticket).Error
}

func (r *ticketRepository) Delete(id string) error {
	// First remove technician associations
	r.db.Exec("DELETE FROM ticket_technicians WHERE ticket_id = ?", id)
	return r.db.Where("id = ?", id).Delete(&models.Ticket{}).Error
}

func (r *ticketRepository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Ticket{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (r *ticketRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&models.Ticket{}).Count(&count).Error
	return count, err
}

func (r *ticketRepository) GroupByStatus() ([]models.TicketsByStatus, error) {
	var result []models.TicketsByStatus
	err := r.db.Model(&models.Ticket{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Order("count DESC").
		Scan(&result).Error
	return result, err
}

func (r *ticketRepository) UpdateStatus(id string, status string) error {
	return r.db.Model(&models.Ticket{}).Where("id = ?", id).Update("status", status).Error
}

func (r *ticketRepository) AssignTechnicians(id string, technicians []models.Technician) error {
	var ticket models.Ticket
	if err := r.db.First(&ticket, "id = ?", id).Error; err != nil {
		return err
	}
	return r.db.Model(&ticket).Association("Technicians").Replace(technicians)
}

func (r *ticketRepository) GetRecent(limit int) ([]models.Ticket, error) {
	var tickets []models.Ticket
	err := r.db.Order("updated_at DESC").Limit(limit).Find(&tickets).Error
	return tickets, err
}
