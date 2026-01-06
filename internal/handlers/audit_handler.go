package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"gorm.io/gorm"
)

// AuditHandler handles audit log HTTP requests
type AuditHandler struct {
	repo          repositories.AuditRepository
	hierarchyRepo repositories.HierarchyRepository
	userRepo      repositories.UserRepository
	db            *gorm.DB
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{
		repo:          repositories.NewAuditRepository(db),
		hierarchyRepo: repositories.NewHierarchyRepository(db),
		userRepo:      repositories.NewUserRepository(db),
		db:            db,
	}
}

// GetAuditLogs returns audit logs with filters (respecting hierarchy)
// GET /api/v1/audit
func (h *AuditHandler) GetAuditLogs(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)
	userRole := c.Locals("role").(string)
	
	// Parse filters
	filter := models.AuditLogFilter{
		EntityType:  c.Query("entityType"),
		EntityID:    c.Query("entityId"),
		Action:      models.AuditAction(c.Query("action")),
		UserID:      c.Query("userId"),
		SearchQuery: c.Query("search"),
		Page:        0,
		Size:        20,
	}
	
	if page, err := strconv.Atoi(c.Query("page", "0")); err == nil {
		filter.Page = page
	}
	if size, err := strconv.Atoi(c.Query("size", "20")); err == nil {
		filter.Size = size
	}
	
	// Parse dates
	if startDate := c.Query("startDate"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			filter.StartDate = &t
		}
	}
	if endDate := c.Query("endDate"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &endOfDay
		}
	}
	
	// Check if user is admin - admins see everything
	if userRole == "ADMIN" {
		result, err := h.repo.FindAll(filter)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch audit logs"})
		}
		return c.JSON(result)
	}
	
	// For non-admins, get accessible technician IDs based on hierarchy
	accessibleTechIDs, err := h.getAccessibleTechnicianIDs(userID)
	if err != nil {
		// If hierarchy check fails, just show user's own actions
		accessibleTechIDs = []string{}
	}
	
	result, err := h.repo.FindAllWithHierarchy(userID, accessibleTechIDs, filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch audit logs"})
	}
	
	return c.JSON(result)
}

// GetAuditLogByID returns a specific audit log entry
// GET /api/v1/audit/:id
func (h *AuditHandler) GetAuditLogByID(c *fiber.Ctx) error {
	id := c.Params("id")
	
	log, err := h.repo.FindByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Audit log not found"})
	}
	
	return c.JSON(log)
}

// GetEntityAuditLogs returns audit logs for a specific entity
// GET /api/v1/audit/entity/:entityType/:entityId
func (h *AuditHandler) GetEntityAuditLogs(c *fiber.Ctx) error {
	entityType := c.Params("entityType")
	entityID := c.Params("entityId")
	
	page, _ := strconv.Atoi(c.Query("page", "0"))
	size, _ := strconv.Atoi(c.Query("size", "20"))
	
	logs, total, err := h.repo.FindByEntity(entityType, entityID, page, size)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch entity audit logs"})
	}
	
	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}
	
	return c.JSON(fiber.Map{
		"items":      logs,
		"total":      total,
		"page":       page,
		"size":       size,
		"totalPages": totalPages,
	})
}

// GetUserAuditLogs returns audit logs for actions by a specific user
// GET /api/v1/audit/user/:userId
func (h *AuditHandler) GetUserAuditLogs(c *fiber.Ctx) error {
	targetUserID := c.Params("userId")
	
	page, _ := strconv.Atoi(c.Query("page", "0"))
	size, _ := strconv.Atoi(c.Query("size", "20"))
	
	logs, total, err := h.repo.FindByUser(targetUserID, page, size)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch user audit logs"})
	}
	
	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}
	
	return c.JSON(fiber.Map{
		"items":      logs,
		"total":      total,
		"page":       page,
		"size":       size,
		"totalPages": totalPages,
	})
}

// GetAuditStats returns audit statistics
// GET /api/v1/audit/stats
func (h *AuditHandler) GetAuditStats(c *fiber.Ctx) error {
	filter := models.AuditLogFilter{
		EntityType: c.Query("entityType"),
	}
	
	// Parse dates
	if startDate := c.Query("startDate"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			filter.StartDate = &t
		}
	}
	if endDate := c.Query("endDate"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &endOfDay
		}
	}
	
	stats, err := h.repo.GetStats(filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch audit stats"})
	}
	
	return c.JSON(stats)
}

