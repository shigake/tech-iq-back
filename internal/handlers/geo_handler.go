package handlers

import (
	"strconv"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"github.com/shigake/tech-iq-back/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type GeoHandler struct {
	geoService       *services.GeoService
	geocodingService *services.GeocodingService
}

func NewGeoHandler(geoService *services.GeoService, geocodingService *services.GeocodingService) *GeoHandler {
	return &GeoHandler{
		geoService:       geoService,
		geocodingService: geocodingService,
	}
}

// CreateLocation godoc
// @Summary Enviar localização
// @Description Registra a localização do técnico (check-in, check-out ou heartbeat)
// @Tags Geo
// @Accept json
// @Produce json
// @Param request body models.CreateLocationRequest true "Dados da localização"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 429 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/locations [post]
func (h *GeoHandler) CreateLocation(c *fiber.Ctx) error {
	// Obter usuário do contexto (JWT)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "Invalid or missing token",
			},
		})
	}

	// Parse do body
	var req models.CreateLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_BODY",
				"message": "Invalid request body",
			},
		})
	}

	// Validar campos obrigatórios
	if req.EventType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "MISSING_EVENT_TYPE",
				"message": "eventType is required",
			},
		})
	}

	// Criar localização
	location, err := h.geoService.CreateLocation(userID.String(), &req)
	if err != nil {
		if err.Error() == "rate limited: too many location updates" {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":              "RATE_LIMITED",
					"message":           "Too many location updates. Try again in 45 seconds.",
					"retryAfterSeconds": 45,
				},
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_COORDINATES",
				"message": err.Error(),
			},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":         location.ID,
			"serverTime": location.ServerTime,
		},
	})
}

// CreateBatchLocations godoc
// @Summary Enviar múltiplas localizações
// @Description Sincroniza localizações armazenadas offline
// @Tags Geo
// @Accept json
// @Produce json
// @Param request body models.BatchLocationRequest true "Lote de localizações"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/locations/batch [post]
func (h *GeoHandler) CreateBatchLocations(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "Invalid or missing token",
			},
		})
	}

	var req models.BatchLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_BODY",
				"message": "Invalid request body",
			},
		})
	}

	if len(req.Locations) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "EMPTY_BATCH",
				"message": "At least one location is required",
			},
		})
	}

	if len(req.Locations) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "BATCH_TOO_LARGE",
				"message": "Maximum 100 locations per batch",
			},
		})
	}

	results, err := h.geoService.CreateBatchLocations(userID.String(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
	}

	processed := 0
	for _, r := range results {
		if r.Status == "created" {
			processed++
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"processed": processed,
			"results":   results,
		},
	})
}

// GetTechniciansLastLocations godoc
// @Summary Listar última localização dos técnicos
// @Description Retorna a última localização conhecida de cada técnico
// @Tags Geo
// @Produce json
// @Param scopeId query string false "Filtrar por escopo"
// @Param status query string false "Filtrar por status (AVAILABLE, IN_SERVICE)"
// @Param q query string false "Buscar por nome"
// @Param state query string false "Filtrar por estado (UF)"
// @Param city query string false "Filtrar por cidade"
// @Param updatedSince query string false "Apenas atualizados após (ISO8601)"
// @Param page query int false "Página" default(1)
// @Param limit query int false "Limite" default(50)
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/technicians/last [get]
func (h *GeoHandler) GetTechniciansLastLocations(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "Invalid or missing token",
			},
		})
	}

	// TODO: Verificar permissão TECH_LOCATION_VIEW

	// Parse dos filtros
	filter := repositories.GeoFilter{
		Status: c.Query("status"),
		Query:  c.Query("q"),
		State:  c.Query("state"),
		City:   c.Query("city"),
	}

	if updatedSince := c.Query("updatedSince"); updatedSince != "" {
		t, err := time.Parse(time.RFC3339, updatedSince)
		if err == nil {
			filter.UpdatedSince = &t
		}
	}

	// Bounds para filtrar pelo viewport do mapa
	if swLat, err := strconv.ParseFloat(c.Query("swLat"), 64); err == nil {
		filter.SwLat = &swLat
	}
	if swLng, err := strconv.ParseFloat(c.Query("swLng"), 64); err == nil {
		filter.SwLng = &swLng
	}
	if neLat, err := strconv.ParseFloat(c.Query("neLat"), 64); err == nil {
		filter.NeLat = &neLat
	}
	if neLng, err := strconv.ParseFloat(c.Query("neLng"), 64); err == nil {
		filter.NeLng = &neLng
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "500"))
	if limit > 10000 {
		limit = 10000
	}
	filter.Limit = limit
	filter.Offset = (page - 1) * limit

	technicians, total, err := h.geoService.GetLastLocations(userID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"technicians": technicians,
			"pagination": fiber.Map{
				"page":       page,
				"limit":      limit,
				"total":      total,
				"totalPages": totalPages,
			},
		},
	})
}

