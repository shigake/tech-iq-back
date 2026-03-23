package handlers

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/services"
)

type TicketSignatureHandler struct {
	service  services.TicketSignatureService
	validate *validator.Validate
}

func NewTicketSignatureHandler(service services.TicketSignatureService) *TicketSignatureHandler {
	return &TicketSignatureHandler{
		service:  service,
		validate: validator.New(),
	}
}

func (h *TicketSignatureHandler) CreateOrUpdateSignature(c *fiber.Ctx) error {
	ticketID := c.Params("id")

	var req models.CreateTicketSignatureRequest
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

	signature, err := h.service.CreateOrUpdate(ticketID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(signature)
}

func (h *TicketSignatureHandler) GetSignature(c *fiber.Ctx) error {
	ticketID := c.Params("id")

	signature, err := h.service.GetByTicketID(ticketID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Signature not found",
		})
	}

	return c.JSON(signature)
}

func (h *TicketSignatureHandler) DeleteSignatureRecord(c *fiber.Ctx) error {
	ticketID := c.Params("id")

	if err := h.service.Delete(ticketID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete signature",
		})
	}

	return c.JSON(fiber.Map{"message": "Signature deleted"})
}