// GetEntityTypes returns all distinct entity types
// GET /api/v1/audit/entity-types
func (h *AuditHandler) GetEntityTypes(c *fiber.Ctx) error {
	types, err := h.repo.GetDistinctEntityTypes()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch entity types"})
	}
	
	// Add labels
	result := make([]map[string]string, len(types))
	for i, t := range types {
		label, ok := models.EntityTypeLabels[t]
		if !ok {
			label = t
		}
		result[i] = map[string]string{
			"value": t,
			"label": label,
		}
	}
	
	return c.JSON(result)
}

// GetAuditUsers returns all users with audit log entries
// GET /api/v1/audit/users
func (h *AuditHandler) GetAuditUsers(c *fiber.Ctx) error {
	users, err := h.repo.GetDistinctUsers()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch audit users"})
	}
	
	return c.JSON(users)
}

// GetActionTypes returns all available action types
// GET /api/v1/audit/action-types
func (h *AuditHandler) GetActionTypes(c *fiber.Ctx) error {
	actions := []map[string]string{
		{"value": string(models.AuditActionCreate), "label": models.ActionLabels[models.AuditActionCreate]},
		{"value": string(models.AuditActionUpdate), "label": models.ActionLabels[models.AuditActionUpdate]},
		{"value": string(models.AuditActionDelete), "label": models.ActionLabels[models.AuditActionDelete]},
		{"value": string(models.AuditActionView), "label": models.ActionLabels[models.AuditActionView]},
		{"value": string(models.AuditActionExport), "label": models.ActionLabels[models.AuditActionExport]},
		{"value": string(models.AuditActionImport), "label": models.ActionLabels[models.AuditActionImport]},
		{"value": string(models.AuditActionLogin), "label": models.ActionLabels[models.AuditActionLogin]},
		{"value": string(models.AuditActionLogout), "label": models.ActionLabels[models.AuditActionLogout]},
	}
	
	return c.JSON(actions)
}

// getAccessibleTechnicianIDs returns technician IDs the user has access to via hierarchy
func (h *AuditHandler) getAccessibleTechnicianIDs(userID string) ([]string, error) {
	// Get user's memberships
	memberships, err := h.hierarchyRepo.GetUserMemberships(userID)
	if err != nil {
		return nil, err
	}
	
	if len(memberships) == 0 {
		return []string{}, nil
	}
	
	// Get all node IDs the user has access to (including children)
	var nodeIDs []uint
	for _, m := range memberships {
		nodeIDs = append(nodeIDs, m.NodeID)
		
		// Get children nodes recursively
		children, err := h.hierarchyRepo.GetNodeChildren(m.NodeID)
		if err == nil {
			for _, child := range children {
				nodeIDs = append(nodeIDs, child.ID)
				// Get grandchildren
				grandchildren, _ := h.hierarchyRepo.GetNodeChildren(child.ID)
				for _, gc := range grandchildren {
					nodeIDs = append(nodeIDs, gc.ID)
				}
			}
		}
	}
	
	// Get technicians associated with these nodes
	// For now, we'll return all technician IDs (to be refined based on actual business logic)
	// In a real implementation, you'd have a relationship between technicians and nodes
	var technicianIDs []string
	
	err = h.db.Model(&models.Technician{}).
		Pluck("id", &technicianIDs).Error
	if err != nil {
		return nil, err
	}
	
	return technicianIDs, nil
}

// RegisterAuditRoutes registers all audit routes
func RegisterAuditRoutes(app fiber.Router, db *gorm.DB) {
	handler := NewAuditHandler(db)
	
	audit := app.Group("/audit")
	
	// Main endpoints
	audit.Get("/", handler.GetAuditLogs)
	audit.Get("/stats", handler.GetAuditStats)
	audit.Get("/entity-types", handler.GetEntityTypes)
	audit.Get("/action-types", handler.GetActionTypes)
	audit.Get("/users", handler.GetAuditUsers)
	
	// Specific queries
	audit.Get("/:id", handler.GetAuditLogByID)
	audit.Get("/entity/:entityType/:entityId", handler.GetEntityAuditLogs)
	audit.Get("/user/:userId", handler.GetUserAuditLogs)
}
