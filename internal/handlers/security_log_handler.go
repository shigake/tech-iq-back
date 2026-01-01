package handlers

import (
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/services"
)

type SecurityLogHandler struct {
	service  services.SecurityLogService
	validate *validator.Validate
}

func NewSecurityLogHandler(service services.SecurityLogService) *SecurityLogHandler {
	return &SecurityLogHandler{
		service:  service,
		validate: validator.New(),
	}
}

// GetSecurityLogs godoc
// @Summary Get all security logs (admin only)
// @Description Returns paginated security logs with optional filters
// @Tags Security
// @Accept json
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Items per page (default 20)"
// @Param action query string false "Filter by action (login_success, login_failed, logout, etc.)"
// @Param success query bool false "Filter by success status"
// @Param email query string false "Filter by email"
// @Param ipAddress query string false "Filter by IP address"
// @Param startDate query string false "Filter by start date (RFC3339)"
// @Param endDate query string false "Filter by end date (RFC3339)"
// @Success 200 {object} models.PaginatedSecurityLogs
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Security BearerAuth
// @Router /api/admin/security-logs [get]
func (h *SecurityLogHandler) GetSecurityLogs(c *fiber.Ctx) error {
	// Check if user is admin
	userRole := getUserRole(c)
	if userRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can view security logs",
		})
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	// Parse filters
	filter := &models.SecurityLogFilter{
		Action:    c.Query("action"),
		Email:     c.Query("email"),
		IPAddress: c.Query("ipAddress"),
	}

	// Parse success filter
	if successStr := c.Query("success"); successStr != "" {
		success := successStr == "true"
		filter.Success = &success
	}

	// Parse date filters
	if startDateStr := c.Query("startDate"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			filter.StartDate = startDate
		}
	}
	if endDateStr := c.Query("endDate"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			filter.EndDate = endDate
		}
	}

	result, err := h.service.GetAll(filter, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch security logs",
		})
	}

	return c.JSON(result)
}

// GetRecentSecurityLogs godoc
// @Summary Get recent security logs (admin only)
// @Description Returns the most recent security logs
// @Tags Security
// @Accept json
// @Produce json
// @Param limit query int false "Number of logs to return (default 10)"
// @Success 200 {array} models.SecurityLog
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Security BearerAuth
// @Router /api/admin/security-logs/recent [get]
func (h *SecurityLogHandler) GetRecentSecurityLogs(c *fiber.Ctx) error {
	// Check if user is admin
	userRole := getUserRole(c)
	if userRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can view security logs",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit > 100 {
		limit = 100
	}

	logs, err := h.service.GetRecentLogs(limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch security logs",
		})
	}

	return c.JSON(logs)
}

// GetSecurityStats godoc
// @Summary Get security statistics (admin only)
// @Description Returns today's security statistics
// @Tags Security
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Security BearerAuth
// @Router /api/admin/security-logs/stats [get]
func (h *SecurityLogHandler) GetSecurityStats(c *fiber.Ctx) error {
	// Check if user is admin
	userRole := getUserRole(c)
	if userRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can view security stats",
		})
	}

	successful, failed, err := h.service.GetTodayStats()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch security stats",
		})
	}

	return c.JSON(fiber.Map{
		"todaySuccessful": successful,
		"todayFailed":     failed,
		"totalToday":      successful + failed,
	})
}
