package services

import (
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type TechnicianHistoryService interface {
	Create(technicianID string, req *models.CreateTechnicianHistoryRequest, userID string) (*models.TechnicianHistory, error)
	GetByTechnicianID(technicianID string, page, size int) (*models.PaginatedResponse, error)
}

type technicianHistoryService struct {
	repo           repositories.TechnicianHistoryRepository
	technicianRepo repositories.TechnicianRepository
}

func NewTechnicianHistoryService(repo repositories.TechnicianHistoryRepository, technicianRepo repositories.TechnicianRepository) TechnicianHistoryService {
	return &technicianHistoryService{
		repo:           repo,
		technicianRepo: technicianRepo,
	}
}

func (s *technicianHistoryService) Create(technicianID string, req *models.CreateTechnicianHistoryRequest, userID string) (*models.TechnicianHistory, error) {
	technician, err := s.technicianRepo.FindByID(technicianID)
	if err != nil {
		return nil, err
	}

	activatedAt, err := time.Parse(time.RFC3339, req.ActivatedAt)
	if err != nil {
		activatedAt = time.Now()
	}

	history := &models.TechnicianHistory{
		TechnicianID:  technicianID,
		ActivatedAt:   activatedAt,
		AcceptedValue: req.AcceptedValue,
		ProposedValue: req.ProposedValue,
		StatusAtTime:  technician.Status,
		Notes:         req.Notes,
		CreatedBy:     userID,
	}

	if req.TicketID != "" {
		history.TicketID = &req.TicketID
	}

	if err := s.repo.Create(history); err != nil {
		return nil, err
	}

	return history, nil
}

func (s *technicianHistoryService) GetByTechnicianID(technicianID string, page, size int) (*models.PaginatedResponse, error) {
	histories, total, err := s.repo.FindByTechnicianID(technicianID, page, size)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	return &models.PaginatedResponse{
		Content:       histories,
		Page:          page,
		Size:          size,
		TotalElements: total,
		TotalPages:    totalPages,
	}, nil
}
