package repositories

import (
	"github.com/tech-erp/backend/internal/models"
	"gorm.io/gorm"
)

type ClientRepository interface {
	Create(client *models.Client) error
	GetByID(id string) (*models.Client, error)
	GetAll(page, size int) ([]models.Client, int64, error)
	Update(client *models.Client) error
	Delete(id string) error
	GetByDocument(cpf, cnpj string) (*models.Client, error)
	Search(query string, page, size int) ([]models.Client, int64, error)
	Count() (int64, error)
}

type clientRepository struct {
	db *gorm.DB
}

func NewClientRepository(db *gorm.DB) ClientRepository {
	return &clientRepository{db: db}
}

func (r *clientRepository) Create(client *models.Client) error {
	return r.db.Create(client).Error
}

func (r *clientRepository) GetByID(id string) (*models.Client, error) {
	var client models.Client
	if err := r.db.First(&client, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) GetAll(page, size int) ([]models.Client, int64, error) {
	var clients []models.Client
	var total int64

	r.db.Model(&models.Client{}).Count(&total)

	offset := page * size
	if err := r.db.Offset(offset).Limit(size).Order("id DESC").Find(&clients).Error; err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}

func (r *clientRepository) Update(client *models.Client) error {
	return r.db.Save(client).Error
}

func (r *clientRepository) Delete(id string) error {
	return r.db.Delete(&models.Client{}, "id = ?", id).Error
}

func (r *clientRepository) GetByDocument(cpf, cnpj string) (*models.Client, error) {
	var client models.Client
	query := r.db
	if cpf != "" {
		query = query.Where("cpf = ?", cpf)
	}
	if cnpj != "" {
		query = query.Or("cnpj = ?", cnpj)
	}
	if err := query.First(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) Search(query string, page, size int) ([]models.Client, int64, error) {
	var clients []models.Client
	var total int64

	searchQuery := "%" + query + "%"
	baseQuery := r.db.Model(&models.Client{}).Where(
		"full_name ILIKE ? OR cpf ILIKE ? OR cnpj ILIKE ? OR email ILIKE ?",
		searchQuery, searchQuery, searchQuery, searchQuery,
	)

	baseQuery.Count(&total)

	offset := page * size
	if err := baseQuery.Offset(offset).Limit(size).Order("id DESC").Find(&clients).Error; err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}

func (r *clientRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.Client{}).Count(&count).Error
	return count, err
}
