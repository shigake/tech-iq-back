package repositories

import (
	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
)

type TechnicianRepository interface {
	Create(technician *models.Technician) error
	FindAll(page, size int) ([]models.Technician, int64, error)
	FindByID(id string) (*models.Technician, error)
	Update(technician *models.Technician) error
	Delete(id string) error
	FindByCity(city string) ([]models.Technician, error)
	FindByState(state string) ([]models.Technician, error)
	Search(query string, page, size int) ([]models.Technician, int64, error)
	SearchWithFilters(query, status, techType, city, state string, page, size int) ([]models.Technician, int64, error)
	CountByStatus(status string) (int64, error)
	CountAll() (int64, error)
	GroupByState() ([]models.TechniciansByState, error)
	FindByIDs(ids []string) ([]models.Technician, error)
	GetDistinctCities() ([]string, error)
	GetRecent(limit int) ([]models.Technician, error)
}

type technicianRepository struct {
	db *gorm.DB
}

func NewTechnicianRepository(db *gorm.DB) TechnicianRepository {
	return &technicianRepository{db: db}
}

func (r *technicianRepository) Create(technician *models.Technician) error {
	return r.db.Create(technician).Error
}

func (r *technicianRepository) FindAll(page, size int) ([]models.Technician, int64, error) {
	var technicians []models.Technician
	var total int64

	r.db.Model(&models.Technician{}).Count(&total)

	offset := page * size
	err := r.db.Offset(offset).Limit(size).Order("full_name ASC").Find(&technicians).Error
	if err != nil {
		return nil, 0, err
	}

	return technicians, total, nil
}

func (r *technicianRepository) FindByID(id string) (*models.Technician, error) {
	var technician models.Technician
	err := r.db.Where("id = ?", id).First(&technician).Error
	if err != nil {
		return nil, err
	}
	return &technician, nil
}

func (r *technicianRepository) Update(technician *models.Technician) error {
	return r.db.Save(technician).Error
}

func (r *technicianRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Technician{}).Error
}

func (r *technicianRepository) FindByCity(city string) ([]models.Technician, error) {
	var technicians []models.Technician
	err := r.db.Where("city ILIKE ?", "%"+city+"%").Find(&technicians).Error
	return technicians, err
}

func (r *technicianRepository) FindByState(state string) ([]models.Technician, error) {
	var technicians []models.Technician
	err := r.db.Where("state = ?", state).Find(&technicians).Error
	return technicians, err
}

func (r *technicianRepository) Search(query string, page, size int) ([]models.Technician, int64, error) {
	var technicians []models.Technician
	var total int64

	searchQuery := "%" + query + "%"
	baseQuery := r.db.Model(&models.Technician{}).Where(
		"full_name ILIKE ? OR trade_name ILIKE ? OR city ILIKE ? OR cpf ILIKE ? OR cnpj ILIKE ?",
		searchQuery, searchQuery, searchQuery, searchQuery, searchQuery,
	)

	baseQuery.Count(&total)

	offset := page * size
	err := baseQuery.Offset(offset).Limit(size).Order("full_name ASC").Find(&technicians).Error
	if err != nil {
		return nil, 0, err
	}

	return technicians, total, nil
}

func (r *technicianRepository) SearchWithFilters(query, status, techType, city, state string, page, size int) ([]models.Technician, int64, error) {
	var technicians []models.Technician
	var total int64

	baseQuery := r.db.Model(&models.Technician{})

	// Apply text search if query is provided
	if query != "" {
		searchQuery := "%" + query + "%"
		baseQuery = baseQuery.Where(
			"full_name ILIKE ? OR trade_name ILIKE ? OR city ILIKE ? OR cpf ILIKE ? OR cnpj ILIKE ?",
			searchQuery, searchQuery, searchQuery, searchQuery, searchQuery,
		)
	}

	// Apply filters
	if status != "" {
		baseQuery = baseQuery.Where("status = ?", status)
	}
	if techType != "" {
		baseQuery = baseQuery.Where("type = ?", techType)
	}
	if city != "" {
		baseQuery = baseQuery.Where("city ILIKE ?", "%"+city+"%")
	}
	if state != "" {
		baseQuery = baseQuery.Where("state = ?", state)
	}

	baseQuery.Count(&total)

	offset := page * size
	err := baseQuery.Offset(offset).Limit(size).Order("full_name ASC").Find(&technicians).Error
	if err != nil {
		return nil, 0, err
	}

	return technicians, total, nil
}

func (r *technicianRepository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Technician{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (r *technicianRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&models.Technician{}).Count(&count).Error
	return count, err
}

func (r *technicianRepository) GroupByState() ([]models.TechniciansByState, error) {
	var result []models.TechniciansByState
	err := r.db.Model(&models.Technician{}).
		Select("state, COUNT(*) as count").
		Where("state IS NOT NULL AND state != ''").
		Group("state").
		Order("count DESC").
		Scan(&result).Error
	return result, err
}

func (r *technicianRepository) FindByIDs(ids []string) ([]models.Technician, error) {
	var technicians []models.Technician
	err := r.db.Where("id IN ?", ids).Find(&technicians).Error
	return technicians, err
}

func (r *technicianRepository) GetDistinctCities() ([]string, error) {
	var cities []string
	err := r.db.Model(&models.Technician{}).
		Select("DISTINCT city").
		Where("city IS NOT NULL AND city != ''").
		Order("city ASC").
		Pluck("city", &cities).Error
	return cities, err
}

func (r *technicianRepository) GetRecent(limit int) ([]models.Technician, error) {
	var technicians []models.Technician
	err := r.db.Order("updated_at DESC").Limit(limit).Find(&technicians).Error
	return technicians, err
}
