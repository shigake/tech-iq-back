package services

import (
	"fmt"
	"log"
	"strings"
	
	"github.com/redis/go-redis/v9"
	"github.com/tech-erp/backend/internal/cache"
	"github.com/tech-erp/backend/internal/models"
	"github.com/tech-erp/backend/internal/repositories"
)

type TechnicianService interface {
	Create(req *models.CreateTechnicianRequest) (*models.Technician, error)
	GetAll(page, size int) (*models.PaginatedResponse, error)
	GetByID(id string) (*models.Technician, error)
	Update(id string, req *models.CreateTechnicianRequest) (*models.Technician, error)
	Delete(id string) error
	Search(query string, page, size int) (*models.PaginatedResponse, error)
	SearchWithFilters(query, status, techType, city, state string, page, size int) (*models.PaginatedResponse, error)
	FindByIDs(idsParam string) (*models.PaginatedResponse, error)
	GetByCity(city string) ([]models.TechnicianDTO, error)
	GetByState(state string) ([]models.TechnicianDTO, error)
	GetCities() ([]string, error)
}

type technicianService struct {
	repo  repositories.TechnicianRepository
	cache *cache.RedisClient
}

func NewTechnicianService(repo repositories.TechnicianRepository, cache *cache.RedisClient) TechnicianService {
	return &technicianService{
		repo:  repo,
		cache: cache,
	}
}

func (s *technicianService) Create(req *models.CreateTechnicianRequest) (*models.Technician, error) {
	technician := req.ToModel()
	if err := s.repo.Create(technician); err != nil {
		return nil, err
	}
	
	// Invalidate cache patterns after creating
	s.invalidateTechnicianCaches()
	
	return technician, nil
}

func (s *technicianService) GetAll(page, size int) (*models.PaginatedResponse, error) {
	fmt.Printf(">>> GetAll called: page=%d, size=%d, cache_enabled=%v\n", page, size, s.cache != nil)
	
	// Try cache first
	cacheKey := cache.TechnicianCacheKey(page, size, "")
	if s.cache != nil {
		var cachedResult models.PaginatedResponse
		cacheErr := s.cache.Get(cacheKey, &cachedResult)
		if cacheErr == nil {
			log.Printf("Cache HIT: %s", cacheKey)
			return &cachedResult, nil
		}
		if cacheErr != redis.Nil {
			log.Printf("Cache error: %v", cacheErr)
		}
	}

	// Cache miss, get from database
	technicians, total, err := s.repo.FindAll(page, size)
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	dtos := make([]models.TechnicianDTO, len(technicians))
	for i, t := range technicians {
		dtos[i] = t.ToDTO()
	}

	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	result := &models.PaginatedResponse{
		Content:       dtos,
		Page:          page,
		Size:          size,
		TotalElements: total,
		TotalPages:    totalPages,
	}

	// Cache the result
	if s.cache != nil {
		if err := s.cache.Set(cacheKey, result, cache.TechnicianListTTL); err != nil {
			log.Printf("Failed to cache result: %v", err)
		} else {
			log.Printf("Cache SET: %s", cacheKey)
		}
	}

	return result, nil
}

func (s *technicianService) Update(id string, req *models.CreateTechnicianRequest) (*models.Technician, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	existing.FullName = req.FullName
	existing.TradeName = req.TradeName
	existing.CPF = req.CPF
	existing.CNPJ = req.CNPJ
	existing.RG = req.RG
	existing.Contact = req.Contact
	existing.Status = req.Status
	existing.Type = req.Type
	existing.Emails = req.Emails
	existing.Phones = req.Phones
	existing.MinCallValue = req.MinCallValue
	existing.Observation = req.Observation
	existing.Street = req.Street
	existing.Number = req.Number
	existing.Complement = req.Complement
	existing.Neighborhood = req.Neighborhood
	existing.City = req.City
	existing.State = req.State
	existing.ZipCode = req.ZipCode
	existing.BankName = req.BankName
	existing.Agency = req.Agency
	existing.AccountNumber = req.AccountNumber
	existing.AccountType = req.AccountType
	existing.AccountDigit = req.AccountDigit
	existing.AccountHolder = req.AccountHolder
	existing.HolderCPF = req.HolderCPF
	existing.PixKey = req.PixKey
	existing.Skills = req.Skills

	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}

	// Invalidate caches after update
	s.invalidateTechnicianCaches()
	if s.cache != nil {
		s.cache.Delete(cache.TechnicianDetailCacheKey(id))
	}

	return existing, nil
}

