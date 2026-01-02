package repositories

import (
	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
)

type CategoryRepository interface {
	Create(category *models.Category) error
	GetByID(id string) (*models.Category, error)
	GetAll() ([]models.Category, error)
	GetByType(categoryType models.CategoryType) ([]models.Category, error)
	GetByTypeWithChildren(categoryType models.CategoryType) ([]models.Category, error)
	Update(category *models.Category) error
	Delete(id string) error
	GetByName(name string) (*models.Category, error)
	GetByNameAndType(name string, categoryType models.CategoryType) (*models.Category, error)
}

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(category *models.Category) error {
	return r.db.Create(category).Error
}

func (r *categoryRepository) GetByID(id string) (*models.Category, error) {
	var category models.Category
	if err := r.db.Preload("Children", "active = ?", true).First(&category, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *categoryRepository) GetAll() ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.Where("active = ? AND parent_id IS NULL", true).
		Preload("Children", "active = ?", true).
		Order("sort_order ASC, name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *categoryRepository) GetByType(categoryType models.CategoryType) ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.Where("active = ? AND type = ? AND parent_id IS NULL", true, categoryType).
		Order("sort_order ASC, name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *categoryRepository) GetByTypeWithChildren(categoryType models.CategoryType) ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.Where("active = ? AND type = ? AND parent_id IS NULL", true, categoryType).
		Preload("Children", "active = ?", true).
		Order("sort_order ASC, name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *categoryRepository) Update(category *models.Category) error {
	return r.db.Save(category).Error
}

func (r *categoryRepository) Delete(id string) error {
	return r.db.Delete(&models.Category{}, "id = ?", id).Error
}

func (r *categoryRepository) GetByName(name string) (*models.Category, error) {
	var category models.Category
	if err := r.db.First(&category, "name = ?", name).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *categoryRepository) GetByNameAndType(name string, categoryType models.CategoryType) (*models.Category, error) {
	var category models.Category
	if err := r.db.First(&category, "name = ? AND type = ? AND parent_id IS NULL", name, categoryType).Error; err != nil {
		return nil, err
	}
	return &category, nil
}
