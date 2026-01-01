package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EmailEntry represents a single email entry with type
type EmailEntry struct {
	Email string `json:"email"`
	Type  string `json:"type"`
}

// EmailArray is a custom type for PostgreSQL JSONB array of email objects
type EmailArray []EmailEntry

func (a EmailArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	return json.Marshal(a)
}

func (a *EmailArray) Scan(value interface{}) error {
	if value == nil {
		*a = []EmailEntry{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan EmailArray")
	}
	return json.Unmarshal(bytes, a)
}

// PhoneEntry represents a single phone entry with type
type PhoneEntry struct {
	Number string `json:"number"`
	Type   string `json:"type"`
}

// PhoneArray is a custom type for PostgreSQL JSONB array of phone objects
type PhoneArray []PhoneEntry

func (a PhoneArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	return json.Marshal(a)
}

func (a *PhoneArray) Scan(value interface{}) error {
	if value == nil {
		*a = []PhoneEntry{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan PhoneArray")
	}
	return json.Unmarshal(bytes, a)
}

// StringArray is a custom type for PostgreSQL JSONB array of strings
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	return json.Marshal(a)
}

func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan StringArray")
	}
	return json.Unmarshal(bytes, a)
}

// SkillsMap is a custom type for PostgreSQL JSONB map of skills
type SkillsMap map[string]bool

func (s SkillsMap) Value() (driver.Value, error) {
	if s == nil {
		return "{}", nil
	}
	return json.Marshal(s)
}

func (s *SkillsMap) Scan(value interface{}) error {
	if value == nil {
		*s = make(map[string]bool)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan SkillsMap")
	}
	return json.Unmarshal(bytes, s)
}

type Technician struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	FullName  string    `json:"fullName" gorm:"not null;type:varchar(255)"`
	TradeName string    `json:"tradeName" gorm:"type:varchar(255)"`
	CPF       string    `json:"cpf" gorm:"type:varchar(14);index"`
	CNPJ      string    `json:"cnpj" gorm:"type:varchar(18);index"`
	RG        string    `json:"rg" gorm:"type:varchar(20)"`
	Contact   string    `json:"contact" gorm:"type:varchar(255)"`
	Status    string    `json:"status" gorm:"type:varchar(20);default:ATIVO;index"`
	Type      string    `json:"type" gorm:"type:varchar(20);default:PARCERIA"` // PARCERIA, PONTUAL

	// Contact info (JSONB arrays)
	Emails EmailArray `json:"emails" gorm:"type:jsonb;default:'[]'"`
	Phones PhoneArray `json:"phones" gorm:"type:jsonb;default:'[]'"`

	MinCallValue string `json:"minCallValue" gorm:"type:varchar(50)"`
	Observation  string `json:"observation" gorm:"type:text"`

	// Address
	Street       string `json:"street" gorm:"type:varchar(255)"`
	Number       string `json:"number" gorm:"type:varchar(20)"`
	Complement   string `json:"complement" gorm:"type:varchar(100)"`
	Neighborhood string `json:"neighborhood" gorm:"type:varchar(100)"`
	City         string `json:"city" gorm:"type:varchar(100);index"`
	State        string `json:"state" gorm:"type:varchar(50);index"`
	ZipCode      string `json:"zipCode" gorm:"type:varchar(10)"`

	// Bank Info
	BankName      string `json:"bankName" gorm:"type:varchar(200)"`
	Agency        string `json:"agency" gorm:"type:varchar(50)"`
	AccountNumber string `json:"accountNumber" gorm:"type:varchar(50)"`
	AccountType   string `json:"accountType" gorm:"type:varchar(50)"`
	AccountDigit  string `json:"accountDigit" gorm:"type:varchar(10)"`
	AccountHolder string `json:"accountHolder" gorm:"type:varchar(255)"`
	HolderCPF     string `json:"holderCpf" gorm:"type:varchar(14)"`
	PixKey        string `json:"pixKey" gorm:"type:varchar(255)"`

	// Skills (JSONB)
	Skills SkillsMap `json:"skills" gorm:"type:jsonb;default:'{}'"`

	// Knowledge descriptions
	KnowledgeDescription  string `json:"knowledgeDescription" gorm:"type:text"`
	EquipmentDescription  string `json:"equipmentDescription" gorm:"type:text"`

	// Other Info
	Vehicle string `json:"vehicle" gorm:"type:varchar(20);default:NONE"` // NONE, CAR, MOTORBIKE

	// Control
	InAttendance bool      `json:"inAttendance" gorm:"default:false"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`

	// Relationships
	Tickets []Ticket `json:"tickets,omitempty" gorm:"many2many:ticket_technicians"`
}

