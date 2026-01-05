package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"unicode"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// CityService handles city operations
type CityService struct {
	cityRepo repositories.CityRepository
}

// NewCityService creates a new city service
func NewCityService(cityRepo repositories.CityRepository) *CityService {
	return &CityService{cityRepo: cityRepo}
}

// IBGECity represents the structure from the IBGE JSON
type IBGECity struct {
	CodigoIBGE int     `json:"codigo_ibge"`
	Nome       string  `json:"nome"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Capital    int     `json:"capital"`
	CodigoUF   int     `json:"codigo_uf"`
	DDD        int     `json:"ddd"`
}

// UF code to abbreviation mapping
var ufCodeToAbbr = map[int]string{
	11: "RO", 12: "AC", 13: "AM", 14: "RR", 15: "PA", 16: "AP", 17: "TO",
	21: "MA", 22: "PI", 23: "CE", 24: "RN", 25: "PB", 26: "PE", 27: "AL", 28: "SE", 29: "BA",
	31: "MG", 32: "ES", 33: "RJ", 35: "SP",
	41: "PR", 42: "SC", 43: "RS",
	50: "MS", 51: "MT", 52: "GO", 53: "DF",
}

// normalizeString removes accents and converts to lowercase
func normalizeCityString(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

// LoadCitiesFromIBGE downloads and populates the cities table
func (s *CityService) LoadCitiesFromIBGE() (int, error) {
	// Check if already populated
	count, err := s.cityRepo.Count()
	if err != nil {
		return 0, fmt.Errorf("failed to count cities: %w", err)
	}

	if count > 5000 {
		log.Printf("[CITIES] Database already has %d cities, skipping load", count)
		return int(count), nil
	}

	log.Println("[CITIES] Downloading cities from IBGE dataset...")

	// Download from GitHub
	url := "https://raw.githubusercontent.com/kelvins/municipios-brasileiros/main/json/municipios.json"
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to download cities: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to download cities: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var ibgeCities []IBGECity
	if err := json.Unmarshal(body, &ibgeCities); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %w", err)
	}

	log.Printf("[CITIES] Downloaded %d cities, converting...", len(ibgeCities))

	// Convert to our model
	cities := make([]models.City, 0, len(ibgeCities))
	for _, ic := range ibgeCities {
		uf, ok := ufCodeToAbbr[ic.CodigoUF]
		if !ok {
			log.Printf("[CITIES] Unknown UF code: %d for city %s", ic.CodigoUF, ic.Nome)
			continue
		}

		city := models.City{
			CodigoIBGE: ic.CodigoIBGE,
			Nome:       ic.Nome,
			NomeNorm:   normalizeCityString(ic.Nome),
			UF:         uf,
			Latitude:   ic.Latitude,
			Longitude:  ic.Longitude,
			Capital:    ic.Capital == 1,
			DDD:        ic.DDD,
		}
		cities = append(cities, city)
	}

	// Clear existing data
	log.Println("[CITIES] Clearing existing cities...")
	if err := s.cityRepo.DeleteAll(); err != nil {
		return 0, fmt.Errorf("failed to clear cities: %w", err)
	}

	// Insert new data
	log.Println("[CITIES] Inserting cities...")
	if err := s.cityRepo.BulkInsert(cities); err != nil {
		return 0, fmt.Errorf("failed to insert cities: %w", err)
	}

	log.Printf("[CITIES] Successfully loaded %d cities", len(cities))
	return len(cities), nil
}

// GetCoordinates returns coordinates for a city/state combination
func (s *CityService) GetCoordinates(cityName, state string) (lat, lng float64, found bool) {
	// Try to find by name and UF
	if cityName != "" && state != "" {
		city, err := s.cityRepo.FindByNameAndUF(cityName, state)
		if err == nil && city != nil {
			return city.Latitude, city.Longitude, true
		}
	}

	// Try just by name
	if cityName != "" {
		city, err := s.cityRepo.FindByName(cityName)
		if err == nil && city != nil {
			return city.Latitude, city.Longitude, true
		}
	}

	// Fallback to state capital
	if state != "" {
		capital, err := s.cityRepo.GetCapitalByUF(state)
		if err == nil && capital != nil {
			return capital.Latitude, capital.Longitude, false
		}
	}

	// Ultimate fallback: Brasília
	return -15.7942, -47.8822, false
}

// GetCount returns the number of cities in the database
func (s *CityService) GetCount() (int64, error) {
	return s.cityRepo.Count()
}
