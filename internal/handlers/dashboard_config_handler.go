package handlers

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/services"
)

type DashboardConfigHandler struct {
	service  services.DashboardConfigService
	validate *validator.Validate
}

func NewDashboardConfigHandler(service services.DashboardConfigService) *DashboardConfigHandler {
	return &DashboardConfigHandler{
		service:  service,
		validate: validator.New(),
	}
}

func (h *DashboardConfigHandler) GetConfig(c *fiber.Ctx) error {
	userID, _ := c.Locals("userId").(string)

	config, err := h.service.GetConfig(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch dashboard config",
		})
	}

	return c.JSON(config)
}

func (h *DashboardConfigHandler) SaveConfig(c *fiber.Ctx) error {
	userID, _ := c.Locals("userId").(string)

	var req models.SaveDashboardConfigRequest
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

	config, err := h.service.SaveConfig(userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save dashboard config",
		})
	}

	return c.JSON(config)
}