// GetTechniciansClusters godoc
// @Summary Clusters de técnicos para o mapa
// @Description Retorna clusters de técnicos agrupados por zoom, ou técnicos individuais em zoom >= 12
// @Tags Geo
// @Produce json
// @Param zoom query int true "Nível de zoom do mapa (1-20)"
// @Param swLat query number true "Latitude south-west do viewport"
// @Param swLng query number true "Longitude south-west do viewport"
// @Param neLat query number true "Latitude north-east do viewport"
// @Param neLng query number true "Longitude north-east do viewport"
// @Param status query string false "Filtrar por status"
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/clusters [get]
func (h *GeoHandler) GetTechniciansClusters(c *fiber.Ctx) error {
	_, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "UNAUTHORIZED", "message": "Invalid or missing token"},
		})
	}

	zoom, _ := strconv.Atoi(c.Query("zoom", "5"))
	swLat, _ := strconv.ParseFloat(c.Query("swLat", "0"), 64)
	swLng, _ := strconv.ParseFloat(c.Query("swLng", "0"), 64)
	neLat, _ := strconv.ParseFloat(c.Query("neLat", "0"), 64)
	neLng, _ := strconv.ParseFloat(c.Query("neLng", "0"), 64)
	status := c.Query("status")

	result, err := h.geoService.GetClusters(zoom, swLat, swLng, neLat, neLng, status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// GetTechnicianHistory godoc
// @Summary Histórico de localização de um técnico
// @Description Retorna o histórico de localizações do técnico por período
// @Tags Geo
// @Produce json
// @Param id path string true "ID do técnico"
// @Param from query string true "Data início (ISO8601)"
// @Param to query string true "Data fim (ISO8601)"
// @Param ticketId query string false "Filtrar por ticket"
// @Param eventType query string false "Filtrar por tipo (CHECKIN, CHECKOUT, HEARTBEAT)"
// @Param page query int false "Página" default(1)
// @Param limit query int false "Limite" default(100)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/technicians/{id}/history [get]
func (h *GeoHandler) GetTechnicianHistory(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "Invalid or missing token",
			},
		})
	}

	// TODO: Verificar permissão TECH_LOCATION_HISTORY_VIEW

	technicianID := c.Params("id")
	if technicianID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "Invalid technician ID",
			},
		})
	}

	// Parse do período
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr == "" || toStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "MISSING_PERIOD",
				"message": "from and to are required",
			},
		})
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_FROM",
				"message": "Invalid from date format",
			},
		})
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_TO",
				"message": "Invalid to date format",
			},
		})
	}

	filter := repositories.HistoryFilter{
		From:      from,
		To:        to,
		EventType: c.Query("eventType"),
	}

	if ticketIDStr := c.Query("ticketId"); ticketIDStr != "" {
		ticketID, err := uuid.Parse(ticketIDStr)
		if err == nil {
			filter.TicketID = &ticketID
		}
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	if limit > 10000 {
		limit = 10000
	}
	filter.Limit = limit
	filter.Offset = (page - 1) * limit

	history, total, err := h.geoService.GetTechnicianHistory(userID, technicianID, filter)
	if err != nil {
		if err.Error() == "access denied: cannot view this technician's history" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "ACCESS_DENIED",
					"message": err.Error(),
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"technicianId":   history.TechnicianID,
			"technicianName": history.TechnicianName,
			"period":         history.Period,
			"summary":        history.Summary,
			"locations":      history.Locations,
			"pagination": fiber.Map{
				"page":       page,
				"limit":      limit,
				"total":      total,
				"totalPages": totalPages,
			},
		},
	})
}

// GetTicketLocations godoc
// @Summary Localizações de um ticket
// @Description Retorna as localizações (check-in/check-out) associadas a um ticket
// @Tags Geo
// @Produce json
// @Param id path string true "ID do ticket"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/tickets/{id}/locations [get]
func (h *GeoHandler) GetTicketLocations(c *fiber.Ctx) error {
	ticketID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "Invalid ticket ID",
			},
		})
	}

	// TODO: Verificar se o usuário tem acesso ao ticket

	locations, err := h.geoService.GetTicketLocations(ticketID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    locations,
	})
}

// GetGeoSettings godoc
// @Summary Obter configurações de geolocalização
// @Description Retorna as configurações globais e por escopo
// @Tags Geo
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/settings [get]
func (h *GeoHandler) GetGeoSettings(c *fiber.Ctx) error {
	// TODO: Verificar permissão TECH_LOCATION_ADMIN

	settings, err := h.geoService.GetGeoSettings()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    settings,
	})
}

