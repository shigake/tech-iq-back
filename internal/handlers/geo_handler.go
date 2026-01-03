package handlers

import (
	"strconv"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"github.com/shigake/tech-iq-back/internal/services"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type GeoHandler struct {
	geoService *services.GeoService
}

func NewGeoHandler(geoService *services.GeoService) *GeoHandler {
	return &GeoHandler{geoService: geoService}
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
	}

	if updatedSince := c.Query("updatedSince"); updatedSince != "" {
		t, err := time.Parse(time.RFC3339, updatedSince)
		if err == nil {
			filter.UpdatedSince = &t
		}
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if limit > 200 {
		limit = 200
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
	if limit > 1000 {
		limit = 1000
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
