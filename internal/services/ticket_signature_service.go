package services

import (
	"errors"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type TicketSignatureService interface {
	CreateOrUpdate(ticketID string, req *models.CreateTicketSignatureRequest) (*models.TicketSignature, error)
	GetByTicketID(ticketID string) (*models.TicketSignature, error)
	Delete(ticketID string) error
}

type ticketSignatureService struct {
	signatureRepo repositories.TicketSignatureRepository
	ticketRepo    repositories.TicketRepository
}

func NewTicketSignatureService(signatureRepo repositories.TicketSignatureRepository, ticketRepo repositories.TicketRepository) TicketSignatureService {
	return &ticketSignatureService{
		signatureRepo: signatureRepo,
		ticketRepo:    ticketRepo,
	}
}

func (s *ticketSignatureService) CreateOrUpdate(ticketID string, req *models.CreateTicketSignatureRequest) (*models.TicketSignature, error) {
	ticket, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		return nil, err
	}

	if req.SignatureType != "DIGITAL" && req.SignatureType != "PHYSICAL" {
		return nil, errors.New("signatureType must be DIGITAL or PHYSICAL")
	}

	if req.SignatureType == "DIGITAL" && req.SignatureData == "" {
		return nil, errors.New("signatureData is required for DIGITAL signature")
	}

	if req.SignatureType == "PHYSICAL" && req.SignatureImageURL == "" {
		return nil, errors.New("signatureImageUrl is required for PHYSICAL signature")
	}

	existing, err := s.signatureRepo.FindByTicketID(ticketID)
	if err == nil && existing != nil {
		existing.SignatureType = req.SignatureType
		existing.SignatureImageURL = req.SignatureImageURL
		existing.SignatureData = req.SignatureData
		existing.SignedBy = req.SignedBy
		existing.SignedAt = time.Now()
		if err := s.signatureRepo.Update(existing); err != nil {
			return nil, err
		}
		s.updateTicketToFinalized(ticket)
		return existing, nil
	}

	signature := &models.TicketSignature{
		TicketID:          ticketID,
		SignatureType:     req.SignatureType,
		SignatureImageURL: req.SignatureImageURL,
		SignatureData:     req.SignatureData,
		SignedAt:          time.Now(),
		SignedBy:          req.SignedBy,
	}

	if err := s.signatureRepo.Create(signature); err != nil {
		return nil, err
	}

	s.updateTicketToFinalized(ticket)

	return signature, nil
}

func (s *ticketSignatureService) updateTicketToFinalized(ticket *models.Ticket) error {
	if ticket.Status == models.TicketStatusClosed {
		return s.ticketRepo.UpdateStatus(ticket.ID, string(models.TicketStatusFinalized))
	}
	return nil
}

func (s *ticketSignatureService) GetByTicketID(ticketID string) (*models.TicketSignature, error) {
	return s.signatureRepo.FindByTicketID(ticketID)
}

func (s *ticketSignatureService) Delete(ticketID string) error {
	return s.signatureRepo.Delete(ticketID)
}
