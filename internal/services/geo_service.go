package services

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shigake/tech-iq-back/internal/cache"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type GeoService struct {
	geoRepo          *repositories.GeoRepository
	userRepo         repositories.UserRepository
	technicianRepo   repositories.TechnicianRepository
	ticketRepo       repositories.TicketRepository
	hierarchyService *HierarchyService
	redisClient      *cache.RedisClient
}

func NewGeoService(geoRepo *repositories.GeoRepository, userRepo repositories.UserRepository, technicianRepo repositories.TechnicianRepository, ticketRepo repositories.TicketRepository, hierarchyService *HierarchyService, redisClient *cache.RedisClient) *GeoService {
	svc := &GeoService{
		geoRepo:          geoRepo,
		userRepo:         userRepo,
		technicianRepo:   technicianRepo,
		ticketRepo:       ticketRepo,
		hierarchyService: hierarchyService,
		redisClient:      redisClient,
	}

	// Carregar cache de técnicos em background
	go svc.loadTechniciansToCache()

	return svc
}

// CreateLocation cria um registro de localização
// userID é o ID do usuário logado (do JWT), não do técnico
func (s *GeoService) CreateLocation(userID string, req *models.CreateLocationRequest) (*models.TechnicianLocation, error) {
	// Buscar técnico pelo userID
	technician, err := s.technicianRepo.FindByUserID(userID)
	if err != nil || technician == nil {
		return nil, errors.New("usuário não está vinculado a um técnico. Cadastre um técnico com este usuário primeiro")
	}
	technicianID := technician.ID

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

	// Validar check-in: não permitir se já existe um aberto para este ticket
	if req.EventType == models.EventTypeCheckin && req.TicketID != nil {
		hasOpen, existingCheckin, err := s.geoRepo.HasOpenCheckin(*req.TicketID)
		if err != nil {
			return nil, err
		}
		if hasOpen {
			// Retornar erro informativo
			return nil, fmt.Errorf("já existe um check-in aberto para este ticket (técnico: %s, desde: %s)",
				existingCheckin.TechnicianID, existingCheckin.ServerTime.Format("02/01/2006 15:04"))
		}
	}

	// Validar checkout: só permitir se existe check-in aberto
	if req.EventType == models.EventTypeCheckout && req.TicketID != nil {
		hasOpen, _, err := s.geoRepo.HasOpenCheckin(*req.TicketID)
		if err != nil {
			return nil, err
		}
		if !hasOpen {
			return nil, errors.New("não é possível fazer check-out sem um check-in aberto")
		}
	}

	// Converter DeviceTime de FlexibleTime para *time.Time
	var deviceTime *time.Time
	if req.DeviceTime != nil {
		deviceTime = req.DeviceTime.ToTime()
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
		DeviceTime:   deviceTime,
		ServerTime:   time.Now().UTC(),
		IsMocked:     req.IsMocked,
	}

	if err := s.geoRepo.CreateLocation(location); err != nil {
		return nil, err
	}

	// Atualizar status do ticket automaticamente
	if req.TicketID != nil {
		ticketID := req.TicketID.String()
		switch req.EventType {
		case models.EventTypeCheckin:
			// Check-in: mudar status para EM_ATENDIMENTO
			if err := s.ticketRepo.UpdateStatus(ticketID, string(models.TicketStatusInProgress)); err != nil {
				log.Printf("Erro ao atualizar status do ticket %s para EM_ATENDIMENTO: %v", ticketID, err)
			}
		case models.EventTypeCheckout:
			// Check-out: mudar status para PARA_FECHAMENTO
			if err := s.ticketRepo.UpdateStatus(ticketID, string(models.TicketStatusForClosing)); err != nil {
				log.Printf("Erro ao atualizar status do ticket %s para PARA_FECHAMENTO: %v", ticketID, err)
			}
		}
	}

	// Atualizar última localização
	go s.updateLastLocation(location)

	return location, nil
}

