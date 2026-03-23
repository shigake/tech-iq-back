package services

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shigake/tech-iq-back/internal/cache"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

// memCacheTTL define por quanto tempo o slice desserializado fica em memória
// antes de buscar novamente no Redis. Evita HGetAll + JSON decode a cada request.
const memCacheTTL = 30 * time.Second

type GeoService struct {
	geoRepo          *repositories.GeoRepository
	userRepo         repositories.UserRepository
	technicianRepo   repositories.TechnicianRepository
	ticketRepo       repositories.TicketRepository
	hierarchyService *HierarchyService
	redisClient      *cache.RedisClient
	cityService      *CityService

	// in-memory cache: evita HGetAll + desserialização JSON a cada requisição
	memMu        sync.RWMutex
	memCache     []cache.TechnicianGeoData
	memCacheExp  time.Time
}

func NewGeoService(geoRepo *repositories.GeoRepository, userRepo repositories.UserRepository, technicianRepo repositories.TechnicianRepository, ticketRepo repositories.TicketRepository, hierarchyService *HierarchyService, redisClient *cache.RedisClient, cityService *CityService) *GeoService {
	svc := &GeoService{
		geoRepo:          geoRepo,
		userRepo:         userRepo,
		technicianRepo:   technicianRepo,
		ticketRepo:       ticketRepo,
		hierarchyService: hierarchyService,
		redisClient:      redisClient,
		cityService:      cityService,
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
		// Filtro por viewport (bounds do mapa)
		if filter.SwLat != nil && filter.SwLng != nil && filter.NeLat != nil && filter.NeLng != nil {
			if tech.Latitude < *filter.SwLat || tech.Latitude > *filter.NeLat ||
				tech.Longitude < *filter.SwLng || tech.Longitude > *filter.NeLng {
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

// geoGridCellSize retorna o tamanho de célula em graus para um dado nível de zoom
func geoGridCellSize(zoom int) float64 {
	switch {
	case zoom <= 5:
		return 5.0
	case zoom <= 7:
		return 2.0
	case zoom <= 9:
		return 1.0
	case zoom <= 10:
		return 0.5
	default:
		return 0.2
	}
}

// GetClusters retorna clusters de técnicos agrupados por zoom ou técnicos individuais (zoom >= 12)
func (s *GeoService) GetClusters(zoom int, swLat, swLng, neLat, neLng float64, status string) (*models.GeoClustersResponse, error) {
	allTechs, err := s.GetAllTechniciansFromCache()
	if err != nil {
		return nil, err
	}

	hasBounds := swLat != 0 || swLng != 0 || neLat != 0 || neLng != 0

	statusSet := make(map[string]bool)
	if status != "" {
		for _, s := range strings.Split(status, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				statusSet[s] = true
			}
		}
	}

	filtered := make([]cache.TechnicianGeoData, 0, len(allTechs))
	for _, tech := range allTechs {
		if tech.Latitude == 0 && tech.Longitude == 0 {
			continue
		}
		if len(statusSet) > 0 && !statusSet[tech.Status] {
			continue
		}
		if hasBounds {
			if tech.Latitude < swLat || tech.Latitude > neLat ||
				tech.Longitude < swLng || tech.Longitude > neLng {
				continue
			}
		}
		filtered = append(filtered, tech)
	}

	totalCount := len(filtered)

	// Zoom alto (>= 12): retornar técnicos individuais, limitando a 300 por viewport
	// (acima disso o Flutter Web começa a travar ao renderizar markers)
	if zoom >= 12 {
		limit := 300
		if len(filtered) > limit {
			filtered = filtered[:limit]
		}

		points := make([]models.GeoClusterItem, 0, len(filtered))
		for _, tech := range filtered {
			minutesAgo := -1
			if tech.LastUpdateTime != nil {
				minutesAgo = int(time.Since(time.Unix(*tech.LastUpdateTime, 0)).Minutes())
			}
			points = append(points, models.GeoClusterItem{
				Lat:          tech.Latitude,
				Lng:          tech.Longitude,
				Count:        1,
				TechnicianID: tech.TechnicianID,
				Name:         tech.Name,
				Status:       tech.Status,
				City:         tech.City,
				State:        tech.State,
				Phone:        tech.Phone,
				Skills:       tech.Skills,
				HasRealLoc:   tech.HasRealLocation,
				MinutesAgo:   minutesAgo,
			})
		}

		return &models.GeoClustersResponse{
			Clusters:   points,
			TotalCount: totalCount,
			Zoom:       zoom,
			IsExact:    true,
		}, nil
	}

	// Zoom baixo: agrupar em grid cells
	cellSize := geoGridCellSize(zoom)

	type cellKey struct {
		col, row int
	}
	cells := make(map[cellKey][]cache.TechnicianGeoData)
	for _, tech := range filtered {
		col := int(math.Floor(tech.Longitude / cellSize))
		row := int(math.Floor(tech.Latitude / cellSize))
		cells[cellKey{col, row}] = append(cells[cellKey{col, row}], tech)
	}

	clusters := make([]models.GeoClusterItem, 0, len(cells))
	for _, techs := range cells {
		var sumLat, sumLng float64
		for _, t := range techs {
			sumLat += t.Latitude
			sumLng += t.Longitude
		}
		n := len(techs)
		clusters = append(clusters, models.GeoClusterItem{
			Lat:   sumLat / float64(n),
			Lng:   sumLng / float64(n),
			Count: n,
		})
	}

	return &models.GeoClustersResponse{
		Clusters:   clusters,
		TotalCount: totalCount,
		Zoom:       zoom,
		IsExact:    false,
	}, nil
}

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

	// Verificar pelo count real do hash, não apenas pelo flag
	// O flag GeoCacheLoadedKey tem TTL de 24h mas o hash tem 30min.
	// Se o count real for > threshold, o cache está íntegro — pula.
	count, _ := s.redisClient.GetGeoCacheCount()
	if count > 50 {
		log.Printf("✅ Geo cache has %d technicians, skipping reload", count)
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
		var phone string
		if len(tech.Phones) > 0 {
			phone = tech.Phones[0].Number
		}

		var skills []string
		for skill, active := range tech.Skills {
			if active {
				skills = append(skills, skill)
			}
		}

		data := cache.TechnicianGeoData{
			TechnicianID: tech.ID,
			Name:         tech.FullName,
			City:         tech.City,
			State:        tech.State,
			Street:       tech.Street,
			Number:       tech.Number,
			Neighborhood: tech.Neighborhood,
			Status:       tech.Status,
			Phone:        phone,
			Skills:       skills,
		}

		// Prioridade de coordenadas:
		// 1. Última localização GPS real (tracking)
		// 2. Coordenadas geocodificadas do endereço (Latitude/Longitude do técnico)
		// 3. Coordenadas da cidade/estado com offset

		if lastLoc, ok := locMap[tech.ID]; ok {
			// Tem localização GPS real do tracking
			data.Latitude = lastLoc.Latitude
			data.Longitude = lastLoc.Longitude
			data.AccuracyM = lastLoc.AccuracyM
			data.EventType = string(lastLoc.EventType)
			data.HasRealLocation = true
			if !lastLoc.ServerTime.IsZero() {
				ts := lastLoc.ServerTime.Unix()
				data.LastUpdateTime = &ts
			}
		} else if tech.Latitude != nil && tech.Longitude != nil {
			// Tem coordenadas geocodificadas do endereço
			data.Latitude = *tech.Latitude
			data.Longitude = *tech.Longitude
			data.HasRealLocation = true // Marca como real pois é a localização correta do endereço
		} else {
			// Usar coordenadas da cidade/estado do banco de cidades
			lat, lng, found := s.getCoordinatesFromCityDB(tech.City, tech.State, tech.ID)
			data.Latitude = lat
			data.Longitude = lng
			data.HasRealLocation = found
		}

		geoData = append(geoData, data)
	}

	// Salvar no Redis
	if err := s.redisClient.SetAllTechniciansGeo(geoData); err != nil {
		log.Printf("❌ Error saving technicians to cache: %v", err)
		return
	}

	// Atualizar in-memory cache imediatamente após carregar no Redis
	s.memMu.Lock()
	s.memCache = geoData
	s.memCacheExp = time.Now().Add(memCacheTTL)
	s.memMu.Unlock()

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

	// Se o hash não existir (expirado), forçar recarga completa em vez de criar com 1 entry
	if err := s.redisClient.UpdateTechnicianLocation(data); err != nil {
		log.Printf("⚠️ Geo cache hash expirado para técnico %s, recarregando cache completo...", technicianID)
		go s.loadTechniciansToCache()
	}

	// Invalidar in-memory cache para que a próxima leitura pegue o dado atualizado
	s.invalidateMemCache()
}

// GetAllTechniciansFromCache retorna todos os técnicos, usando in-memory cache como
// primeira camada, Redis como segunda e banco como fallback final.
func (s *GeoService) GetAllTechniciansFromCache() ([]cache.TechnicianGeoData, error) {
	// --- 1ª camada: in-memory (sem network, sem desserialização) ---
	s.memMu.RLock()
	if s.memCache != nil && time.Now().Before(s.memCacheExp) {
		result := s.memCache
		s.memMu.RUnlock()
		return result, nil
	}
	s.memMu.RUnlock()

	// --- 2ª camada: Redis ---
	if s.redisClient == nil {
		return s.loadTechniciansDirectly()
	}

	technicians, err := s.redisClient.GetAllTechniciansGeo()
	if err != nil || len(technicians) == 0 {
		log.Println("⚠️ Geo cache miss, loading from database...")
		go s.loadTechniciansToCache()
		return s.loadTechniciansDirectly()
	}

	// Guardar em memória para os próximos 30s
	s.memMu.Lock()
	s.memCache = technicians
	s.memCacheExp = time.Now().Add(memCacheTTL)
	s.memMu.Unlock()

	return technicians, nil
}

// invalidateMemCache descarta o in-memory cache para que a próxima leitura
// busque dados atualizados no Redis.
func (s *GeoService) invalidateMemCache() {
	s.memMu.Lock()
	s.memCache = nil
	s.memCacheExp = time.Time{}
	s.memMu.Unlock()
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
			lat, lng, found := s.getCoordinatesFromCityDB(tech.City, tech.State, tech.ID)
			data.Latitude = lat
			data.Longitude = lng
			data.HasRealLocation = found
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

	// Deletar tanto o hash quanto o flag para forçar recarga completa
	s.redisClient.Delete(cache.AllTechniciansGeoKey)
	s.redisClient.Delete(cache.GeoCacheLoadedKey)

	// Recarregar (bloqueante para que o caller saiba quando terminou)
	s.loadTechniciansToCache()

	return nil
}

// GetGeoCacheStats retorna estatísticas do cache de geolocalização
func (s *GeoService) GetGeoCacheStats() (map[string]interface{}, error) {
	if s.redisClient == nil {
		return nil, errors.New("redis client not available")
	}

	// Quantidade de técnicos no cache
	count, err := s.redisClient.GetGeoCacheCount()
	if err != nil {
		count = 0
	}

	// Verificar se o cache está marcado como loaded
	isLoaded := s.redisClient.IsGeoCacheLoaded()

	// Buscar todos do cache para estatísticas
	technicians, _ := s.redisClient.GetAllTechniciansGeo()

	// Contar técnicos por status
	statusCounts := make(map[string]int)
	withRealLocation := 0
	withEstimatedLocation := 0

	for _, tech := range technicians {
		statusCounts[tech.Status]++
		if tech.HasRealLocation {
			withRealLocation++
		} else {
			withEstimatedLocation++
		}
	}

	// Quantidade de técnicos no banco (para comparar)
	dbTechnicians, _ := s.technicianRepo.GetAll()
	dbCount := len(dbTechnicians)

	return map[string]interface{}{
		"cacheLoaded":           isLoaded,
		"cachedCount":           count,
		"databaseCount":         dbCount,
		"mismatch":              dbCount != int(count),
		"statusCounts":          statusCounts,
		"withRealLocation":      withRealLocation,
		"withEstimatedLocation": withEstimatedLocation,
	}, nil
}

// getCoordinatesFromCityDB busca coordenadas do banco de cidades
func (s *GeoService) getCoordinatesFromCityDB(city, state, technicianID string) (lat, lng float64, found bool) {
	// Primeiro tenta usar o CityService se disponível
	if s.cityService != nil {
		lat, lng, found = s.cityService.GetCoordinates(city, state)
		if found {
			return lat, lng, found
		}
	}

	// Fallback para o mapa estático
	lat, lng, found = GetCoordinatesForLocation(city, state)
	return lat, lng, found
}

// LoadCities carrega as cidades do IBGE para o banco de dados
func (s *GeoService) LoadCities() (int, error) {
	if s.cityService == nil {
		return 0, errors.New("city service not available")
	}
	return s.cityService.LoadCitiesFromIBGE()
}

// GetCityCount retorna o número de cidades no banco
func (s *GeoService) GetCityCount() (int64, error) {
	if s.cityService == nil {
		return 0, errors.New("city service not available")
	}
	return s.cityService.GetCount()
}
