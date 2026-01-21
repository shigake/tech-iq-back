package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TicketStatus string

const (
	TicketStatusOpen         TicketStatus = "ABERTO"
	TicketStatusInProgress   TicketStatus = "EM_ATENDIMENTO"
	TicketStatusForClosing   TicketStatus = "PARA_FECHAMENTO"
	TicketStatusClosed       TicketStatus = "FECHADO"
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

	// Hierarchy Node (area/department) - e.g., "Itaú", "Vivo"
	NodeID *uint `json:"nodeId" gorm:"index"`
	Node   *Node `json:"node,omitempty" gorm:"foreignKey:NodeID"`

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

	// External Reference (e.g., RITM6261364)
	ExternalReference string `json:"externalReference" gorm:"column:external_reference;type:varchar(100);index"`

	// Store/Location Info
	StoreCode string `json:"storeCode" gorm:"column:store_code;type:varchar(50);index"`
	StoreName string `json:"storeName" gorm:"column:store_name;type:varchar(255)"`

	// Service Location Address
	ServiceStreet       string `json:"serviceStreet" gorm:"column:service_street;type:varchar(255)"`
	ServiceNumber       string `json:"serviceNumber" gorm:"column:service_number;type:varchar(20)"`
	ServiceNeighborhood string `json:"serviceNeighborhood" gorm:"column:service_neighborhood;type:varchar(100)"`
	ServiceCity         string `json:"serviceCity" gorm:"column:service_city;type:varchar(100)"`
	ServiceState        string `json:"serviceState" gorm:"column:service_state;type:varchar(2)"`
	ServiceZipCode      string `json:"serviceZipCode" gorm:"column:service_zipcode;type:varchar(10)"`

	// Contact at location
	ContactName  string `json:"contactName" gorm:"column:contact_name;type:varchar(255)"`
	ContactPhone string `json:"contactPhone" gorm:"column:contact_phone;type:varchar(50)"`

	// Signatures (base64 encoded images)
	TechnicianSignature string     `json:"technicianSignature" gorm:"column:technician_signature;type:text"`
	ClientSignature     string     `json:"clientSignature" gorm:"column:client_signature;type:text"`
	SignedAt            *time.Time `json:"signedAt" gorm:"column:signed_at"`
	SignedByName        string     `json:"signedByName" gorm:"column:signed_by_name;type:varchar(255)"`

	// Dates
	StartDate *time.Time     `json:"startDate"`
	DueDate   *time.Time     `json:"dueDate"`
	ClosedAt  *time.Time     `json:"closedAt" gorm:"column:closed_at"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (t *Ticket) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.OSNumber == "" {
		year := time.Now().Year()
		seqName := fmt.Sprintf("os_number_seq_%d", year)

		tx.Exec(fmt.Sprintf(`
			DO $$ 
			BEGIN
				CREATE SEQUENCE IF NOT EXISTS %s START WITH 1 INCREMENT BY 1;
			EXCEPTION WHEN duplicate_table THEN
				NULL;
			END $$;
		`, seqName))

		var nextSeq int
		tx.Raw(fmt.Sprintf("SELECT nextval('%s')", seqName)).Scan(&nextSeq)

		t.OSNumber = generateOSNumber(year, nextSeq)
	}
	return nil
}

func generateOSNumber(year int, seq int) string {
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
	ID                  string     `json:"id"`
	OSNumber            string     `json:"osNumber"`
	Status              string     `json:"status"`
	Priority            string     `json:"priority"`
	ErrorDescription    string     `json:"errorDescription"`
	CustomerFeedback    string     `json:"customerFeedback"`
	NodeID              *uint      `json:"nodeId"`
	NodeName            string     `json:"nodeName"`
	ClientName          string     `json:"clientName"`
	CategoryName        string     `json:"categoryName"`
	TechnicianCount     int        `json:"technicianCount"`
	ComputerBrand       string     `json:"computerBrand"`
	ComputerModel       string     `json:"computerModel"`
	SerialNumber        string     `json:"serialNumber"`
	TechnicianSignature string     `json:"technicianSignature,omitempty"`
	ClientSignature     string     `json:"clientSignature,omitempty"`
	SignedAt            *time.Time `json:"signedAt"`
	SignedByName        string     `json:"signedByName"`
	IsSigned            bool       `json:"isSigned"`
	StartDate           *time.Time `json:"startDate"`
	DueDate             *time.Time `json:"dueDate"`
	ClosedAt            *time.Time `json:"closedAt"`
	CreatedAt           time.Time  `json:"createdAt"`
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
	nodeName := ""
	if t.Node != nil && t.Node.Name != "" {
		nodeName = t.Node.Name
	}

	return TicketDTO{
		ID:                  t.ID,
		OSNumber:            t.OSNumber,
		Status:              string(t.Status),
		Priority:            string(t.Priority),
		ErrorDescription:    t.ErrorDescription,
		CustomerFeedback:    t.CustomerFeedback,
		NodeID:              t.NodeID,
		NodeName:            nodeName,
		ClientName:          clientName,
		CategoryName:        categoryName,
		TechnicianCount:     len(t.Technicians),
		ComputerBrand:       t.ComputerBrand,
		ComputerModel:       t.ComputerModel,
		SerialNumber:        t.SerialNumber,
		TechnicianSignature: t.TechnicianSignature,
		ClientSignature:     t.ClientSignature,
		SignedAt:            t.SignedAt,
		SignedByName:        t.SignedByName,
		IsSigned:            t.TechnicianSignature != "" && t.ClientSignature != "",
		StartDate:           t.StartDate,
		DueDate:             t.DueDate,
		ClosedAt:            t.ClosedAt,
		CreatedAt:           t.CreatedAt,
	}
}

type CreateTicketRequest struct {
	ErrorDescription string   `json:"errorDescription" validate:"required"`
	Priority         string   `json:"priority"`
	NodeID           *uint    `json:"nodeId"`
	ClientID         string   `json:"clientId"`
	CategoryID       string   `json:"categoryId"`
	TechnicianIDs    []string `json:"technicianIds"`
	StartDate        string   `json:"startDate"`
	DueDate          string   `json:"dueDate"`
	// Accept both old and new field names for compatibility
	ComputerBrand string `json:"computerBrand"`
	ComputerModel string `json:"computerModel"`
	Manufacturer  string `json:"manufacturer"` // alias for ComputerBrand
	Model         string `json:"model"`        // alias for ComputerModel
	SerialNumber  string `json:"serialNumber"`
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

type SignTicketRequest struct {
	TechnicianSignature string `json:"technicianSignature" validate:"required"`
	ClientSignature     string `json:"clientSignature" validate:"required"`
	SignedByName        string `json:"signedByName" validate:"required"`
}

type AssignTechnicianRequest struct {
	TechnicianIDs []string `json:"technicianIds" validate:"required"`
}

// TicketFilters contains all possible filters for ticket queries
type TicketFilters struct {
	Status       string `json:"status"`
	Priority     string `json:"priority"`
	NodeID       string `json:"nodeId"`
	ClientID     string `json:"clientId"`
	CategoryID   string `json:"categoryId"`
	TechnicianID string `json:"technicianId"`
	Search       string `json:"search"`
	DateFrom     string `json:"dateFrom"`
	DateTo       string `json:"dateTo"`
}