func (s *technicianService) Delete(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	
	// Invalidate caches after delete
	s.invalidateTechnicianCaches()
	if s.cache != nil {
		s.cache.Delete(cache.TechnicianDetailCacheKey(id))
	}
	
	return nil
}

func (s *technicianService) Search(query string, page, size int) (*models.PaginatedResponse, error) {
	// Try cache first for search queries
	cacheKey := cache.TechnicianCacheKey(page, size, query)
	if s.cache != nil {
		var cachedResult models.PaginatedResponse
		cacheErr := s.cache.Get(cacheKey, &cachedResult)
		if cacheErr == nil {
			log.Printf("Cache HIT: %s", cacheKey)
			return &cachedResult, nil
		}
		if cacheErr != redis.Nil {
			log.Printf("Cache error: %v", cacheErr)
		}
	}

	// Cache miss, search in database
	technicians, total, err := s.repo.Search(query, page, size)
	if err != nil {
		return nil, err
	}

	dtos := make([]models.TechnicianDTO, len(technicians))
	for i, t := range technicians {
		dtos[i] = t.ToDTO()
	}

	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	result := &models.PaginatedResponse{
		Content:       dtos,
		Page:          page,
		Size:          size,
		TotalElements: total,
		TotalPages:    totalPages,
	}

	// Cache the search result
	if s.cache != nil {
		if err := s.cache.Set(cacheKey, result, cache.TechnicianSearchTTL); err != nil {
			log.Printf("Failed to cache search result: %v", err)
		} else {
			log.Printf("Cache SET: %s", cacheKey)
		}
	}

	return result, nil
}

func (s *technicianService) SearchWithFilters(query, status, techType, city, state string, page, size int) (*models.PaginatedResponse, error) {
	// Build cache key with all filter parameters
	cacheKey := fmt.Sprintf("technicians:filter:q=%s:s=%s:t=%s:c=%s:st=%s:p=%d:sz=%d", 
		query, status, techType, city, state, page, size)
	
	if s.cache != nil {
		var cachedResult models.PaginatedResponse
		cacheErr := s.cache.Get(cacheKey, &cachedResult)
		if cacheErr == nil {
			log.Printf("Cache HIT: %s", cacheKey)
			return &cachedResult, nil
		}
		if cacheErr != redis.Nil {
			log.Printf("Cache error: %v", cacheErr)
		}
	}

	// Cache miss, search in database with filters
	technicians, total, err := s.repo.SearchWithFilters(query, status, techType, city, state, page, size)
	if err != nil {
		return nil, err
	}

	dtos := make([]models.TechnicianDTO, len(technicians))
	for i, t := range technicians {
		dtos[i] = t.ToDTO()
	}

	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	result := &models.PaginatedResponse{
		Content:       dtos,
		Page:          page,
		Size:          size,
		TotalElements: total,
		TotalPages:    totalPages,
	}

	// Cache the result
	if s.cache != nil {
		if err := s.cache.Set(cacheKey, result, cache.TechnicianSearchTTL); err != nil {
			log.Printf("Failed to cache filter result: %v", err)
		} else {
			log.Printf("Cache SET: %s", cacheKey)
		}
	}

	return result, nil
}

func (s *technicianService) FindByIDs(idsParam string) (*models.PaginatedResponse, error) {
	// Parse comma-separated IDs
	ids := make([]string, 0)
	if idsParam != "" {
		// Simple split by comma
		for _, id := range strings.Split(idsParam, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				ids = append(ids, trimmed)
			}
		}
	}

	if len(ids) == 0 {
		return &models.PaginatedResponse{
			Content:      []models.TechnicianDTO{},
			TotalPages:   0,
			TotalElements: 0,
			Size:         0,
			Page:         0,
		}, nil
	}

	technicians, err := s.repo.FindByIDs(ids)
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	dtos := make([]models.TechnicianDTO, len(technicians))
	for i, t := range technicians {
		dtos[i] = t.ToDTO()
	}

	return &models.PaginatedResponse{
		Content:       dtos,
		TotalPages:    1,
		TotalElements: int64(len(dtos)),
		Size:          len(dtos),
		Page:         0,
	}, nil
}

func (s *technicianService) GetByID(id string) (*models.Technician, error) {
	// Try cache first
	cacheKey := cache.TechnicianDetailCacheKey(id)
	if s.cache != nil {
		var cachedTechnician models.Technician
		cacheErr := s.cache.Get(cacheKey, &cachedTechnician)
		if cacheErr == nil {
			log.Printf("Cache HIT: %s", cacheKey)
			return &cachedTechnician, nil
		}
		if cacheErr != redis.Nil {
			log.Printf("Cache error: %v", cacheErr)
		}
	}

	// Cache miss, get from database
	technician, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.cache != nil {
		if err := s.cache.Set(cacheKey, technician, cache.TechnicianDetailTTL); err != nil {
			log.Printf("Failed to cache technician: %v", err)
		} else {
			log.Printf("Cache SET: %s", cacheKey)
		}
	}

	return technician, nil
}

