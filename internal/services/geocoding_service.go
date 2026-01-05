package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

// NominatimResponse represents the response from Nominatim API
type NominatimResponse struct {
	PlaceID     int64   `json:"place_id"`
	Licence     string  `json:"licence"`
	OsmType     string  `json:"osm_type"`
	OsmID       int64   `json:"osm_id"`
	Lat         string  `json:"lat"`
	Lon         string  `json:"lon"`
	DisplayName string  `json:"display_name"`
	Class       string  `json:"class"`
	Type        string  `json:"type"`
	Importance  float64 `json:"importance"`
}

// GeocodingResult represents the result of a geocoding operation
type GeocodingResult struct {
	TechnicianID   string   `json:"technicianId"`
	TechnicianName string   `json:"technicianName"`
	Address        string   `json:"address"`
	Success        bool     `json:"success"`
	Latitude       *float64 `json:"latitude,omitempty"`
	Longitude      *float64 `json:"longitude,omitempty"`
	Error          string   `json:"error,omitempty"`
}

// GeocodingStats represents statistics from a batch geocoding operation
type GeocodingStats struct {
	TotalProcessed  int               `json:"totalProcessed"`
	SuccessCount    int               `json:"successCount"`
	FailedCount     int               `json:"failedCount"`
	SkippedCount    int               `json:"skippedCount"`
	AlreadyGeocoded int               `json:"alreadyGeocoded"`
	Results         []GeocodingResult `json:"results"`
	StartTime       time.Time         `json:"startTime"`
	EndTime         time.Time         `json:"endTime"`
	Duration        string            `json:"duration"`
}

// GeocodingService handles geocoding operations using Nominatim
type GeocodingService struct {
	technicianRepo repositories.TechnicianRepository
	httpClient     *http.Client
	rateLimiter    *time.Ticker
	mu             sync.Mutex
	isRunning      bool
}

// NewGeocodingService creates a new geocoding service
func NewGeocodingService(technicianRepo repositories.TechnicianRepository) *GeocodingService {
	return &GeocodingService{
		technicianRepo: technicianRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		// Nominatim requires max 1 request per second
		rateLimiter: time.NewTicker(1100 * time.Millisecond),
	}
}

// buildAddress creates a formatted address string for geocoding
func (s *GeocodingService) buildAddress(t *models.Technician) string {
	parts := []string{}

	if t.Street != "" {
		street := t.Street
		if t.Number != "" {
			street += ", " + t.Number
		}
		parts = append(parts, street)
	}

	if t.Neighborhood != "" {
		parts = append(parts, t.Neighborhood)
	}

	if t.City != "" {
		parts = append(parts, t.City)
	}

	if t.State != "" {
		parts = append(parts, t.State)
	}

	// Always add Brazil to improve accuracy
	parts = append(parts, "Brasil")

	return strings.Join(parts, ", ")
}

// geocodeAddress calls Nominatim API to geocode an address
func (s *GeocodingService) geocodeAddress(address string) (*float64, *float64, error) {
	// Wait for rate limiter
	<-s.rateLimiter.C

	// Build request URL
	baseURL := "https://nominatim.openstreetmap.org/search"
	params := url.Values{}
	params.Add("q", address)
	params.Add("format", "json")
	params.Add("limit", "1")
	params.Add("countrycodes", "br")

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create request with proper headers (Nominatim requires User-Agent)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "TechERP/1.0 (contact@techerp.com)")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("nominatim returned status %d", resp.StatusCode)
	}

	// Parse response
	var results []NominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(results) == 0 {
		return nil, nil, fmt.Errorf("no results found for address")
	}

	// Parse coordinates
	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse latitude: %w", err)
	}

	lon, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse longitude: %w", err)
	}

	return &lat, &lon, nil
}

// GeocodeTechnician geocodes a single technician's address
func (s *GeocodingService) GeocodeTechnician(technician *models.Technician) (*GeocodingResult, error) {
	result := &GeocodingResult{
		TechnicianID:   technician.ID,
		TechnicianName: technician.FullName,
	}

	// Check if technician has an address to geocode
	if technician.Street == "" && technician.City == "" {
		result.Success = false
		result.Error = "no address available"
		return result, nil
	}

	// Build address
	address := s.buildAddress(technician)
	result.Address = address

	// Geocode
	lat, lon, err := s.geocodeAddress(address)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}

	// Update technician
	technician.Latitude = lat
	technician.Longitude = lon

	if err := s.technicianRepo.Update(technician); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to update technician: %v", err)
		return result, nil
	}

	result.Success = true
	result.Latitude = lat
	result.Longitude = lon

	return result, nil
}

// GeocodeTechnicianByID geocodes a technician by their ID
func (s *GeocodingService) GeocodeTechnicianByID(technicianID string) (*GeocodingResult, error) {
	// Find the technician
	technician, err := s.technicianRepo.FindByID(technicianID)
	if err != nil {
		return nil, fmt.Errorf("failed to find technician: %w", err)
	}
	if technician == nil {
		return nil, nil // Not found
	}

	return s.GeocodeTechnician(technician)
}

// GeocodeAllTechnicians geocodes all technicians that have addresses but no coordinates
func (s *GeocodingService) GeocodeAllTechnicians(forceAll bool) (*GeocodingStats, error) {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return nil, fmt.Errorf("geocoding already in progress")
	}
	s.isRunning = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
	}()

	stats := &GeocodingStats{
		StartTime: time.Now(),
		Results:   []GeocodingResult{},
	}

	// Get all technicians (page 0 for offset 0)
	technicians, _, err := s.technicianRepo.FindAll(0, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to get technicians: %w", err)
	}

	log.Printf("[GEOCODING] Starting batch geocoding for %d technicians (forceAll=%v)", len(technicians), forceAll)

	for _, tech := range technicians {
		stats.TotalProcessed++

		// Skip if already geocoded and not forcing
		if !forceAll && tech.Latitude != nil && tech.Longitude != nil {
			stats.AlreadyGeocoded++
			continue
		}

		// Skip if no address
		if tech.Street == "" && tech.City == "" {
			stats.SkippedCount++
			result := GeocodingResult{
				TechnicianID:   tech.ID,
				TechnicianName: tech.FullName,
				Success:        false,
				Error:          "no address available",
			}
			stats.Results = append(stats.Results, result)
			continue
		}

		// Geocode
		result, err := s.GeocodeTechnician(&tech)
		if err != nil {
			log.Printf("[GEOCODING] Error geocoding technician %s: %v", tech.ID, err)
			stats.FailedCount++
		} else if result.Success {
			stats.SuccessCount++
			log.Printf("[GEOCODING] Successfully geocoded %s: %s -> (%f, %f)",
				tech.FullName, result.Address, *result.Latitude, *result.Longitude)
		} else {
			stats.FailedCount++
			log.Printf("[GEOCODING] Failed to geocode %s: %s", tech.FullName, result.Error)
		}

		stats.Results = append(stats.Results, *result)
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime).String()

	log.Printf("[GEOCODING] Batch completed: %d total, %d success, %d failed, %d skipped, %d already geocoded",
		stats.TotalProcessed, stats.SuccessCount, stats.FailedCount, stats.SkippedCount, stats.AlreadyGeocoded)

	return stats, nil
}

// GetGeocodingStatus returns whether a geocoding job is currently running
func (s *GeocodingService) GetGeocodingStatus() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRunning
}