// UpdateGeoSettings godoc
// @Summary Atualizar configurações de geolocalização
// @Description Atualiza as configurações globais ou por escopo
// @Tags Geo
// @Accept json
// @Produce json
// @Param request body models.UpdateGeoSettingsRequest true "Configurações"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/settings [put]
func (h *GeoHandler) UpdateGeoSettings(c *fiber.Ctx) error {
	// TODO: Verificar permissão TECH_LOCATION_ADMIN

	var req models.UpdateGeoSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_BODY",
				"message": "Invalid request body",
			},
		})
	}

	if err := h.geoService.UpdateGeoSettings(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Settings updated successfully",
	})
}

// RefreshGeoCache godoc
// @Summary Força recarga do cache de geolocalização
// @Description Invalida e recarrega o cache de técnicos para o mapa
// @Tags Geo
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/cache/refresh [post]
func (h *GeoHandler) RefreshGeoCache(c *fiber.Ctx) error {
	if err := h.geoService.RefreshGeoCache(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "CACHE_REFRESH_ERROR",
				"message": err.Error(),
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Geo cache refreshed successfully",
	})
}

// GetGeoCacheStatus godoc
// @Summary Status do cache de geolocalização
// @Description Retorna estatísticas do cache de técnicos
// @Tags Geo
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/cache/status [get]
func (h *GeoHandler) GetGeoCacheStatus(c *fiber.Ctx) error {
	stats, err := h.geoService.GetGeoCacheStats()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "CACHE_STATUS_ERROR",
				"message": err.Error(),
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// LoadCities godoc
// @Summary Carrega cidades do IBGE
// @Description Baixa e popula a tabela de cidades brasileiras com coordenadas
// @Tags Geo
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/cities/load [post]
func (h *GeoHandler) LoadCities(c *fiber.Ctx) error {
	count, err := h.geoService.LoadCities()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "CITIES_LOAD_ERROR",
				"message": err.Error(),
			},
		})
	}

	// Refresh cache after loading cities
	h.geoService.RefreshGeoCache()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Cities loaded successfully",
		"count":   count,
	})
}

// GetCityCount godoc
// @Summary Conta cidades no banco
// @Description Retorna o número de cidades cadastradas
// @Tags Geo
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/cities/count [get]
func (h *GeoHandler) GetCityCount(c *fiber.Ctx) error {
	count, err := h.geoService.GetCityCount()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "CITY_COUNT_ERROR",
				"message": err.Error(),
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"count":   count,
	})
}

// GeocodeAllTechnicians godoc
// @Summary Geocodifica endereços dos técnicos
// @Description Converte os endereços dos técnicos em coordenadas usando Nominatim
// @Tags Geo
// @Produce json
// @Param forceAll query bool false "Regeocodificar todos, mesmo os que já têm coordenadas"
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/geocode/batch [post]
func (h *GeoHandler) GeocodeAllTechnicians(c *fiber.Ctx) error {
	forceAll := c.Query("forceAll") == "true"

	// Run geocoding in background
	go func() {
		stats, err := h.geocodingService.GeocodeAllTechnicians(forceAll)
		if err != nil {
			// Log error but don't affect the response
			return
		}

		// Refresh geo cache after geocoding completes
		if stats.SuccessCount > 0 {
			h.geoService.RefreshGeoCache()
		}
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success": true,
		"message": "Geocoding started in background. Use GET /api/geo/geocode/status to check progress.",
	})
}

// GetGeocodingStatus godoc
// @Summary Status do job de geocodificação
// @Description Verifica se há um job de geocodificação em andamento
// @Tags Geo
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/geocode/status [get]
func (h *GeoHandler) GetGeocodingStatus(c *fiber.Ctx) error {
	isRunning := h.geocodingService.GetGeocodingStatus()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"isRunning": isRunning,
	})
}

// GeocodeSingleTechnician godoc
// @Summary Geocodifica um único técnico
// @Description Converte o endereço de um técnico específico em coordenadas
// @Tags Geo
// @Produce json
// @Param id path string true "ID do técnico"
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/geo/geocode/{id} [post]
func (h *GeoHandler) GeocodeSingleTechnician(c *fiber.Ctx) error {
	technicianID := c.Params("id")
	if technicianID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "MISSING_ID",
				"message": "Technician ID is required",
			},
		})
	}

	// Geocode using the service (it will find the technician)
	result, err := h.geocodingService.GeocodeTechnicianByID(technicianID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "GEOCODING_ERROR",
				"message": err.Error(),
			},
		})
	}

	if result == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Technician not found",
			},
		})
	}

	// Refresh cache if successful
	if result.Success {
		h.geoService.RefreshGeoCache()
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// Helper para obter userID do contexto JWT
func getUserIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	userIDStr := c.Locals("userId")
	if userIDStr == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "User not authenticated")
	}

	switch v := userIDStr.(type) {
	case string:
		return uuid.Parse(v)
	case uuid.UUID:
		return v, nil
	default:
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid user ID type")
	}
}
