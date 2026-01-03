package handlers

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type HierarchyHandler struct {
	repo     repositories.HierarchyRepository
	validate *validator.Validate
}

func NewHierarchyHandler(repo repositories.HierarchyRepository) *HierarchyHandler {
	return &HierarchyHandler{
		repo:     repo,
		validate: validator.New(),
	}
}

// ==================== Hierarchy Endpoints ====================

// GetAllHierarchies returns all hierarchies
func (h *HierarchyHandler) GetAllHierarchies(c *fiber.Ctx) error {
	hierarchies, err := h.repo.GetAllHierarchies()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch hierarchies",
		})
	}
	return c.JSON(hierarchies)
}

// GetHierarchy returns a hierarchy with its tree structure
func (h *HierarchyHandler) GetHierarchy(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid hierarchy ID",
		})
	}

	hierarchy, err := h.repo.GetHierarchyWithTree(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Hierarchy not found",
		})
	}
	return c.JSON(hierarchy)
}

// CreateHierarchy creates a new hierarchy
func (h *HierarchyHandler) CreateHierarchy(c *fiber.Ctx) error {
	var req models.CreateHierarchyRequest
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

	hierarchy := &models.Hierarchy{
		Name:        req.Name,
		Icon:        req.Icon,
		Description: req.Description,
	}

	if err := h.repo.CreateHierarchy(hierarchy); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create hierarchy",
		})
	}

	// Log the action
	h.logAction(c, "CREATE", "hierarchy", hierarchy.ID, nil, hierarchy)

	return c.Status(fiber.StatusCreated).JSON(hierarchy)
}

// UpdateHierarchy updates a hierarchy
func (h *HierarchyHandler) UpdateHierarchy(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid hierarchy ID",
		})
	}

	existing, err := h.repo.GetHierarchyByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Hierarchy not found",
		})
	}

	var req models.CreateHierarchyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	oldValue := *existing
	existing.Name = req.Name
	existing.Icon = req.Icon
	existing.Description = req.Description

	if err := h.repo.UpdateHierarchy(existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update hierarchy",
		})
	}

	h.logAction(c, "UPDATE", "hierarchy", existing.ID, oldValue, existing)

	return c.JSON(existing)
}

// DeleteHierarchy deletes a hierarchy
func (h *HierarchyHandler) DeleteHierarchy(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid hierarchy ID",
		})
	}

	existing, err := h.repo.GetHierarchyByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Hierarchy not found",
		})
	}

	if err := h.repo.DeleteHierarchy(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete hierarchy",
		})
	}

	h.logAction(c, "DELETE", "hierarchy", uint(id), existing, nil)

	return c.SendStatus(fiber.StatusNoContent)
}

// ==================== Node Endpoints ====================

// GetAllNodes returns all nodes (flat list)
func (h *HierarchyHandler) GetAllNodes(c *fiber.Ctx) error {
	nodes, err := h.repo.GetAllNodes()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch nodes",
		})
	}

	// Convert to DTO for response
	response := make([]fiber.Map, len(nodes))
	for i, node := range nodes {
		response[i] = fiber.Map{
			"id":            node.ID,
			"name":          node.Name,
			"path":          node.Path,
			"hierarchyId":   node.HierarchyID,
			"hierarchyName": "",
		}
		if node.Hierarchy != nil {
			response[i]["hierarchyName"] = node.Hierarchy.Name
		}
	}

	return c.JSON(response)
}

// CreateNode creates a new node in a hierarchy
func (h *HierarchyHandler) CreateNode(c *fiber.Ctx) error {
	hierarchyID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid hierarchy ID",
		})
	}

	// Check if hierarchy exists
	if _, err := h.repo.GetHierarchyByID(uint(hierarchyID)); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Hierarchy not found",
		})
	}

	var req models.CreateNodeRequest
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

	node := &models.Node{
		HierarchyID: uint(hierarchyID),
		ParentID:    req.ParentID,
		Name:        req.Name,
	}

	if err := h.repo.CreateNode(node); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create node: " + err.Error(),
		})
	}

	h.logAction(c, "CREATE", "node", node.ID, nil, node)

	return c.Status(fiber.StatusCreated).JSON(node)
}

