package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/go-playground/validator/v10"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type CategoryHandler struct {
	repo     repositories.CategoryRepository
	validate *validator.Validate
}

func NewCategoryHandler(repo repositories.CategoryRepository) *CategoryHandler {
	return &CategoryHandler{
		repo:     repo,
		validate: validator.New(),
	}
}

// GetAll returns all categories
func (h *CategoryHandler) GetAll(c *fiber.Ctx) error {
	// Check for type filter
	categoryType := c.Query("type")
	
	var categories []models.Category
	var err error
	
	if categoryType != "" {
		categories, err = h.repo.GetByTypeWithChildren(models.CategoryType(categoryType))
	} else {
		categories, err = h.repo.GetAll()
	}
	
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch categories",
		})
	}

	return c.JSON(categories)
}

// GetByID returns a category by ID
func (h *CategoryHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid category ID",
		})
	}

	category, err := h.repo.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Category not found",
		})
	}

	return c.JSON(category)
}

// Create creates a new category
func (h *CategoryHandler) Create(c *fiber.Ctx) error {
	var category models.Category
	if err := c.BodyParser(&category); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Set default type if not provided
	if category.Type == "" {
		category.Type = models.CategoryTypeTicket
	}

	// Check if category name already exists for this type (only for parent categories)
	if category.ParentID == nil {
		existing, _ := h.repo.GetByNameAndType(category.Name, category.Type)
		if existing != nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Category with this name already exists for this type",
			})
		}
	}

	if err := h.repo.Create(&category); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(category)
}

// Update updates a category
func (h *CategoryHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid category ID",
		})
	}

	existing, err := h.repo.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Category not found",
		})
	}

	var update models.Category
	if err := c.BodyParser(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	existing.Name = update.Name
	existing.Description = update.Description
	existing.Color = update.Color
	existing.Icon = update.Icon
	existing.Active = update.Active
	existing.SortOrder = update.SortOrder

	if err := h.repo.Update(existing); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(existing)
}

// Delete deletes a category
func (h *CategoryHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid category ID",
		})
	}

	if err := h.repo.Delete(id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Category not found",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
