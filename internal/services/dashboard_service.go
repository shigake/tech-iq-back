package services

import (
	"fmt"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type DashboardService interface {
	GetStats() (*models.DashboardStats, error)
	GetTicketsByStatus() ([]models.TicketsByStatus, error)
	GetTechniciansByState() ([]models.TechniciansByState, error)
	GetRecentActivity(limit int) ([]models.RecentActivity, error)
}

type dashboardService struct {
	technicianRepo repositories.TechnicianRepository
	ticketRepo     repositories.TicketRepository
	clientRepo     repositories.ClientRepository
}

func NewDashboardService(
	technicianRepo repositories.TechnicianRepository,
	ticketRepo repositories.TicketRepository,
	clientRepo repositories.ClientRepository,
) DashboardService {
	return &dashboardService{
		technicianRepo: technicianRepo,
		ticketRepo:     ticketRepo,
		clientRepo:     clientRepo,
	}
}

func (s *dashboardService) GetStats() (*models.DashboardStats, error) {
	totalTechnicians, _ := s.technicianRepo.CountAll()
	activeTechnicians, _ := s.technicianRepo.CountByStatus("ATIVO")
	totalTickets, _ := s.ticketRepo.CountAll()
	openTickets, _ := s.ticketRepo.CountByStatus("ABERTO")
	inProgressTickets, _ := s.ticketRepo.CountByStatus("EM_ATENDIMENTO")
	closedTickets, _ := s.ticketRepo.CountByStatus("FECHADO")
	totalClients, _ := s.clientRepo.Count()

	return &models.DashboardStats{
		TotalTechnicians:  totalTechnicians,
		ActiveTechnicians: activeTechnicians,
		TotalTickets:      totalTickets,
		OpenTickets:       openTickets,
		InProgressTickets: inProgressTickets,
		ClosedTickets:     closedTickets,
		TotalClients:      totalClients,
	}, nil
}

func (s *dashboardService) GetTicketsByStatus() ([]models.TicketsByStatus, error) {
	return s.ticketRepo.GroupByStatus()
}

func (s *dashboardService) GetTechniciansByState() ([]models.TechniciansByState, error) {
	return s.technicianRepo.GroupByState()
}

func (s *dashboardService) GetRecentActivity(limit int) ([]models.RecentActivity, error) {
	var activities []models.RecentActivity

	// Get recent technicians
	technicians, _ := s.technicianRepo.GetRecent(limit / 2)
	for _, t := range technicians {
		action := "criado"
		if t.UpdatedAt.After(t.CreatedAt) {
			action = "atualizado"
		}
		activities = append(activities, models.RecentActivity{
			ID:          t.ID,
			Type:        "technician",
			Action:      action,
			Title:       fmt.Sprintf("Técnico %s", t.FullName),
			Description: fmt.Sprintf("Técnico %s foi %s", t.FullName, action),
			Timestamp:   formatTimeAgo(t.UpdatedAt),
		})
	}

	// Get recent tickets
	tickets, _ := s.ticketRepo.GetRecent(limit / 2)
	for _, t := range tickets {
		action := "criado"
		if t.UpdatedAt.After(t.CreatedAt) {
			action = "atualizado"
		}
		activities = append(activities, models.RecentActivity{
			ID:          fmt.Sprintf("%d", t.ID),
			Type:        "ticket",
			Action:      action,
			Title:       fmt.Sprintf("Ticket #%d", t.ID),
			Description: fmt.Sprintf("Ticket #%d foi %s - %s", t.ID, action, t.Status),
			Timestamp:   formatTimeAgo(t.UpdatedAt),
		})
	}

	return activities, nil
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)
	
	if diff < time.Minute {
		return "agora"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "há 1 minuto"
		}
		return fmt.Sprintf("há %d minutos", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "há 1 hora"
		}
		return fmt.Sprintf("há %d horas", hours)
	} else {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "há 1 dia"
		}
		return fmt.Sprintf("há %d dias", days)
	}
}
