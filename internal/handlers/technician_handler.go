package handlers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/go-playground/validator/v10"
	"github.com/tech-erp/backend/internal/models"
	"github.com/tech-erp/backend/internal/services"
)

type TechnicianHandler struct {
	service  services.TechnicianService
	validate *validator.Validate
}

func NewTechnicianHandler(service services.TechnicianService) *TechnicianHandler {
	return &TechnicianHandler{
		service:  service,
		validate: validator.New(),
	}
}

// GetAll returns paginated list of technicians
// @Summary List all technicians
// @Tags Technicians
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(0)
// @Param size query int false "Page size" default(20)
// @Param search query string false "Search term"
// @Param ids query string false "Comma-separated IDs"
// @Security BearerAuth
// @Success 200 {object} models.PaginatedResponse
// @Router /api/v1/technicians [get]
func (h *TechnicianHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "0"))
	size, _ := strconv.Atoi(c.Query("size", "20"))
	search := c.Query("search", "")
	idsParam := c.Query("ids", "")
	
	// Filter parameters
	status := c.Query("status", "")
	techType := c.Query("type", "")
	city := c.Query("city", "")
	state := c.Query("state", "")
	
	fmt.Printf(">>> GetAll params: page=%d, size=%d, search='%s', status='%s', type='%s', city='%s', state='%s'\n", 
		page, size, search, status, techType, city, state)

	if size > 1000 {
		size = 1000
	}

	// Handle specific IDs filter
	if idsParam != "" {
		response, err := h.service.FindByIDs(idsParam)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch technicians",
			})
		}
		return c.JSON(response)
	}

	// Handle search with optional filters
	if search != "" || status != "" || techType != "" || city != "" || state != "" {
		response, err := h.service.SearchWithFilters(search, status, techType, city, state, page, size)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to search technicians",
			})
		}
		return c.JSON(response)
	}

	// Regular listing
	response, err := h.service.GetAll(page, size)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch technicians",
		})
	}

	return c.JSON(response)
}

// GetByID returns a technician by ID
// @Summary Get technician by ID
// @Tags Technicians
// @Accept json
// @Produce json
// @Param id path string true "Technician ID"
// @Security BearerAuth
// @Success 200 {object} models.Technician
// @Failure 404 {object} map[string]string
// @Router /api/v1/technicians/{id} [get]
func (h *TechnicianHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	
	technician, err := h.service.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Technician not found",
		})
	}

	return c.JSON(technician)
}

// Create creates a new technician
// @Summary Create technician
// @Tags Technicians
// @Accept json
// @Produce json
// @Param request body models.CreateTechnicianRequest true "Technician data"
// @Security BearerAuth
// @Success 201 {object} models.Technician
// @Failure 400 {object} map[string]string
// @Router /api/v1/technicians [post]
func (h *TechnicianHandler) Create(c *fiber.Ctx) error {
	var req models.CreateTechnicianRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": formatValidationErrors(err),
		})
	}

	technician, err := h.service.Create(&req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(technician)
}

// Update updates a technician
// @Summary Update technician
// @Tags Technicians
// @Accept json
// @Produce json
// @Param id path string true "Technician ID"
// @Param request body models.CreateTechnicianRequest true "Technician data"
// @Security BearerAuth
// @Success 200 {object} models.Technician
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/technicians/{id} [put]
func (h *TechnicianHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	
	var req models.CreateTechnicianRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	technician, err := h.service.Update(id, &req)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Technician not found",
		})
	}

	return c.JSON(technician)
}

// Delete deletes a technician
// @Summary Delete technician
// @Tags Technicians
// @Accept json
// @Produce json
// @Param id path string true "Technician ID"
// @Security BearerAuth
// @Success 204
// @Failure 404 {object} map[string]string
// @Router /api/v1/technicians/{id} [delete]
func (h *TechnicianHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	
	if err := h.service.Delete(id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Technician not found",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Search searches technicians
// @Summary Search technicians
// @Tags Technicians
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param page query int false "Page number" default(0)
// @Param size query int false "Page size" default(20)
// @Security BearerAuth
// @Success 200 {object} models.PaginatedResponse
// @Router /api/v1/technicians/search [get]
func (h *TechnicianHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q", "")
	page, _ := strconv.Atoi(c.Query("page", "0"))
	size, _ := strconv.Atoi(c.Query("size", "20"))

	response, err := h.service.Search(query, page, size)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Search failed",
		})
	}

	return c.JSON(response)
}

// GetByCity returns technicians by city
// @Summary Get technicians by city
// @Tags Technicians
// @Accept json
// @Produce json
// @Param city path string true "City name"
// @Security BearerAuth
// @Success 200 {array} models.TechnicianDTO
// @Router /api/v1/technicians/by-city/{city} [get]
func (h *TechnicianHandler) GetByCity(c *fiber.Ctx) error {
	city := c.Params("city")
	
	technicians, err := h.service.GetByCity(city)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch technicians",
		})
	}

	return c.JSON(technicians)
}

// GetByState returns technicians by state
// @Summary Get technicians by state
// @Tags Technicians
// @Accept json
// @Produce json
// @Param state path string true "State code (e.g., SP, RJ)"
// @Security BearerAuth
// @Success 200 {array} models.TechnicianDTO
// @Router /api/v1/technicians/by-state/{state} [get]
func (h *TechnicianHandler) GetByState(c *fiber.Ctx) error {
	state := c.Params("state")
	
	technicians, err := h.service.GetByState(state)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch technicians",
		})
	}

	return c.JSON(technicians)
}

// GetCities returns list of cities with technicians
// @Summary Get cities list
// @Tags Technicians
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} string
// @Router /api/v1/technicians/cities [get]
func (h *TechnicianHandler) GetCities(c *fiber.Ctx) error {
	cities, err := h.service.GetCities()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch cities",
		})
	}

	return c.JSON(cities)
}
