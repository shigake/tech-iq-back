package repositories

import (
	"strings"

	"unicode"

	"github.com/shigake/tech-iq-back/internal/models"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"
)

type CityRepository interface {
	FindByName(name string) (*models.City, error)
	FindByNameAndUF(name, uf string) (*models.City, error)
	FindByUF(uf string) ([]models.City, error)
	GetCapitalByUF(uf string) (*models.City, error)
	Count() (int64, error)
	BulkInsert(cities []models.City) error
	DeleteAll() error
}

type cityRepository struct {
	db *gorm.DB
}

func NewCityRepository(db *gorm.DB) CityRepository {
	return &cityRepository{db: db}
}

// normalizeString remove acentos e converte para minúsculas
func normalizeString(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

func (r *cityRepository) FindByName(name string) (*models.City, error) {
	var city models.City
	normalized := normalizeString(name)

	err := r.db.Where("nome_norm = ?", normalized).First(&city).Error
	if err != nil {
		return nil, err
	}
	return &city, nil
}

func (r *cityRepository) FindByNameAndUF(name, uf string) (*models.City, error) {
	var city models.City
	normalized := normalizeString(name)
	uf = strings.ToUpper(strings.TrimSpace(uf))

	err := r.db.Where("nome_norm = ? AND uf = ?", normalized, uf).First(&city).Error
	if err != nil {
		return nil, err
	}
	return &city, nil
}

func (r *cityRepository) FindByUF(uf string) ([]models.City, error) {
	var cities []models.City
	uf = strings.ToUpper(strings.TrimSpace(uf))

	err := r.db.Where("uf = ?", uf).Order("nome ASC").Find(&cities).Error
	if err != nil {
		return nil, err
	}
	return cities, nil
}

func (r *cityRepository) GetCapitalByUF(uf string) (*models.City, error) {
	var city models.City
	uf = strings.ToUpper(strings.TrimSpace(uf))

	err := r.db.Where("uf = ? AND capital = true", uf).First(&city).Error
	if err != nil {
		return nil, err
	}
	return &city, nil
}

func (r *cityRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.City{}).Count(&count).Error
	return count, err
}

func (r *cityRepository) BulkInsert(cities []models.City) error {
	// Insert in batches of 500
	batchSize := 500
	for i := 0; i < len(cities); i += batchSize {
		end := i + batchSize
		if end > len(cities) {
			end = len(cities)
		}
		if err := r.db.Create(cities[i:end]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *cityRepository) DeleteAll() error {
	return r.db.Exec("DELETE FROM cities").Error
}