// UpdateNode updates a node
func (h *HierarchyHandler) UpdateNode(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid node ID",
		})
	}

	existing, err := h.repo.GetNodeByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Node not found",
		})
	}

	var req models.CreateNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	oldValue := *existing
	existing.Name = req.Name

	if err := h.repo.UpdateNode(existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update node",
		})
	}

	h.logAction(c, "UPDATE", "node", existing.ID, oldValue, existing)

	return c.JSON(existing)
}

// MoveNode moves a node to a new parent
func (h *HierarchyHandler) MoveNode(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid node ID",
		})
	}

	existing, err := h.repo.GetNodeByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Node not found",
		})
	}

	var req models.MoveNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	oldValue := *existing

	if err := h.repo.MoveNode(uint(id), req.NewParentID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to move node: " + err.Error(),
		})
	}

	updatedNode, _ := h.repo.GetNodeByID(uint(id))
	h.logAction(c, "MOVE", "node", existing.ID, oldValue, updatedNode)

	return c.JSON(updatedNode)
}

// DeleteNode deletes a node
func (h *HierarchyHandler) DeleteNode(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid node ID",
		})
	}

	existing, err := h.repo.GetNodeByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Node not found",
		})
	}

	if err := h.repo.DeleteNode(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete node",
		})
	}

	h.logAction(c, "DELETE", "node", uint(id), existing, nil)

	return c.SendStatus(fiber.StatusNoContent)
}

// ==================== Membership Endpoints ====================

// GetNodeMembers returns all members of a node
func (h *HierarchyHandler) GetNodeMembers(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid node ID",
		})
	}

	members, err := h.repo.GetMembersByNode(uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch members",
		})
	}
	return c.JSON(members)
}

// AddNodeMember adds a member to a node
func (h *HierarchyHandler) AddNodeMember(c *fiber.Ctx) error {
	nodeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid node ID",
		})
	}

	// Check if node exists
	if _, err := h.repo.GetNodeByID(uint(nodeID)); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Node not found",
		})
	}

	var req models.AddMemberRequest
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

	// Check for duplicate
	existing, err := h.repo.CheckDuplicateMembership(req.UserID, uint(nodeID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check membership",
		})
	}
	if existing != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":              "User already has access to this node",
			"existingMembership": existing,
		})
	}

	// Get current user ID from context
	currentUserID := c.Locals("userId")
	var grantedBy *string
	if uid, ok := currentUserID.(string); ok {
		grantedBy = &uid
	}

	membership := &models.Membership{
		UserID:    req.UserID,
		NodeID:    uint(nodeID),
		RoleID:    req.RoleID,
		IsDirect:  true,
		GrantedBy: grantedBy,
		GrantedAt: time.Now(),
	}

	if err := h.repo.AddMember(membership); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add member",
		})
	}

	h.logAction(c, "CREATE", "membership", membership.ID, nil, membership)

	return c.Status(fiber.StatusCreated).JSON(membership)
}

// UpdateMembership updates a membership
func (h *HierarchyHandler) UpdateMembership(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid membership ID",
		})
	}

	existing, err := h.repo.GetMembershipByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Membership not found",
		})
	}

	var req models.UpdateMembershipRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	oldValue := *existing
	existing.RoleID = req.RoleID

	if err := h.repo.UpdateMembership(existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update membership",
		})
	}

	h.logAction(c, "UPDATE", "membership", existing.ID, oldValue, existing)

	return c.JSON(existing)
}

// DeleteMembership removes a membership
func (h *HierarchyHandler) DeleteMembership(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid membership ID",
		})
	}

	existing, err := h.repo.GetMembershipByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Membership not found",
		})
	}

	if err := h.repo.RemoveMembership(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove membership",
		})
	}

	h.logAction(c, "DELETE", "membership", uint(id), existing, nil)

	return c.SendStatus(fiber.StatusNoContent)
}

// ==================== Role Endpoints ====================

// GetAllRoles returns all roles
func (h *HierarchyHandler) GetAllRoles(c *fiber.Ctx) error {
	roles, err := h.repo.GetAllRoles()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch roles",
		})
	}
	return c.JSON(roles)
}

// GetRole returns a role with its permissions
func (h *HierarchyHandler) GetRole(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role ID",
		})
	}

	role, err := h.repo.GetRoleWithPermissions(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Role not found",
		})
	}
	return c.JSON(role)
}

