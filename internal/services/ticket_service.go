package services

import (
	"errors"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type TicketService interface {
	Create(req *models.CreateTicketRequest) (*models.Ticket, error)
	GetAll(page, size int, filters *models.TicketFilters) (*models.PaginatedResponse, error)
	GetAllForUser(page, size int, filters *models.TicketFilters, userID string, userRole string) (*models.PaginatedResponse, error)
	GetByID(id string) (*models.Ticket, error)
	Update(id string, req *models.CreateTicketRequest) (*models.Ticket, error)
	Delete(id string) error
	UpdateStatus(id string, status string) error
	AssignTechnicians(id string, technicianIDs []string) error
	SignTicket(id string, req *models.SignTicketRequest) (*models.Ticket, error)
	DeleteSignature(id string) (*models.Ticket, error)
}

type ticketService struct {
	ticketRepo     repositories.TicketRepository
	technicianRepo repositories.TechnicianRepository
	clientRepo     repositories.ClientRepository
	categoryRepo   repositories.CategoryRepository
	hierarchyRepo  repositories.HierarchyRepository
}

func NewTicketService(
	ticketRepo repositories.TicketRepository,
	technicianRepo repositories.TechnicianRepository,
	clientRepo repositories.ClientRepository,
	categoryRepo repositories.CategoryRepository,
	hierarchyRepo repositories.HierarchyRepository,
) TicketService {
	return &ticketService{
		ticketRepo:     ticketRepo,
		technicianRepo: technicianRepo,
		clientRepo:     clientRepo,
		categoryRepo:   categoryRepo,
		hierarchyRepo:  hierarchyRepo,
	}
}

func (s *ticketService) Create(req *models.CreateTicketRequest) (*models.Ticket, error) {
	ticket := &models.Ticket{
		ErrorDescription: req.ErrorDescription,
		Priority:         models.TicketPriority(req.Priority),
		Status:           models.TicketStatusOpen,
		OSNumber:         req.OSNumber,
		ComputerBrand:    req.GetBrand(),
		ComputerModel:    req.GetModel(),
		SerialNumber:     req.SerialNumber,
		NodeID:           req.GetNodeID(),
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

func (s *ticketService) GetAll(page, size int, filters *models.TicketFilters) (*models.PaginatedResponse, error) {
	tickets, total, err := s.ticketRepo.FindAll(page, size, filters)
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

// GetAllForUser returns tickets filtered by user permissions
// If user has tickets.view permission, they can see all tickets
// If user has tickets.view_own permission, they only see their assigned tickets
func (s *ticketService) GetAllForUser(page, size int, filters *models.TicketFilters, userID string, userRole string) (*models.PaginatedResponse, error) {
	permissions, _ := s.hierarchyRepo.GetUserPermissions(userID)
	if len(permissions) == 0 {
		permissions, _ = s.hierarchyRepo.GetPermissionsByRoleName(userRole)
	}

	hasViewAll := false
	hasViewOwn := false
	for _, p := range permissions {
		if p == "tickets.view" || p == "*" {
			hasViewAll = true
			break
		}
		if p == "tickets.view_own" {
			hasViewOwn = true
		}
	}

	if hasViewAll {
		return s.GetAll(page, size, filters)
	}

	if hasViewOwn {
		technician, err := s.technicianRepo.FindByUserID(userID)
		if err != nil {
			return &models.PaginatedResponse{
				Content:       []models.TicketDTO{},
				Page:          page,
				Size:          size,
				TotalElements: 0,
				TotalPages:    0,
			}, nil
		}

		if filters == nil {
			filters = &models.TicketFilters{}
		}
		filters.TechnicianID = technician.ID
		return s.GetAll(page, size, filters)
	}

	return s.GetAll(page, size, filters)
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
	existing.CustomerFeedback = req.CustomerFeedback
	existing.Priority = models.TicketPriority(req.Priority)
	if req.OSNumber != "" {
		existing.OSNumber = req.OSNumber
	}
	existing.ComputerBrand = req.GetBrand()
	existing.ComputerModel = req.GetModel()
	existing.SerialNumber = req.SerialNumber
	existing.NodeID = req.GetNodeID()

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

	if err := s.ticketRepo.AssignTechnicians(id, []models.Technician{}); err != nil {
		return nil, err
	}

	if len(req.TechnicianIDs) > 0 {
		technicians, err := s.technicianRepo.FindByIDs(req.TechnicianIDs)
		if err != nil {
			return nil, err
		}
		if err := s.ticketRepo.AssignTechnicians(id, technicians); err != nil {
			return nil, err
		}
	}

	return s.ticketRepo.FindByID(id)
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

func (s *ticketService) SignTicket(id string, req *models.SignTicketRequest) (*models.Ticket, error) {
	ticket, err := s.ticketRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if ticket.TechnicianSignature != "" && ticket.ClientSignature != "" {
		return nil, errors.New("ticket already signed")
	}

	now := time.Now()
	ticket.TechnicianSignature = req.TechnicianSignature
	ticket.ClientSignature = req.ClientSignature
	ticket.SignedByName = req.SignedByName
	ticket.SignedAt = &now

	if err := s.ticketRepo.Update(ticket); err != nil {
		return nil, err
	}

	return ticket, nil
}

func (s *ticketService) DeleteSignature(id string) (*models.Ticket, error) {
	ticket, err := s.ticketRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if ticket.TechnicianSignature == "" && ticket.ClientSignature == "" {
		return nil, errors.New("ticket is not signed")
	}

	ticket.TechnicianSignature = ""
	ticket.ClientSignature = ""
	ticket.SignedByName = ""
	ticket.SignedAt = nil

	if err := s.ticketRepo.Update(ticket); err != nil {
		return nil, err
	}

	return ticket, nil
}
