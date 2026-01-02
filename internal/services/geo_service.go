package services

import (
	"errors"
	"math"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"time"

	"github.com/google/uuid"
)

type GeoService struct {
	geoRepo          *repositories.GeoRepository
	userRepo         repositories.UserRepository
	hierarchyService *HierarchyService
}

func NewGeoService(geoRepo *repositories.GeoRepository, userRepo repositories.UserRepository, hierarchyService *HierarchyService) *GeoService {
	return &GeoService{
		geoRepo:          geoRepo,
		userRepo:         userRepo,
		hierarchyService: hierarchyService,
	}
}

// CreateLocation cria um registro de localização
func (s *GeoService) CreateLocation(technicianID uuid.UUID, req *models.CreateLocationRequest) (*models.TechnicianLocation, error) {
	// Validar coordenadas
	if err := s.validateCoordinates(req.Latitude, req.Longitude); err != nil {
		return nil, err
	}

	// Verificar rate limit para HEARTBEAT
	if req.EventType == models.EventTypeHeartbeat {
		if limited, err := s.checkRateLimit(technicianID, req.EventType); err != nil {
			return nil, err
		} else if limited {
			return nil, errors.New("rate limited: too many location updates")
		}
	}

	// Criar localização
	location := &models.TechnicianLocation{
		TechnicianID: technicianID,
		TicketID:     req.TicketID,
		EventType:    req.EventType,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		AccuracyM:    req.AccuracyM,
		AltitudeM:    req.AltitudeM,
		SpeedMps:     req.SpeedMps,
		HeadingDeg:   req.HeadingDeg,
		Provider:     req.Provider,
		DeviceTime:   req.DeviceTime,
		ServerTime:   time.Now().UTC(),
		IsMocked:     req.IsMocked,
	}

	if err := s.geoRepo.CreateLocation(location); err != nil {
		return nil, err
	}

	// Atualizar última localização
	go s.updateLastLocation(location)

	return location, nil
}

// CreateBatchLocations cria múltiplas localizações (sync offline)
func (s *GeoService) CreateBatchLocations(technicianID uuid.UUID, req *models.BatchLocationRequest) ([]models.BatchLocationResult, error) {
	results := make([]models.BatchLocationResult, 0, len(req.Locations))

	for _, item := range req.Locations {
		result := models.BatchLocationResult{
			LocalID: item.LocalID,
		}

		// Verificar duplicata
		if item.DeviceTime != nil {
			isDup, err := s.geoRepo.CheckDuplicate(technicianID, item.TicketID, item.EventType, *item.DeviceTime)
			if err != nil {
				result.Status = "error"
				result.Error = err.Error()
				results = append(results, result)
				continue
			}
			if isDup {
				result.Status = "duplicate"
				results = append(results, result)
				continue
			}
		}

		// Criar localização
		location := &models.TechnicianLocation{
			TechnicianID:  technicianID,
			TicketID:      item.TicketID,
			EventType:     item.EventType,
			Latitude:      item.Latitude,
			Longitude:     item.Longitude,
			AccuracyM:     item.AccuracyM,
			AltitudeM:     item.AltitudeM,
			SpeedMps:      item.SpeedMps,
			HeadingDeg:    item.HeadingDeg,
			Provider:      item.Provider,
			DeviceTime:    item.DeviceTime,
			ServerTime:    time.Now().UTC(),
			IsMocked:      item.IsMocked,
			IsOfflineSync: true,
		}

		if err := s.geoRepo.CreateLocation(location); err != nil {
			result.Status = "error"
			result.Error = err.Error()
		} else {
			result.ServerID = location.ID
			result.Status = "created"
			go s.updateLastLocation(location)
		}

		results = append(results, result)
	}

	return results, nil
}