// CreateRole creates a new role
func (h *HierarchyHandler) CreateRole(c *fiber.Ctx) error {
	var req models.CreateRoleRequest
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

	role := &models.Role{
		Name:        req.Name,
		Description: req.Description,
	}

	// Get permissions
	if len(req.PermissionIDs) > 0 {
		permissions, _ := h.repo.GetAllPermissions()
		permMap := make(map[uint]models.Permission)
		for _, p := range permissions {
			permMap[p.ID] = p
		}
		for _, pid := range req.PermissionIDs {
			if p, ok := permMap[pid]; ok {
				role.Permissions = append(role.Permissions, p)
			}
		}
	}

	if err := h.repo.CreateRole(role); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create role",
		})
	}

	h.logAction(c, "CREATE", "role", role.ID, nil, role)

	return c.Status(fiber.StatusCreated).JSON(role)
}

// UpdateRole updates a role
func (h *HierarchyHandler) UpdateRole(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role ID",
		})
	}

	existing, err := h.repo.GetRoleWithPermissions(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Role not found",
		})
	}

	if existing.IsSystem {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot modify system role",
		})
	}

	var req models.UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	oldValue := *existing
	existing.Name = req.Name
	existing.Description = req.Description

	// Update permissions
	existing.Permissions = nil
	if len(req.PermissionIDs) > 0 {
		permissions, _ := h.repo.GetAllPermissions()
		permMap := make(map[uint]models.Permission)
		for _, p := range permissions {
			permMap[p.ID] = p
		}
		for _, pid := range req.PermissionIDs {
			if p, ok := permMap[pid]; ok {
				existing.Permissions = append(existing.Permissions, p)
			}
		}
	}

	if err := h.repo.UpdateRole(existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update role: " + err.Error(),
		})
	}

	h.logAction(c, "UPDATE", "role", existing.ID, oldValue, existing)

	return c.JSON(existing)
}

// DeleteRole deletes a role
func (h *HierarchyHandler) DeleteRole(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role ID",
		})
	}

	existing, err := h.repo.GetRoleByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Role not found",
		})
	}

	if err := h.repo.DeleteRole(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete role: " + err.Error(),
		})
	}

	h.logAction(c, "DELETE", "role", uint(id), existing, nil)

	return c.SendStatus(fiber.StatusNoContent)
}

// ==================== Permission Endpoints ====================

// GetAllPermissions returns all permissions grouped by category
func (h *HierarchyHandler) GetAllPermissions(c *fiber.Ctx) error {
	permissions, err := h.repo.GetPermissionsByCategory()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch permissions",
		})
	}
	return c.JSON(permissions)
}

// ==================== Simulation Endpoints ====================

// SimulateAccess simulates access changes
func (h *HierarchyHandler) SimulateAccess(c *fiber.Ctx) error {
	var req models.SimulateAccessRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.repo.SimulateAccess(req.UserID, req.Changes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to simulate access: " + err.Error(),
		})
	}

	return c.JSON(result)
}

// GetUserAccess returns a user's current access
func (h *HierarchyHandler) GetUserAccess(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	access, err := h.repo.GetUserAccess(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch user access",
		})
	}

	return c.JSON(access)
}

// ==================== History Endpoints ====================

// GetHistory returns the audit log
func (h *HierarchyHandler) GetHistory(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	logs, err := h.repo.GetAuditLog(limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch history",
		})
	}

	return c.JSON(logs)
}

// RevertChange reverts a change
func (h *HierarchyHandler) RevertChange(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid log ID",
		})
	}

	if err := h.repo.RevertChange(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to revert change: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Change reverted successfully",
	})
}

// ==================== Helper Functions ====================

func (h *HierarchyHandler) logAction(c *fiber.Ctx, action, entityType string, entityID uint, oldValue, newValue interface{}) {
	var oldJSON, newJSON json.RawMessage
	if oldValue != nil {
		oldJSON, _ = json.Marshal(oldValue)
	}
	if newValue != nil {
		newJSON, _ = json.Marshal(newValue)
	}

	currentUserID := c.Locals("userId")
	var userID *string
	if uid, ok := currentUserID.(string); ok {
		userID = &uid
	}

	log := &models.AccessAuditLog{
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		UserID:     userID,
		OldValue:   oldJSON,
		NewValue:   newJSON,
	}
	h.repo.CreateAuditLog(log)
}
