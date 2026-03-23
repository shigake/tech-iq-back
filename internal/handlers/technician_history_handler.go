package handlers

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/services"
)

type TechnicianHistoryHandler struct {
	service  services.TechnicianHistoryService
	validate *validator.Validate
}

func NewTechnicianHistoryHandler(service services.TechnicianHistoryService) *TechnicianHistoryHandler {
	return &TechnicianHistoryHandler{
		service:  service,
		validate: validator.New(),
	}
}

func (h *TechnicianHistoryHandler) GetHistory(c *fiber.Ctx) error {
	technicianID := c.Params("id")
	page, _ := strconv.Atoi(c.Query("page", "0"))
	size, _ := strconv.Atoi(c.Query("size", "20"))

	if size > 100 {
		size = 100
	}

	result, err := h.service.GetByTechnicianID(technicianID, page, size)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch technician history",
		})
	}

	return c.JSON(result)
}

func (h *TechnicianHistoryHandler) CreateHistory(c *fiber.Ctx) error {
	technicianID := c.Params("id")
	userID, _ := c.Locals("userId").(string)

	var req models.CreateTechnicianHistoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	history, err := h.service.Create(technicianID, &req, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(history)
}