func (t *Technician) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.Skills == nil {
		t.Skills = make(map[string]bool)
	}
	if t.Emails == nil {
		t.Emails = EmailArray{}
	}
	if t.Phones == nil {
		t.Phones = PhoneArray{}
	}
	return nil
}

// TechnicianDTO is a simplified version for listing
type TechnicianDTO struct {
	ID           string       `json:"id"`
	FullName     string       `json:"fullName"`
	TradeName    string       `json:"tradeName"`
	City         string       `json:"city"`
	State        string       `json:"state"`
	Status       string       `json:"status"`
	Type         string       `json:"type"`
	Emails       []EmailEntry `json:"emails"`
	Phones       []PhoneEntry `json:"phones"`
	InAttendance bool         `json:"inAttendance"`
	CreatedAt    time.Time    `json:"createdAt"`
}

func (t *Technician) ToDTO() TechnicianDTO {
	return TechnicianDTO{
		ID:           t.ID,
		FullName:     t.FullName,
		TradeName:    t.TradeName,
		City:         t.City,
		State:        t.State,
		Status:       t.Status,
		Type:         t.Type,
		Emails:       t.Emails,
		Phones:       t.Phones,
		InAttendance: t.InAttendance,
		CreatedAt:    t.CreatedAt,
	}
}

// CreateTechnicianRequest is the request body for creating a technician
type CreateTechnicianRequest struct {
	FullName             string            `json:"fullName" validate:"required,min=2"`
	TradeName            string            `json:"tradeName"`
	CPF                  string            `json:"cpf"`
	CNPJ                 string            `json:"cnpj"`
	RG                   string            `json:"rg"`
	Contact              string            `json:"contact"`
	Status               string            `json:"status"`
	Type                 string            `json:"type"`
	Emails               []EmailEntry      `json:"emails"`
	Phones               []PhoneEntry      `json:"phones"`
	MinCallValue         string            `json:"minCallValue"`
	Observation          string            `json:"observation"`
	Street               string            `json:"street"`
	Number               string            `json:"number"`
	Complement           string            `json:"complement"`
	Neighborhood         string            `json:"neighborhood"`
	City                 string            `json:"city"`
	State                string            `json:"state"`
	ZipCode              string            `json:"zipCode"`
	BankName             string            `json:"bankName"`
	Agency               string            `json:"agency"`
	AccountNumber        string            `json:"accountNumber"`
	AccountType          string            `json:"accountType"`
	AccountDigit         string            `json:"accountDigit"`
	AccountHolder        string            `json:"accountHolder"`
	HolderCPF            string            `json:"holderCpf"`
	PixKey               string            `json:"pixKey"`
	Skills               map[string]bool   `json:"skills"`
	KnowledgeDescription string            `json:"knowledgeDescription"`
	EquipmentDescription string            `json:"equipmentDescription"`
	Vehicle              string            `json:"vehicle"`
}

func (r *CreateTechnicianRequest) ToModel() *Technician {
	status := r.Status
	if status == "" {
		status = "ATIVO"
	}
	techType := r.Type
	if techType == "" {
		techType = "PARCERIA"
	}
	vehicle := r.Vehicle
	if vehicle == "" {
		vehicle = "NONE"
	}

	return &Technician{
		FullName:             r.FullName,
		TradeName:            r.TradeName,
		CPF:                  r.CPF,
		CNPJ:                 r.CNPJ,
		RG:                   r.RG,
		Contact:              r.Contact,
		Status:               status,
		Type:                 techType,
		Emails:               r.Emails,
		Phones:               r.Phones,
		MinCallValue:         r.MinCallValue,
		Observation:          r.Observation,
		Street:               r.Street,
		Number:               r.Number,
		Complement:           r.Complement,
		Neighborhood:         r.Neighborhood,
		City:                 r.City,
		State:                r.State,
		ZipCode:              r.ZipCode,
		BankName:             r.BankName,
		Agency:               r.Agency,
		AccountNumber:        r.AccountNumber,
		AccountType:          r.AccountType,
		AccountDigit:         r.AccountDigit,
		AccountHolder:        r.AccountHolder,
		HolderCPF:            r.HolderCPF,
		PixKey:               r.PixKey,
		Skills:               r.Skills,
		KnowledgeDescription: r.KnowledgeDescription,
		EquipmentDescription: r.EquipmentDescription,
		Vehicle:              vehicle,
	}
}
