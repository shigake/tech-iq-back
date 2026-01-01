package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tech-erp/backend/internal/services"
)

type DashboardHandler struct {
	service services.DashboardService
}

func NewDashboardHandler(service services.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

// GetStats returns dashboard statistics
func (h *DashboardHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch stats",
		})
	}
	return c.JSON(stats)
}

// GetTicketsByStatus returns tickets grouped by status
func (h *DashboardHandler) GetTicketsByStatus(c *fiber.Ctx) error {
	data, err := h.service.GetTicketsByStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch data",
		})
	}
	return c.JSON(data)
}

// GetTechniciansByState returns technicians grouped by state
func (h *DashboardHandler) GetTechniciansByState(c *fiber.Ctx) error {
	data, err := h.service.GetTechniciansByState()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch data",
		})
	}
	return c.JSON(data)
}

// GetChartData returns chart data for the dashboard
func (h *DashboardHandler) GetChartData(c *fiber.Ctx) error {
	data, err := h.service.GetTicketsByStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch chart data",
		})
	}
	
	// Convert to chart format
	chartData := make([]fiber.Map, len(data))
	for i, d := range data {
		chartData[i] = fiber.Map{
			"label": d.Status,
			"count": d.Count,
		}
	}
	
	return c.JSON(chartData)
}

// GetRecentActivity returns recent activity for the dashboard
func (h *DashboardHandler) GetRecentActivity(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 10)
	
	activities, err := h.service.GetRecentActivity(limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch recent activity",
		})
	}
	
	return c.JSON(activities)
}