// CreateBatchLocations cria múltiplas localizações (sync offline)
// userID é o ID do usuário logado (do JWT), não do técnico
func (s *GeoService) CreateBatchLocations(userID string, req *models.BatchLocationRequest) ([]models.BatchLocationResult, error) {
	// Buscar técnico pelo userID
	technician, err := s.technicianRepo.FindByUserID(userID)
	if err != nil || technician == nil {
		return nil, errors.New("usuário não está vinculado a um técnico. Cadastre um técnico com este usuário primeiro")
	}
	technicianID := technician.ID

	results := make([]models.BatchLocationResult, 0, len(req.Locations))

	for _, item := range req.Locations {
		result := models.BatchLocationResult{
			LocalID: item.LocalID,
		}

		// Converter DeviceTime de FlexibleTime para *time.Time
		var deviceTime *time.Time
		if item.DeviceTime != nil {
			deviceTime = item.DeviceTime.ToTime()
		}

		// Verificar duplicata
		if deviceTime != nil {
			isDup, err := s.geoRepo.CheckDuplicate(technicianID, item.TicketID, item.EventType, *deviceTime)
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
			DeviceTime:    deviceTime,
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

// GetLastLocations obtém as últimas localizações de TODOS os técnicos (do cache Redis)
func (s *GeoService) GetLastLocations(userID uuid.UUID, filter repositories.GeoFilter) ([]models.TechnicianLocationResponse, int64, error) {
	// Verificar se o usuário é admin
	user, err := s.userRepo.FindByID(userID.String())
	if err != nil {
		return nil, 0, err
	}

	// Por enquanto, apenas admins podem ver
	if user.Role != "ADMIN" && user.Role != "admin" {
		return []models.TechnicianLocationResponse{}, 0, nil
	}

	// Buscar do cache Redis (ou fallback para banco)
	allTechnicians, err := s.GetAllTechniciansFromCache()
	if err != nil {
		return nil, 0, err
	}

	// Aplicar filtros
	filtered := make([]cache.TechnicianGeoData, 0)
	for _, tech := range allTechnicians {
		// Filtro por status
		if filter.Status != "" && tech.Status != filter.Status {
			continue
		}
		// Filtro por estado (UF)
		if filter.State != "" && !strings.EqualFold(tech.State, filter.State) {
			continue
		}
		// Filtro por cidade
		if filter.City != "" && !containsIgnoreCase(tech.City, filter.City) {
			continue
		}
		// Filtro por busca (nome)
		if filter.Query != "" {
			// Busca case-insensitive no nome
			if !containsIgnoreCase(tech.Name, filter.Query) {
				continue
			}
		}
		filtered = append(filtered, tech)
	}

	total := int64(len(filtered))

	// Aplicar paginação
	start := filter.Offset
	end := start + filter.Limit
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}
	if filter.Limit == 0 {
		end = len(filtered) // Sem limite = retorna todos
	}

	paginated := filtered[start:end]

	// Montar resposta
	responses := make([]models.TechnicianLocationResponse, 0, len(paginated))

	for _, tech := range paginated {
		response := models.TechnicianLocationResponse{
			TechnicianID: tech.TechnicianID,
			Name:         tech.Name,
			City:         tech.City,
			State:        tech.State,
			Street:       tech.Street,
			Number:       tech.Number,
			Neighborhood: tech.Neighborhood,
			Status:       tech.Status,
		}

		response.Location = &models.LocationInfo{
			Latitude:  tech.Latitude,
			Longitude: tech.Longitude,
			AccuracyM: tech.AccuracyM,
			EventType: models.EventType(tech.EventType),
		}

		if tech.LastUpdateTime != nil {
			serverTime := time.Unix(*tech.LastUpdateTime, 0)
			response.Location.ServerTime = serverTime
			response.Location.MinutesAgo = int(time.Since(serverTime).Minutes())
		}

		// Flag para indicar se tem localização real ou estimada
		if !tech.HasRealLocation {
			response.Location.MinutesAgo = -1 // Indica que é localização estimada
		}

		responses = append(responses, response)
	}

	return responses, total, nil
}

// containsIgnoreCase verifica se a string contém a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// GetTechnicianHistory obtém o histórico de localizações de um técnico
func (s *GeoService) GetTechnicianHistory(userID uuid.UUID, technicianID string, filter repositories.HistoryFilter) (*models.TechnicianHistoryResponse, int64, error) {
	// Buscar técnico
	technician, err := s.technicianRepo.FindByID(technicianID)
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
		Sessions:   make([]models.AttendanceSession, 0),
	}

	// Mapa para agrupar sessões por técnico + ordem temporal
	var sessions []models.AttendanceSession
	var currentSession *models.AttendanceSession
	sessionCount := 0

	for _, loc := range locations {
		switch loc.EventType {
		case models.EventTypeCheckin:
			checkinInfo := &models.CheckinoutInfo{
				TechnicianID: loc.TechnicianID,
				Latitude:     loc.Latitude,
				Longitude:    loc.Longitude,
				AccuracyM:    loc.AccuracyM,
				ServerTime:   loc.ServerTime,
			}
			if loc.Technician != nil {
				checkinInfo.TechnicianName = loc.Technician.FullName
			}

			// Manter compatibilidade: último check-in como principal
			response.Checkin = checkinInfo

			// Criar nova sessão
			sessionCount++
			techName := ""
			if loc.Technician != nil {
				techName = loc.Technician.FullName
			}
			currentSession = &models.AttendanceSession{
				ID:             fmt.Sprintf("%s-%d", loc.TechnicianID, sessionCount),
				TechnicianID:   loc.TechnicianID,
				TechnicianName: techName,
				Checkin:        checkinInfo,
				Status:         "in_progress",
			}
			sessions = append(sessions, *currentSession)

		case models.EventTypeCheckout:
			checkoutInfo := &models.CheckinoutInfo{
				TechnicianID: loc.TechnicianID,
				Latitude:     loc.Latitude,
				Longitude:    loc.Longitude,
				AccuracyM:    loc.AccuracyM,
				ServerTime:   loc.ServerTime,
			}
			if loc.Technician != nil {
				checkoutInfo.TechnicianName = loc.Technician.FullName
			}

			// Manter compatibilidade: último checkout como principal
			response.Checkout = checkoutInfo

			// Fechar a última sessão aberta do mesmo técnico
			for i := len(sessions) - 1; i >= 0; i-- {
				if sessions[i].TechnicianID == loc.TechnicianID && sessions[i].Checkout == nil {
					sessions[i].Checkout = checkoutInfo
					sessions[i].Status = "completed"
					// Calcular duração
					duration := int64(checkoutInfo.ServerTime.Sub(sessions[i].Checkin.ServerTime).Seconds())
					sessions[i].Duration = &duration
					break
				}
			}

		case models.EventTypeHeartbeat:
			response.Heartbeats = append(response.Heartbeats, models.HeartbeatInfo{
				Latitude:   loc.Latitude,
				Longitude:  loc.Longitude,
				ServerTime: loc.ServerTime,
			})
		}
	}

	response.Sessions = sessions
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
			RetentionDays:          90,
			HeartbeatIntervalMin:   5,
			HeartbeatEnabled:       false,
			RequireLocationCheckin: false,
		},
		Scopes: make([]models.ScopeGeoSettings, 0),
	}

	for _, s := range settings {
		if s.ScopeID == nil {
			response.Global = models.GeoSettingsInfo{
				RetentionDays:          s.RetentionDays,
				HeartbeatIntervalMin:   s.HeartbeatIntervalMin,
				HeartbeatEnabled:       s.HeartbeatEnabled,
				RequireLocationCheckin: s.RequireLocationCheckin,
			}
		} else {
			response.Scopes = append(response.Scopes, models.ScopeGeoSettings{
				ScopeID:                *s.ScopeID,
				RetentionDays:          s.RetentionDays,
				HeartbeatIntervalMin:   s.HeartbeatIntervalMin,
				HeartbeatEnabled:       s.HeartbeatEnabled,
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

func (s *GeoService) checkRateLimit(technicianID string, eventType models.EventType) (bool, error) {
	since := time.Now().Add(-time.Minute)
	count, err := s.geoRepo.CountRecentLocations(technicianID, eventType, since)
	if err != nil {
		return false, err
	}
	return count >= 1, nil
}

func (s *GeoService) updateLastLocation(location *models.TechnicianLocation) {
	technician, err := s.technicianRepo.FindByID(location.TechnicianID)
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

	// Atualizar também no Redis cache
	go s.updateTechnicianInCache(location.TechnicianID)
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

// loadTechniciansToCache carrega todos os técnicos no Redis cache
func (s *GeoService) loadTechniciansToCache() {
	if s.redisClient == nil {
		log.Println("⚠️ Redis client not available, skipping geo cache load")
		return
	}

	// Verificar se já foi carregado recentemente
	if s.redisClient.IsGeoCacheLoaded() {
		count, _ := s.redisClient.GetGeoCacheCount()
		log.Printf("✅ Geo cache already loaded with %d technicians", count)
		return
	}

	log.Println("🔄 Loading all technicians to geo cache...")

	// Buscar todos os técnicos ativos
	technicians, err := s.technicianRepo.GetAll()
	if err != nil {
		log.Printf("❌ Error loading technicians: %v", err)
		return
	}

	// Buscar últimas localizações conhecidas
	lastLocations, _, _ := s.geoRepo.GetAllLastLocations(repositories.GeoFilter{Limit: 10000})

	// Criar mapa de últimas localizações
	locMap := make(map[string]models.TechnicianLastLocation)
	for _, loc := range lastLocations {
		locMap[loc.TechnicianID] = loc
	}

	// Preparar dados para cache
	geoData := make([]cache.TechnicianGeoData, 0, len(technicians))

	for _, tech := range technicians {
		data := cache.TechnicianGeoData{
			TechnicianID: tech.ID,
			Name:         tech.FullName,
			City:         tech.City,
			State:        tech.State,
			Street:       tech.Street,
			Number:       tech.Number,
			Neighborhood: tech.Neighborhood,
			Status:       tech.Status,
		}

		// Verificar se tem localização real
		if lastLoc, ok := locMap[tech.ID]; ok {
			data.Latitude = lastLoc.Latitude
			data.Longitude = lastLoc.Longitude
			data.AccuracyM = lastLoc.AccuracyM
			data.EventType = string(lastLoc.EventType)
			data.HasRealLocation = true
			if !lastLoc.ServerTime.IsZero() {
				ts := lastLoc.ServerTime.Unix()
				data.LastUpdateTime = &ts
			}
		} else {
			// Usar coordenadas da cidade/estado
			lat, lng, _ := GetCoordinatesForLocation(tech.City, tech.State)
			data.Latitude = lat
			data.Longitude = lng
			data.HasRealLocation = false
		}

		geoData = append(geoData, data)
	}

	// Salvar no Redis
	if err := s.redisClient.SetAllTechniciansGeo(geoData); err != nil {
		log.Printf("❌ Error saving technicians to cache: %v", err)
		return
	}

	log.Printf("✅ Loaded %d technicians to geo cache", len(geoData))
}

// updateTechnicianInCache atualiza um técnico específico no cache quando recebe nova localização
func (s *GeoService) updateTechnicianInCache(technicianID string) {
	if s.redisClient == nil {
		return
	}

	// Buscar técnico
	tech, err := s.technicianRepo.FindByID(technicianID)
	if err != nil || tech == nil {
		return
	}

	// Buscar última localização
	lastLoc, err := s.geoRepo.GetLastLocation(technicianID)
	if err != nil {
		return
	}

	data := cache.TechnicianGeoData{
		TechnicianID:    tech.ID,
		Name:            tech.FullName,
		City:            tech.City,
		State:           tech.State,
		Street:          tech.Street,
		Number:          tech.Number,
		Neighborhood:    tech.Neighborhood,
		Status:          tech.Status,
		Latitude:        lastLoc.Latitude,
		Longitude:       lastLoc.Longitude,
		AccuracyM:       lastLoc.AccuracyM,
		EventType:       string(lastLoc.EventType),
		HasRealLocation: true,
	}

	if !lastLoc.ServerTime.IsZero() {
		ts := lastLoc.ServerTime.Unix()
		data.LastUpdateTime = &ts
	}

	s.redisClient.UpdateTechnicianLocation(data)
}

// GetAllTechniciansFromCache retorna todos os técnicos do cache Redis
// Fallback para o banco de dados se o cache estiver vazio
func (s *GeoService) GetAllTechniciansFromCache() ([]cache.TechnicianGeoData, error) {
	if s.redisClient == nil {
		return s.loadTechniciansDirectly()
	}

	// Tentar buscar do cache
	technicians, err := s.redisClient.GetAllTechniciansGeo()
	if err != nil || len(technicians) == 0 {
		// Cache miss - carregar do banco
		log.Println("⚠️ Geo cache miss, loading from database...")
		go s.loadTechniciansToCache()
		return s.loadTechniciansDirectly()
	}

	return technicians, nil
}

// loadTechniciansDirectly carrega técnicos diretamente do banco (fallback)
func (s *GeoService) loadTechniciansDirectly() ([]cache.TechnicianGeoData, error) {
	technicians, err := s.technicianRepo.GetAll()
	if err != nil {
		return nil, err
	}

	lastLocations, _, _ := s.geoRepo.GetAllLastLocations(repositories.GeoFilter{Limit: 10000})
	locMap := make(map[string]models.TechnicianLastLocation)
	for _, loc := range lastLocations {
		locMap[loc.TechnicianID] = loc
	}

	geoData := make([]cache.TechnicianGeoData, 0, len(technicians))

	for _, tech := range technicians {
		data := cache.TechnicianGeoData{
			TechnicianID: tech.ID,
			Name:         tech.FullName,
			City:         tech.City,
			State:        tech.State,
			Street:       tech.Street,
			Number:       tech.Number,
			Neighborhood: tech.Neighborhood,
			Status:       tech.Status,
		}

		if lastLoc, ok := locMap[tech.ID]; ok {
			data.Latitude = lastLoc.Latitude
			data.Longitude = lastLoc.Longitude
			data.AccuracyM = lastLoc.AccuracyM
			data.EventType = string(lastLoc.EventType)
			data.HasRealLocation = true
			if !lastLoc.ServerTime.IsZero() {
				ts := lastLoc.ServerTime.Unix()
				data.LastUpdateTime = &ts
			}
		} else {
			lat, lng, _ := GetCoordinatesForLocation(tech.City, tech.State)
			data.Latitude = lat
			data.Longitude = lng
			data.HasRealLocation = false
		}

		geoData = append(geoData, data)
	}

	return geoData, nil
}

// RefreshGeoCache força a recarga do cache de geolocalização
func (s *GeoService) RefreshGeoCache() error {
	if s.redisClient == nil {
		return errors.New("redis client not available")
	}

	// Deletar flag de cache carregado para forçar recarga
	s.redisClient.Delete(cache.GeoCacheLoadedKey)

	// Recarregar
	s.loadTechniciansToCache()

	return nil
}