// GetLastLocations obtém as últimas localizações dos técnicos
func (s *GeoService) GetLastLocations(userID uuid.UUID, filter repositories.GeoFilter) ([]models.TechnicianLocationResponse, int64, error) {
	// Obter técnicos que o usuário pode ver (baseado em hierarquia)
	allowedTechnicianIDs, err := s.getVisibleTechnicianIDs(userID)
	if err != nil {
		return nil, 0, err
	}

	if len(allowedTechnicianIDs) == 0 {
		return []models.TechnicianLocationResponse{}, 0, nil
	}

	// Buscar últimas localizações
	lastLocations, total, err := s.geoRepo.GetAllLastLocations(filter)
	if err != nil {
		return nil, 0, err
	}

	// Filtrar apenas técnicos permitidos e montar resposta
	responses := make([]models.TechnicianLocationResponse, 0)
	allowedMap := make(map[uuid.UUID]bool)
	for _, id := range allowedTechnicianIDs {
		allowedMap[id] = true
	}

	for _, loc := range lastLocations {
		if !allowedMap[loc.TechnicianID] {
			continue
		}

		response := models.TechnicianLocationResponse{
			TechnicianID: loc.TechnicianID,
			TicketID:     loc.TicketID,
		}

		if loc.Technician != nil {
			response.Name = loc.Technician.FullName
			// response.AvatarURL = loc.Technician.AvatarURL
		}

		if loc.StatusSnapshot != nil {
			response.Status = *loc.StatusSnapshot
		}

		response.Location = &models.LocationInfo{
			Latitude:   loc.Latitude,
			Longitude:  loc.Longitude,
			AccuracyM:  loc.AccuracyM,
			ServerTime: loc.ServerTime,
			EventType:  loc.EventType,
			MinutesAgo: int(time.Since(loc.ServerTime).Minutes()),
		}

		responses = append(responses, response)
	}

	return responses, total, nil
}

// GetTechnicianHistory obtém o histórico de localizações de um técnico
func (s *GeoService) GetTechnicianHistory(userID, technicianID uuid.UUID, filter repositories.HistoryFilter) (*models.TechnicianHistoryResponse, int64, error) {
	// Verificar se o usuário pode ver este técnico
	canView, err := s.canViewTechnician(userID, technicianID)
	if err != nil {
		return nil, 0, err
	}
	if !canView {
		return nil, 0, errors.New("access denied: cannot view this technician's history")
	}

	// Buscar técnico
	technician, err := s.userRepo.GetByID(technicianID)
	if err != nil {
		return nil, 0, err
	}

	// Buscar histórico
	locations, total, err := s.geoRepo.GetLocationHistory(technicianID, filter)
	if err != nil {
		return nil, 0, err
	}

	// Buscar resumo
	summary, err := s.geoRepo.GetHistorySummary(technicianID, filter.From, filter.To)
	if err != nil {
		return nil, 0, err
	}

	// Montar resposta
	response := &models.TechnicianHistoryResponse{
		TechnicianID:   technicianID,
		TechnicianName: technician.FullName,
		Period: models.PeriodInfo{
			From: filter.From,
			To:   filter.To,
		},
		Summary:   *summary,
		Locations: make([]models.LocationHistoryItem, 0, len(locations)),
	}

	for _, loc := range locations {
		item := models.LocationHistoryItem{
			ID:         loc.ID,
			EventType:  loc.EventType,
			TicketID:   loc.TicketID,
			Latitude:   loc.Latitude,
			Longitude:  loc.Longitude,
			AccuracyM:  loc.AccuracyM,
			ServerTime: loc.ServerTime,
		}
		response.Locations = append(response.Locations, item)
	}

	return response, total, nil
}

// GetTicketLocations obtém as localizações de um ticket
func (s *GeoService) GetTicketLocations(ticketID uuid.UUID) (*models.TicketLocationsResponse, error) {
	locations, err := s.geoRepo.GetTicketLocations(ticketID)
	if err != nil {
		return nil, err
	}

	response := &models.TicketLocationsResponse{
		TicketID:   ticketID,
		Heartbeats: make([]models.HeartbeatInfo, 0),
	}

	for _, loc := range locations {
		switch loc.EventType {
		case models.EventTypeCheckin:
			response.Checkin = &models.CheckinoutInfo{
				TechnicianID: loc.TechnicianID,
				Latitude:     loc.Latitude,
				Longitude:    loc.Longitude,
				AccuracyM:    loc.AccuracyM,
				ServerTime:   loc.ServerTime,
			}
			if loc.Technician != nil {
				response.Checkin.TechnicianName = loc.Technician.FullName
			}
		case models.EventTypeCheckout:
			response.Checkout = &models.CheckinoutInfo{
				TechnicianID: loc.TechnicianID,
				Latitude:     loc.Latitude,
				Longitude:    loc.Longitude,
				AccuracyM:    loc.AccuracyM,
				ServerTime:   loc.ServerTime,
			}
			if loc.Technician != nil {
				response.Checkout.TechnicianName = loc.Technician.FullName
			}
		case models.EventTypeHeartbeat:
			response.Heartbeats = append(response.Heartbeats, models.HeartbeatInfo{
				Latitude:   loc.Latitude,
				Longitude:  loc.Longitude,
				ServerTime: loc.ServerTime,
			})
		}
	}

	return response, nil
}

