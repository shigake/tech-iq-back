package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/go-playground/validator/v10"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
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
	// First parse as map to handle both 'name' and 'fullName' fields
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Extract fullName from either 'fullName' or 'name' field
	var fullName string
	if fn, exists := body["fullName"]; exists && fn != nil {
		if s, ok := fn.(string); ok {
			fullName = s
		}
	}
	if fullName == "" {
		if n, exists := body["name"]; exists && n != nil {
			if s, ok := n.(string); ok {
				fullName = s
			}
		}
	}

	if fullName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nome é obrigatório",
		})
	}

	// Build client from body
	client := models.Client{
		FullName:          fullName,
		CPF:               getStringFromMap(body, "cpf"),
		CNPJ:              getStringFromMap(body, "cnpj"),
		InscricaoEstadual: getStringFromMap(body, "inscricaoEstadual"),
		Email:             getStringFromMap(body, "email"),
		Phone:             getStringFromMap(body, "phone"),
		Street:            getStringFromMap(body, "street"),
		Number:            getStringFromMap(body, "number"),
		Complement:        getStringFromMap(body, "complement"),
		Neighborhood:      getStringFromMap(body, "neighborhood"),
		City:              getStringFromMap(body, "city"),
		State:             getStringFromMap(body, "state"),
		ZipCode:           getStringFromMap(body, "zipCode"),
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

// getStringFromMap safely extracts a string from a map
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, exists := m[key]; exists && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
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

	// Parse as map to handle both 'name' and 'fullName' fields
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Extract fullName from either 'fullName' or 'name' field
	if fn, exists := body["fullName"]; exists && fn != nil {
		if s, ok := fn.(string); ok && s != "" {
			existing.FullName = s
		}
	} else if n, exists := body["name"]; exists && n != nil {
		if s, ok := n.(string); ok && s != "" {
			existing.FullName = s
		}
	}

	// Update other fields
	if v := getStringFromMap(body, "cpf"); v != "" || body["cpf"] != nil {
		existing.CPF = sanitizeUniqueField(v)
	}
	if v := getStringFromMap(body, "cnpj"); v != "" || body["cnpj"] != nil {
		existing.CNPJ = sanitizeUniqueField(v)
	}
	if v := getStringFromMap(body, "inscricaoEstadual"); v != "" || body["inscricaoEstadual"] != nil {
		existing.InscricaoEstadual = v
	}
	if v := getStringFromMap(body, "email"); v != "" || body["email"] != nil {
		existing.Email = v
	}
	if v := getStringFromMap(body, "phone"); v != "" || body["phone"] != nil {
		existing.Phone = v
	}
	if v := getStringFromMap(body, "street"); v != "" || body["street"] != nil {
		existing.Street = v
	}
	if v := getStringFromMap(body, "number"); v != "" || body["number"] != nil {
		existing.Number = v
	}
	if v := getStringFromMap(body, "complement"); v != "" || body["complement"] != nil {
		existing.Complement = v
	}
	if v := getStringFromMap(body, "neighborhood"); v != "" || body["neighborhood"] != nil {
		existing.Neighborhood = v
	}
	if v := getStringFromMap(body, "city"); v != "" || body["city"] != nil {
		existing.City = v
	}
	if v := getStringFromMap(body, "state"); v != "" || body["state"] != nil {
		existing.State = v
	}
	if v := getStringFromMap(body, "zipCode"); v != "" || body["zipCode"] != nil {
		existing.ZipCode = v
	}

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
