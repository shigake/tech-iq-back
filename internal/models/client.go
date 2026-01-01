package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Client struct {
	ID                 string         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	FullName           string         `json:"fullName" gorm:"not null;type:varchar(255)" validate:"required"`
	CPF                string         `json:"cpf" gorm:"type:varchar(14);index:idx_clients_cpf,unique,where:cpf <> ''"`
	CNPJ               string         `json:"cnpj" gorm:"type:varchar(18);index:idx_clients_cnpj,unique,where:cnpj <> ''"`
	InscricaoEstadual  string         `json:"inscricaoEstadual" gorm:"type:varchar(20)"`
	Email              string         `json:"email" gorm:"type:varchar(255)"`
	Phone              string         `json:"phone" gorm:"type:varchar(20)"`
	
	// Address (embedded)
	Street       string `json:"street" gorm:"type:varchar(255)"`
	Number       string `json:"number" gorm:"type:varchar(20)"`
	Complement   string `json:"complement" gorm:"type:varchar(100)"`
	Neighborhood string `json:"neighborhood" gorm:"type:varchar(100)"`
	City         string `json:"city" gorm:"type:varchar(100)"`
	State        string `json:"state" gorm:"type:varchar(2)"`
	ZipCode      string `json:"zipCode" gorm:"type:varchar(10)"`
	
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (c *Client) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

func (c *Client) IsPJ() bool {
	return c.CNPJ != ""
}

func (c *Client) GetDocument() string {
	if c.CNPJ != "" {
		return c.CNPJ
	}
	return c.CPF
}

type ClientDTO struct {
	ID       string `json:"id"`
	FullName string `json:"fullName"`
	CPF      string `json:"cpf"`
	CNPJ     string `json:"cnpj"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	City     string `json:"city"`
	State    string `json:"state"`
	IsPJ     bool   `json:"isPJ"`
}

func (c *Client) ToDTO() ClientDTO {
	return ClientDTO{
		ID:       c.ID,
		FullName: c.FullName,
		CPF:      c.CPF,
		CNPJ:     c.CNPJ,
		Email:    c.Email,
		Phone:    c.Phone,
		City:     c.City,
		State:    c.State,
		IsPJ:     c.IsPJ(),
	}
}

type CreateClientRequest struct {
	Name               string `json:"name" validate:"required"`
	CPF                string `json:"cpf"`
	CNPJ               string `json:"cnpj"`
	InscricaoEstadual  string `json:"inscricaoEstadual"`
	Email              string `json:"email"`
	Phone              string `json:"phone"`
	Address            string `json:"address"`
	Street             string `json:"street"`
	Number             string `json:"number"`
	Complement         string `json:"complement"`
	Neighborhood       string `json:"neighborhood"`
	City               string `json:"city"`
	State              string `json:"state"`
	ZipCode            string `json:"zipCode"`
	Type               string `json:"type"` // "PF" or "PJ"
}

func (req *CreateClientRequest) ToClient() *Client {
	client := &Client{
		FullName:           req.Name,
		CPF:                req.CPF,
		CNPJ:               req.CNPJ,
		InscricaoEstadual:  req.InscricaoEstadual,
		Email:              req.Email,
		Phone:              req.Phone,
		Street:             req.Street,
		Number:             req.Number,
		Complement:         req.Complement,
		Neighborhood:       req.Neighborhood,
		City:               req.City,
		State:              req.State,
		ZipCode:            req.ZipCode,
	}
	
	// If address is provided but street is empty, use address as street
	if req.Address != "" && req.Street == "" {
		client.Street = req.Address
	}
	
	return client
}
