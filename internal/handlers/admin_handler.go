package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/services"
)

type AdminHandler struct {
	systemMetricsService services.SystemMetricsService
}

func NewAdminHandler(systemMetricsService services.SystemMetricsService) *AdminHandler {
	return &AdminHandler{
		systemMetricsService: systemMetricsService,
	}
}

// GetSystemMetrics godoc
// @Summary Get system metrics (admin only)
// @Description Returns current system metrics including memory, CPU, database, and business metrics
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {object} models.SystemMetrics
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Security BearerAuth
// @Router /api/admin/system-metrics [get]
func (h *AdminHandler) GetSystemMetrics(c *fiber.Ctx) error {
	// Check if user is admin
	userRole := getUserRole(c)
	if userRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can view system metrics",
		})
	}

	metrics, err := h.systemMetricsService.GetMetrics()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch system metrics",
		})
	}

	return c.JSON(metrics)
}

// GetHealthCheck godoc
// @Summary Get server health status
// @Description Returns basic health check information
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/health [get]
func (h *AdminHandler) GetHealthCheck(c *fiber.Ctx) error {
	metrics, err := h.systemMetricsService.GetMetrics()
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "unhealthy",
			"error":  err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":    "healthy",
		"uptime":    metrics.ServerUptime,
		"version":   metrics.ServerVersion,
		"goVersion": metrics.GoVersion,
		"timestamp": metrics.Timestamp,
	})
}
