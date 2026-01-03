package handlers

import (
	"strconv"

	"tech-erp/internal/models"
	"tech-erp/internal/services"

	"github.com/gofiber/fiber/v2"
)

type ErrorLogHandler struct {
	service *services.ErrorLogService
}

func NewErrorLogHandler(service *services.ErrorLogService) *ErrorLogHandler {
	return &ErrorLogHandler{service: service}
}

// GetAll returns all error logs with pagination
// @Summary Get all error logs
// @Tags Error Logs
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(0)
// @Param size query int false "Page size" default(20)
// @Param level query string false "Filter by level (ERROR, WARN, CRITICAL)"
// @Param feature query string false "Filter by feature name"
// @Param endpoint query string false "Filter by endpoint"
// @Param resolved query bool false "Filter by resolved status"
// @Param search query string false "Search in error message, feature, endpoint"
// @Security BearerAuth
// @Success 200 {object} models.PaginatedErrorLogs
// @Router /api/v1/errors [get]
func (h *ErrorLogHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "0"))
	size, _ := strconv.Atoi(c.Query("size", "20"))

	filter := &models.ErrorLogFilter{
		Level:    c.Query("level"),
		Feature:  c.Query("feature"),
		Endpoint: c.Query("endpoint"),
		Search:   c.Query("search"),
	}

	// Handle resolved filter
	if resolvedStr := c.Query("resolved"); resolvedStr != "" {
		resolved := resolvedStr == "true"
		filter.Resolved = &resolved
	}

	result, err := h.service.GetAll(page, size, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch error logs",
		})
	}

	return c.JSON(result)
}

// GetByID returns an error log by ID
// @Summary Get error log by ID
// @Tags Error Logs
// @Accept json
// @Produce json
// @Param id path string true "Error Log ID"
// @Security BearerAuth
// @Success 200 {object} models.ErrorLog
// @Failure 404 {object} map[string]string
// @Router /api/v1/errors/{id} [get]
func (h *ErrorLogHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	log, err := h.service.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Error log not found",
		})
	}

	return c.JSON(log)
}

// GetStats returns statistics about error logs
// @Summary Get error log statistics
// @Tags Error Logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.ErrorLogStats
// @Router /api/v1/errors/stats [get]
func (h *ErrorLogHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch error statistics",
		})
	}

	return c.JSON(stats)
}

// Resolve marks an error as resolved
// @Summary Resolve an error
// @Tags Error Logs
// @Accept json
// @Produce json
// @Param id path string true "Error Log ID"
// @Param body body models.ResolveErrorRequest true "Resolution details"
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/v1/errors/{id}/resolve [post]
func (h *ErrorLogHandler) Resolve(c *fiber.Ctx) error {
	id := c.Params("id")
	
	// Get user ID from context
	userID := c.Locals("userId").(string)

	var req models.ResolveErrorRequest
	if err := c.BodyParser(&req); err != nil {
		req.Notes = ""
	}

	if err := h.service.Resolve(id, userID, req.Notes); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to resolve error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Error resolved successfully",
	})
}

// BulkResolve marks multiple errors as resolved
// @Summary Resolve multiple errors
// @Tags Error Logs
// @Accept json
// @Produce json
// @Param body body object true "IDs to resolve"
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Router /api/v1/errors/bulk-resolve [post]
func (h *ErrorLogHandler) BulkResolve(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)

	var req struct {
		IDs   []string `json:"ids"`
		Notes string   `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.BulkResolve(req.IDs, userID, req.Notes); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to resolve errors",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Errors resolved successfully",
		"count":   len(req.IDs),
	})
}

// Delete deletes an error log
// @Summary Delete an error log
// @Tags Error Logs
// @Accept json
// @Produce json
// @Param id path string true "Error Log ID"
// @Security BearerAuth
// @Success 204
// @Failure 400 {object} map[string]string
// @Router /api/v1/errors/{id} [delete]
func (h *ErrorLogHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.service.Delete(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to delete error log",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Cleanup deletes old resolved error logs
// @Summary Cleanup old resolved error logs
// @Tags Error Logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/errors/cleanup [post]
func (h *ErrorLogHandler) Cleanup(c *fiber.Ctx) error {
	count, err := h.service.CleanupOldLogs()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to cleanup old logs",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Old logs cleaned up successfully",
		"deleted": count,
	})
}
