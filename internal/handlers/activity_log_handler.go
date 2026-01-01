package handlers

import (
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/services"
)

type ActivityLogHandler struct {
	service  services.ActivityLogService
	validate *validator.Validate
}

func NewActivityLogHandler(service services.ActivityLogService) *ActivityLogHandler {
	return &ActivityLogHandler{
		service:  service,
		validate: validator.New(),
	}
}

// GetActivityLogs returns all activity logs with pagination and filters
// @Summary List activity logs
// @Tags ActivityLogs
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param userId query string false "Filter by user ID"
// @Param action query string false "Filter by action type"
// @Param resource query string false "Filter by resource type"
// @Param startDate query string false "Filter by start date (RFC3339)"
// @Param endDate query string false "Filter by end date (RFC3339)"
// @Success 200 {object} models.PaginatedActivityLogs
// @Router /api/v1/activity-logs [get]
func (h *ActivityLogHandler) GetActivityLogs(c *fiber.Ctx) error {
	// Check if user is admin
	userRole := getUserRole(c)
	if userRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can view all activity logs",
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	filter := &models.ActivityLogFilter{
		UserID:     c.Query("userId"),
		Action:     c.Query("action"),
		Resource:   c.Query("resource"),
		ResourceID: c.Query("resourceId"),
	}

	// Parse date filters
	if startDate := c.Query("startDate"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			filter.StartDate = t
		}
	}
	if endDate := c.Query("endDate"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			filter.EndDate = t
		}
	}

	result, err := h.service.GetAll(filter, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch activity logs",
		})
	}

	return c.JSON(result)
}

// GetMyActivityLogs returns activity logs for the current user
// @Summary Get my activity logs
// @Tags ActivityLogs
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} models.PaginatedActivityLogs
// @Router /api/v1/activity-logs/me [get]
func (h *ActivityLogHandler) GetMyActivityLogs(c *fiber.Ctx) error {
	userID := getUserID(c)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	result, err := h.service.GetByUserID(userID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch activity logs",
		})
	}

	return c.JSON(result)
}

// GetRecentActivityLogs returns the most recent activity logs
// @Summary Get recent activity logs
// @Tags ActivityLogs
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of logs" default(10)
// @Success 200 {array} models.ActivityLog
// @Router /api/v1/activity-logs/recent [get]
func (h *ActivityLogHandler) GetRecentActivityLogs(c *fiber.Ctx) error {
	// Check if user is admin
	userRole := getUserRole(c)
	if userRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can view recent activity logs",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	logs, err := h.service.GetRecentLogs(limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch recent activity logs",
		})
	}

	return c.JSON(logs)
}

// GetActivityLogByID returns a specific activity log by ID
// @Summary Get activity log by ID
// @Tags ActivityLogs
// @Security BearerAuth
// @Produce json
// @Param id path string true "Activity Log ID"
// @Success 200 {object} models.ActivityLog
// @Router /api/v1/activity-logs/{id} [get]
func (h *ActivityLogHandler) GetActivityLogByID(c *fiber.Ctx) error {
	id := c.Params("id")

	log, err := h.service.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Activity log not found",
		})
	}

	// Users can only view their own logs (admins can view all)
	userRole := getUserRole(c)
	userID := getUserID(c)
	if userRole != "ADMIN" && log.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	return c.JSON(log)
}
