package services

import (
	"errors"
	"time"

	"github.com/tech-erp/backend/internal/models"
	"github.com/tech-erp/backend/internal/repositories"
)

type TicketService interface {
	Create(req *models.CreateTicketRequest) (*models.Ticket, error)
	GetAll(page, size int) (*models.PaginatedResponse, error)
	GetByID(id string) (*models.Ticket, error)
	Update(id string, req *models.CreateTicketRequest) (*models.Ticket, error)
	Delete(id string) error
	UpdateStatus(id string, status string) error
	AssignTechnicians(id string, technicianIDs []string) error
}

type ticketService struct {
	ticketRepo     repositories.TicketRepository
	technicianRepo repositories.TechnicianRepository
	clientRepo     repositories.ClientRepository
	categoryRepo   repositories.CategoryRepository
}

func NewTicketService(
	ticketRepo repositories.TicketRepository,
	technicianRepo repositories.TechnicianRepository,
	clientRepo repositories.ClientRepository,
	categoryRepo repositories.CategoryRepository,
) TicketService {
	return &ticketService{
		ticketRepo:     ticketRepo,
		technicianRepo: technicianRepo,
		clientRepo:     clientRepo,
		categoryRepo:   categoryRepo,
	}
}

func (s *ticketService) Create(req *models.CreateTicketRequest) (*models.Ticket, error) {
	ticket := &models.Ticket{
		ErrorDescription: req.ErrorDescription,
		Priority:         models.TicketPriority(req.Priority),
		Status:           models.TicketStatusOpen,
		ComputerBrand:    req.GetBrand(),
		ComputerModel:    req.GetModel(),
		SerialNumber:     req.SerialNumber,
	}

	// Set ClientID
	if req.ClientID != "" {
		ticket.ClientID = &req.ClientID
	}

	// Set CategoryID
	if req.CategoryID != "" && req.CategoryID != "0" {
		ticket.CategoryID = &req.CategoryID
	}

	// Parse dates
	if req.StartDate != "" {
		if t, err := time.Parse(time.RFC3339, req.StartDate); err == nil {
			ticket.StartDate = &t
		}
	}
	if req.DueDate != "" {
		if t, err := time.Parse(time.RFC3339, req.DueDate); err == nil {
			ticket.DueDate = &t
		}
	}

	// Assign technicians
	if len(req.TechnicianIDs) > 0 {
		technicians, err := s.technicianRepo.FindByIDs(req.TechnicianIDs)
		if err != nil {
			return nil, err
		}
		ticket.Technicians = technicians
	}

	if err := s.ticketRepo.Create(ticket); err != nil {
		return nil, err
	}

	return ticket, nil
}

func (s *ticketService) GetAll(page, size int) (*models.PaginatedResponse, error) {
	tickets, total, err := s.ticketRepo.FindAll(page, size)
	if err != nil {
		return nil, err
	}

	dtos := make([]models.TicketDTO, len(tickets))
	for i, t := range tickets {
		dtos[i] = t.ToDTO()
	}

	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	return &models.PaginatedResponse{
		Content:       dtos,
		Page:          page,
		Size:          size,
		TotalElements: total,
		TotalPages:    totalPages,
	}, nil
}

func (s *ticketService) GetByID(id string) (*models.Ticket, error) {
	return s.ticketRepo.FindByID(id)
}

func (s *ticketService) Update(id string, req *models.CreateTicketRequest) (*models.Ticket, error) {
	existing, err := s.ticketRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	existing.ErrorDescription = req.ErrorDescription
	existing.Priority = models.TicketPriority(req.Priority)
	existing.ComputerBrand = req.GetBrand()
	existing.ComputerModel = req.GetModel()
	existing.SerialNumber = req.SerialNumber

	// Set ClientID
	if req.ClientID != "" {
		existing.ClientID = &req.ClientID
	}

	// Set CategoryID
	if req.CategoryID != "" && req.CategoryID != "0" {
		existing.CategoryID = &req.CategoryID
	}

	if req.StartDate != "" {
		if t, err := time.Parse(time.RFC3339, req.StartDate); err == nil {
			existing.StartDate = &t
		}
	}
	if req.DueDate != "" {
		if t, err := time.Parse(time.RFC3339, req.DueDate); err == nil {
			existing.DueDate = &t
		}
	}

	if err := s.ticketRepo.Update(existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *ticketService) Delete(id string) error {
	return s.ticketRepo.Delete(id)
}

func (s *ticketService) UpdateStatus(id string, status string) error {
	validStatuses := map[string]bool{
		"ABERTO":          true,
		"EM_ATENDIMENTO":  true,
		"PARA_FECHAMENTO": true,
		"FECHADO":         true,
		"IMPRODUTIVO":     true,
	}

	if !validStatuses[status] {
		return errors.New("invalid status")
	}

	return s.ticketRepo.UpdateStatus(id, status)
}

func (s *ticketService) AssignTechnicians(id string, technicianIDs []string) error {
	technicians, err := s.technicianRepo.FindByIDs(technicianIDs)
	if err != nil {
		return err
	}

	return s.ticketRepo.AssignTechnicians(id, technicians)
}