// GetGeoSettings obtém as configurações de geolocalização
func (s *GeoService) GetGeoSettings() (*models.GeoSettingsResponse, error) {
	settings, err := s.geoRepo.GetAllGeoSettings()
	if err != nil {
		return nil, err
	}

	response := &models.GeoSettingsResponse{
		Global: models.GeoSettingsInfo{
			RetentionDays:        90,
			HeartbeatIntervalMin: 5,
			HeartbeatEnabled:     false,
			RequireLocationCheckin: false,
		},
		Scopes: make([]models.ScopeGeoSettings, 0),
	}

	for _, s := range settings {
		if s.ScopeID == nil {
			response.Global = models.GeoSettingsInfo{
				RetentionDays:        s.RetentionDays,
				HeartbeatIntervalMin: s.HeartbeatIntervalMin,
				HeartbeatEnabled:     s.HeartbeatEnabled,
				RequireLocationCheckin: s.RequireLocationCheckin,
			}
		} else {
			response.Scopes = append(response.Scopes, models.ScopeGeoSettings{
				ScopeID:              *s.ScopeID,
				RetentionDays:        s.RetentionDays,
				HeartbeatIntervalMin: s.HeartbeatIntervalMin,
				HeartbeatEnabled:     s.HeartbeatEnabled,
				RequireLocationCheckin: s.RequireLocationCheckin,
			})
		}
	}

	return response, nil
}

// UpdateGeoSettings atualiza as configurações
func (s *GeoService) UpdateGeoSettings(req *models.UpdateGeoSettingsRequest) error {
	settings, err := s.geoRepo.GetGeoSettings(req.ScopeID)
	if err != nil {
		// Criar novo se não existir
		settings = &models.GeoSettings{
			ScopeID: req.ScopeID,
		}
	}

	if req.RetentionDays != nil {
		settings.RetentionDays = *req.RetentionDays
	}
	if req.HeartbeatIntervalMin != nil {
		settings.HeartbeatIntervalMin = *req.HeartbeatIntervalMin
	}
	if req.HeartbeatEnabled != nil {
		settings.HeartbeatEnabled = *req.HeartbeatEnabled
	}
	if req.RequireLocationCheckin != nil {
		settings.RequireLocationCheckin = *req.RequireLocationCheckin
	}

	return s.geoRepo.UpsertGeoSettings(settings)
}

// CleanupOldLocations remove localizações antigas
func (s *GeoService) CleanupOldLocations() (int64, error) {
	settings, err := s.geoRepo.GetGeoSettings(nil)
	if err != nil {
		return 0, err
	}
	return s.geoRepo.DeleteOldLocations(settings.RetentionDays)
}

// Helpers

func (s *GeoService) validateCoordinates(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
}

func (s *GeoService) checkRateLimit(technicianID uuid.UUID, eventType models.EventType) (bool, error) {
	since := time.Now().Add(-time.Minute)
	count, err := s.geoRepo.CountRecentLocations(technicianID, eventType, since)
	if err != nil {
		return false, err
	}
	return count >= 1, nil
}

func (s *GeoService) updateLastLocation(location *models.TechnicianLocation) {
	technician, err := s.userRepo.GetByID(location.TechnicianID)
	if err != nil {
		return
	}

	var statusSnapshot *string
	if technician != nil {
		status := technician.Status
		statusSnapshot = &status
	}

	lastLoc := &models.TechnicianLastLocation{
		TechnicianID:   location.TechnicianID,
		Latitude:       location.Latitude,
		Longitude:      location.Longitude,
		AccuracyM:      location.AccuracyM,
		EventType:      location.EventType,
		TicketID:       location.TicketID,
		StatusSnapshot: statusSnapshot,
		DeviceTime:     location.DeviceTime,
		ServerTime:     location.ServerTime,
	}

	s.geoRepo.UpsertLastLocation(lastLoc)
}

func (s *GeoService) getVisibleTechnicianIDs(userID uuid.UUID) ([]uuid.UUID, error) {
	// Por enquanto, retorna todos os técnicos
	// TODO: Implementar filtro por hierarquia
	users, err := s.userRepo.GetAll()
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(users))
	for _, u := range users {
		if u.Role == "technician" {
			ids = append(ids, u.ID)
		}
	}
	return ids, nil
}

func (s *GeoService) canViewTechnician(userID, technicianID uuid.UUID) (bool, error) {
	// TODO: Implementar verificação por hierarquia
	return true, nil
}

// CalculateDistance calcula a distância entre dois pontos em metros (Haversine)
func CalculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000 // metros

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