func (s *technicianService) GetByCity(city string) ([]models.TechnicianDTO, error) {
	// Try cache first
	cacheKey := cache.TechniciansByCityCacheKey(city)
	if s.cache != nil {
		var cachedResult []models.TechnicianDTO
		cacheErr := s.cache.Get(cacheKey, &cachedResult)
		if cacheErr == nil {
			log.Printf("Cache HIT: %s", cacheKey)
			return cachedResult, nil
		}
		if cacheErr != redis.Nil {
			log.Printf("Cache error: %v", cacheErr)
		}
	}

	// Cache miss, get from database
	technicians, err := s.repo.FindByCity(city)
	if err != nil {
		return nil, err
	}

	dtos := make([]models.TechnicianDTO, len(technicians))
	for i, t := range technicians {
		dtos[i] = t.ToDTO()
	}

	// Cache the result
	if s.cache != nil {
		if err := s.cache.Set(cacheKey, dtos, cache.TechnicianFilterTTL); err != nil {
			log.Printf("Failed to cache city result: %v", err)
		} else {
			log.Printf("Cache SET: %s", cacheKey)
		}
	}

	return dtos, nil
}

func (s *technicianService) GetByState(state string) ([]models.TechnicianDTO, error) {
	// Try cache first
	cacheKey := cache.TechniciansByStateCacheKey(state)
	if s.cache != nil {
		var cachedResult []models.TechnicianDTO
		cacheErr := s.cache.Get(cacheKey, &cachedResult)
		if cacheErr == nil {
			log.Printf("Cache HIT: %s", cacheKey)
			return cachedResult, nil
		}
		if cacheErr != redis.Nil {
			log.Printf("Cache error: %v", cacheErr)
		}
	}

	// Cache miss, get from database
	technicians, err := s.repo.FindByState(state)
	if err != nil {
		return nil, err
	}

	dtos := make([]models.TechnicianDTO, len(technicians))
	for i, t := range technicians {
		dtos[i] = t.ToDTO()
	}

	// Cache the result
	if s.cache != nil {
		if err := s.cache.Set(cacheKey, dtos, cache.TechnicianFilterTTL); err != nil {
			log.Printf("Failed to cache state result: %v", err)
		} else {
			log.Printf("Cache SET: %s", cacheKey)
		}
	}

	return dtos, nil
}

func (s *technicianService) GetCities() ([]string, error) {
	// Try cache first
	cacheKey := "technicians:cities:list"
	if s.cache != nil {
		var cachedCities []string
		cacheErr := s.cache.Get(cacheKey, &cachedCities)
		if cacheErr == nil {
			log.Printf("Cache HIT: %s", cacheKey)
			return cachedCities, nil
		}
		if cacheErr != redis.Nil {
			log.Printf("Cache error: %v", cacheErr)
		}
	}

	// Cache miss, get from database
	cities, err := s.repo.GetDistinctCities()
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.cache != nil {
		if err := s.cache.Set(cacheKey, cities, cache.TechnicianFilterTTL); err != nil {
			log.Printf("Failed to cache cities: %v", err)
		} else {
			log.Printf("Cache SET: %s", cacheKey)
		}
	}

	return cities, nil
}

// invalidateTechnicianCaches clears all technician-related cache entries
func (s *technicianService) invalidateTechnicianCaches() {
	if s.cache == nil {
		return
	}

	// Clear list caches
	if err := s.cache.DeletePattern("technicians:list:*"); err != nil {
		log.Printf("Failed to clear list cache: %v", err)
	}
	
	// Clear search caches
	if err := s.cache.DeletePattern("technicians:search:*"); err != nil {
		log.Printf("Failed to clear search cache: %v", err)
	}
	
	// Clear filter caches
	if err := s.cache.DeletePattern("technicians:city:*"); err != nil {
		log.Printf("Failed to clear city cache: %v", err)
	}
	
	if err := s.cache.DeletePattern("technicians:state:*"); err != nil {
		log.Printf("Failed to clear state cache: %v", err)
	}
	
	// Clear cities list cache
	if err := s.cache.Delete("technicians:cities:list"); err != nil {
		log.Printf("Failed to clear cities list cache: %v", err)
	}
	
	log.Printf("Cache invalidation completed")
}
