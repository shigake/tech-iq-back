package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/go-playground/validator/v10"
	"github.com/tech-erp/backend/internal/models"
	"github.com/tech-erp/backend/internal/repositories"
)

type ClientHandler struct {
	repo     repositories.ClientRepository
	validate *validator.Validate
}

func NewClientHandler(repo repositories.ClientRepository) *ClientHandler {
	return &ClientHandler{
		repo:     repo,
		validate: validator.New(),
	}
}

// GetAll returns paginated list of clients
func (h *ClientHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "0"))
	size, _ := strconv.Atoi(c.Query("size", "20"))
	search := c.Query("search", "")

	var clients []models.Client
	var total int64
	var err error

	if search != "" {
		clients, total, err = h.repo.Search(search, page, size)
	} else {
		clients, total, err = h.repo.GetAll(page, size)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch clients",
		})
	}

	dtos := make([]models.ClientDTO, len(clients))
	for i, client := range clients {
		dtos[i] = client.ToDTO()
	}

	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	return c.JSON(fiber.Map{
		"content":       dtos,
		"page":          page,
		"size":          size,
		"totalElements": total,
		"totalPages":    totalPages,
	})
}

// GetByID returns a client by ID
func (h *ClientHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client ID",
		})
	}

	client, err := h.repo.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Client not found",
		})
	}

	return c.JSON(client)
}

// Create creates a new client
func (h *ClientHandler) Create(c *fiber.Ctx) error {
	var client models.Client
	if err := c.BodyParser(&client); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Check if we received 'name' field and need to map to FullName
	var body map[string]interface{}
	if err := c.BodyParser(&body); err == nil {
		if name, exists := body["name"]; exists && name != nil {
			if nameStr, ok := name.(string); ok && nameStr != "" {
				client.FullName = nameStr
			}
		}
	}
	
	// Sanitize empty strings to avoid unique constraint issues
	client.CPF = sanitizeUniqueField(client.CPF)
	client.CNPJ = sanitizeUniqueField(client.CNPJ)

	if err := h.repo.Create(&client); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(client)
}

// sanitizeUniqueField returns empty string as is but trims whitespace
// For the unique index to work correctly, we need to handle this at DB level
func sanitizeUniqueField(s string) string {
	s = strings.TrimSpace(s)
	return s
}

// Update updates a client
func (h *ClientHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client ID",
		})
	}

	existing, err := h.repo.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Client not found",
		})
	}

	var update models.Client
	if err := c.BodyParser(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	existing.FullName = update.FullName
	existing.CPF = update.CPF
	existing.CNPJ = update.CNPJ
	existing.InscricaoEstadual = update.InscricaoEstadual
	existing.Email = update.Email
	existing.Phone = update.Phone
	existing.Street = update.Street
	existing.Number = update.Number
	existing.Complement = update.Complement
	existing.Neighborhood = update.Neighborhood
	existing.City = update.City
	existing.State = update.State
	existing.ZipCode = update.ZipCode

	if err := h.repo.Update(existing); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(existing)
}

// Delete deletes a client
func (h *ClientHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client ID",
		})
	}

	if err := h.repo.Delete(id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Client not found",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Count returns total number of clients
func (h *ClientHandler) Count(c *fiber.Ctx) error {
	count, err := h.repo.Count()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to count clients",
		})
	}
	return c.JSON(fiber.Map{"count": count})
}
