package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "ABERTO"
	TicketStatusInProgress TicketStatus = "EM_ATENDIMENTO"
	TicketStatusForClosing TicketStatus = "PARA_FECHAMENTO"
	TicketStatusClosed     TicketStatus = "FECHADO"
	TicketStatusUnproductive TicketStatus = "IMPRODUTIVO"
)

type TicketPriority string

const (
	TicketPriorityLow    TicketPriority = "BAIXA"
	TicketPriorityNormal TicketPriority = "NORMAL"
	TicketPriorityHigh   TicketPriority = "ALTA"
	TicketPriorityUrgent TicketPriority = "URGENTE"
)

type Ticket struct {
	ID               string         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OSNumber         string         `json:"osNumber" gorm:"type:varchar(50);uniqueIndex"`
	Status           TicketStatus   `json:"status" gorm:"type:varchar(50);default:ABERTO;index"`
	Priority         TicketPriority `json:"priority" gorm:"type:varchar(20);default:NORMAL"`
	ErrorDescription string         `json:"errorDescription" gorm:"type:text"`
	CustomerFeedback string         `json:"customerFeedback" gorm:"type:text"`

	// Client
	ClientID *string `json:"clientId" gorm:"type:uuid"`
	Client   *Client `json:"client,omitempty" gorm:"foreignKey:ClientID"`

	// Category
	CategoryID *string   `json:"categoryId" gorm:"type:uuid"`
	Category   *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`

	// Technicians (many-to-many) - uses string ID like Java
	Technicians []Technician `json:"technicians,omitempty" gorm:"many2many:ticket_technicians;joinForeignKey:ticket_id;joinReferences:technician_id"`

	// Files
	Files []TicketFile `json:"files,omitempty" gorm:"foreignKey:TicketID"`

	// Computer Info - matching existing database columns
	ComputerBrand string `json:"computerBrand" gorm:"column:computer_brand;type:varchar(100)"`
	ComputerModel string `json:"computerModel" gorm:"column:computer_model;type:varchar(100)"`
	SerialNumber  string `json:"serialNumber" gorm:"column:serial_number;type:varchar(100)"`

	// Dates
	StartDate *time.Time     `json:"startDate"`
	DueDate   *time.Time     `json:"dueDate"`
	ClosedAt  *time.Time     `json:"closedAt" gorm:"column:closed_at"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (t *Ticket) BeforeCreate(tx *gorm.DB) error {
	// Generate UUID if not set
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	// Generate OS number if not set
	if t.OSNumber == "" {
		// Get the highest sequence number for the current year
		year := time.Now().Year()
		var maxOS string
		tx.Model(&Ticket{}).
			Where("os_number LIKE ?", fmt.Sprintf("%d-%%", year)).
			Order("os_number DESC").
			Limit(1).
			Pluck("os_number", &maxOS)
		
		nextSeq := 1
		if maxOS != "" {
			// Extract sequence number from format "YYYY-NNNNNN"
			parts := strings.Split(maxOS, "-")
			if len(parts) == 2 {
				if seq, err := strconv.Atoi(parts[1]); err == nil {
					nextSeq = seq + 1
				}
			}
		}
		
		t.OSNumber = generateOSNumber(nextSeq)
	}
	return nil
}

func generateOSNumber(seq int) string {
	year := time.Now().Year()
	return fmt.Sprintf("%d-%06d", year, seq)
}

type TicketFile struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TicketID  string    `json:"ticketId" gorm:"type:uuid"`
	FileName  string    `json:"fileName" gorm:"type:varchar(255)"`
	FilePath  string    `json:"filePath" gorm:"type:varchar(500)"`
	FileType  string    `json:"fileType" gorm:"type:varchar(100)"`
	FileSize  int64     `json:"fileSize"`
	CreatedAt time.Time `json:"createdAt"`
}

// DTOs

type TicketDTO struct {
	ID               string     `json:"id"`
	OSNumber         string     `json:"osNumber"`
	Status           string     `json:"status"`
	Priority         string     `json:"priority"`
	ErrorDescription string     `json:"errorDescription"`
	CustomerFeedback string     `json:"customerFeedback"`
	ClientName       string     `json:"clientName"`
	CategoryName     string     `json:"categoryName"`
	TechnicianCount  int        `json:"technicianCount"`
	ComputerBrand    string     `json:"computerBrand"`
	ComputerModel    string     `json:"computerModel"`
	SerialNumber     string     `json:"serialNumber"`
	StartDate        *time.Time `json:"startDate"`
	DueDate          *time.Time `json:"dueDate"`
	ClosedAt         *time.Time `json:"closedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
}

func (t *Ticket) ToDTO() TicketDTO {
	clientName := ""
	if t.Client != nil && t.Client.FullName != "" {
		clientName = t.Client.FullName
	}
	categoryName := ""
	if t.Category != nil && t.Category.Name != "" {
		categoryName = t.Category.Name
	}

	return TicketDTO{
		ID:               t.ID,
		OSNumber:         t.OSNumber,
		Status:           string(t.Status),
		Priority:         string(t.Priority),
		ErrorDescription: t.ErrorDescription,
		CustomerFeedback: t.CustomerFeedback,
		ClientName:       clientName,
		CategoryName:     categoryName,
		TechnicianCount:  len(t.Technicians),
		ComputerBrand:    t.ComputerBrand,
		ComputerModel:    t.ComputerModel,
		SerialNumber:     t.SerialNumber,
		StartDate:        t.StartDate,
		DueDate:          t.DueDate,
		ClosedAt:         t.ClosedAt,
		CreatedAt:        t.CreatedAt,
	}
}

type CreateTicketRequest struct {
	ErrorDescription string   `json:"errorDescription" validate:"required"`
	Priority         string   `json:"priority"`
	ClientID         string   `json:"clientId"`
	CategoryID       string   `json:"categoryId"`
	TechnicianIDs    []string `json:"technicianIds"`
	StartDate        string   `json:"startDate"`
	DueDate          string   `json:"dueDate"`
	// Accept both old and new field names for compatibility
	ComputerBrand    string   `json:"computerBrand"`
	ComputerModel    string   `json:"computerModel"`
	Manufacturer     string   `json:"manufacturer"`    // alias for ComputerBrand
	Model            string   `json:"model"`           // alias for ComputerModel
	SerialNumber     string   `json:"serialNumber"`
}

// GetBrand returns computerBrand or manufacturer (for backward compatibility)
func (r *CreateTicketRequest) GetBrand() string {
	if r.ComputerBrand != "" {
		return r.ComputerBrand
	}
	return r.Manufacturer
}

// GetModel returns computerModel or model (for backward compatibility)
func (r *CreateTicketRequest) GetModel() string {
	if r.ComputerModel != "" {
		return r.ComputerModel
	}
	return r.Model
}

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required"`
}

type AssignTechnicianRequest struct {
	TechnicianIDs []string `json:"technicianIds" validate:"required"`
}
