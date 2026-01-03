package repositories

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/gorm"
)

type HierarchyRepository interface {
	// Hierarchy CRUD
	GetAllHierarchies() ([]models.Hierarchy, error)
	GetHierarchyByID(id uint) (*models.Hierarchy, error)
	GetHierarchyWithTree(id uint) (*models.HierarchyWithTree, error)
	CreateHierarchy(hierarchy *models.Hierarchy) error
	UpdateHierarchy(hierarchy *models.Hierarchy) error
	DeleteHierarchy(id uint) error

	// Node CRUD
	GetNodeByID(id uint) (*models.Node, error)
	GetNodesByHierarchy(hierarchyID uint) ([]models.Node, error)
	GetNodeChildren(nodeID uint) ([]models.Node, error)
	CreateNode(node *models.Node) error
	UpdateNode(node *models.Node) error
	MoveNode(nodeID uint, newParentID *uint) error
	DeleteNode(id uint) error

	// Role CRUD
	GetAllRoles() ([]models.Role, error)
	GetRoleByID(id uint) (*models.Role, error)
	GetRoleWithPermissions(id uint) (*models.Role, error)
	CreateRole(role *models.Role) error
	UpdateRole(role *models.Role) error
	DeleteRole(id uint) error

	// Permission
	GetAllPermissions() ([]models.Permission, error)
	GetPermissionsByCategory() (map[string][]models.Permission, error)
	GetUserPermissions(userID string) ([]string, error)

	// Membership CRUD
	GetMembersByNode(nodeID uint) ([]models.MemberWithDetails, error)
	GetMembershipByID(id uint) (*models.Membership, error)
	GetUserMemberships(userID string) ([]models.Membership, error)
	AddMember(membership *models.Membership) error
	UpdateMembership(membership *models.Membership) error
	RemoveMembership(id uint) error
	CheckDuplicateMembership(userID string, nodeID uint) (*models.Membership, error)

	// Access simulation
	GetUserAccess(userID string) (*models.UserAccessView, error)
	SimulateAccess(userID string, changes []models.SimulationChange) (*models.SimulateAccessResponse, error)

	// Audit log
	GetAuditLog(limit, offset int) ([]models.AccessAuditLog, error)
	CreateAuditLog(log *models.AccessAuditLog) error
	RevertChange(logID uint) error
}

type hierarchyRepository struct {
	db *gorm.DB
}

func NewHierarchyRepository(db *gorm.DB) HierarchyRepository {
	return &hierarchyRepository{db: db}
}

// ==================== Hierarchy CRUD ====================

func (r *hierarchyRepository) GetAllHierarchies() ([]models.Hierarchy, error) {
	var hierarchies []models.Hierarchy
	err := r.db.Order("name ASC").Find(&hierarchies).Error
	return hierarchies, err
}

func (r *hierarchyRepository) GetHierarchyByID(id uint) (*models.Hierarchy, error) {
	var hierarchy models.Hierarchy
	err := r.db.First(&hierarchy, id).Error
	if err != nil {
		return nil, err
	}
	return &hierarchy, nil
}

func (r *hierarchyRepository) GetHierarchyWithTree(id uint) (*models.HierarchyWithTree, error) {
	var hierarchy models.Hierarchy
	if err := r.db.First(&hierarchy, id).Error; err != nil {
		return nil, err
	}

	// Get all nodes for this hierarchy
	var nodes []models.Node
	if err := r.db.Where("hierarchy_id = ?", id).
		Order("path ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}

	// Get member counts for each node
	memberCounts := make(map[uint]int)
	var counts []struct {
		NodeID uint
		Count  int
	}
	r.db.Model(&models.Membership{}).
		Select("node_id, COUNT(*) as count").
		Where("node_id IN (?)", r.db.Model(&models.Node{}).Select("id").Where("hierarchy_id = ?", id)).
		Group("node_id").
		Scan(&counts)
	for _, c := range counts {
		memberCounts[c.NodeID] = c.Count
	}

	// Build tree structure
	nodeMap := make(map[uint]*models.NodeWithChildren)
	var rootNodes []models.NodeWithChildren

	for _, node := range nodes {
		nwc := models.NodeWithChildren{
			ID:          node.ID,
			Name:        node.Name,
			Path:        node.Path,
			Depth:       node.Depth,
			MemberCount: memberCounts[node.ID],
			Children:    []models.NodeWithChildren{},
		}
		nodeMap[node.ID] = &nwc
	}

	for _, node := range nodes {
		if node.ParentID == nil {
			rootNodes = append(rootNodes, *nodeMap[node.ID])
		} else {
			if parent, ok := nodeMap[*node.ParentID]; ok {
				parent.Children = append(parent.Children, *nodeMap[node.ID])
			}
		}
	}

	// Rebuild with updated children
	for _, node := range nodes {
		if node.ParentID == nil {
			for i, rn := range rootNodes {
				if rn.ID == node.ID {
					rootNodes[i] = r.buildNodeTree(node.ID, nodeMap)
				}
			}
		}
	}

	return &models.HierarchyWithTree{
		Hierarchy: hierarchy,
		RootNodes: rootNodes,
	}, nil
}

func (r *hierarchyRepository) buildNodeTree(nodeID uint, nodeMap map[uint]*models.NodeWithChildren) models.NodeWithChildren {
	node := nodeMap[nodeID]
	result := *node
	result.Children = []models.NodeWithChildren{}
	for _, child := range node.Children {
		result.Children = append(result.Children, r.buildNodeTree(child.ID, nodeMap))
	}
	return result
}

func (r *hierarchyRepository) CreateHierarchy(hierarchy *models.Hierarchy) error {
	return r.db.Create(hierarchy).Error
}

func (r *hierarchyRepository) UpdateHierarchy(hierarchy *models.Hierarchy) error {
	return r.db.Save(hierarchy).Error
}

func (r *hierarchyRepository) DeleteHierarchy(id uint) error {
	return r.db.Delete(&models.Hierarchy{}, id).Error
}

// ==================== Node CRUD ====================

func (r *hierarchyRepository) GetNodeByID(id uint) (*models.Node, error) {
	var node models.Node
	err := r.db.Preload("Hierarchy").First(&node, id).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *hierarchyRepository) GetNodesByHierarchy(hierarchyID uint) ([]models.Node, error) {
	var nodes []models.Node
	err := r.db.Where("hierarchy_id = ?", hierarchyID).Order("path ASC").Find(&nodes).Error
	return nodes, err
}

func (r *hierarchyRepository) GetNodeChildren(nodeID uint) ([]models.Node, error) {
	var children []models.Node
	err := r.db.Where("parent_id = ?", nodeID).Order("name ASC").Find(&children).Error
	return children, err
}

func (r *hierarchyRepository) CreateNode(node *models.Node) error {
	// Calculate path and depth
	if node.ParentID != nil {
		var parent models.Node
		if err := r.db.First(&parent, *node.ParentID).Error; err != nil {
			return fmt.Errorf("parent node not found: %w", err)
		}
		node.Depth = parent.Depth + 1
		// Path will be set after creation with the ID
	} else {
		node.Depth = 0
	}

	if err := r.db.Create(node).Error; err != nil {
		return err
	}

	// Update path with the new ID
	if node.ParentID != nil {
		var parent models.Node
		r.db.First(&parent, *node.ParentID)
		node.Path = fmt.Sprintf("%s.%d", parent.Path, node.ID)
	} else {
		node.Path = fmt.Sprintf("%d", node.ID)
	}
	return r.db.Save(node).Error
}

func (r *hierarchyRepository) UpdateNode(node *models.Node) error {
	return r.db.Model(node).Updates(map[string]interface{}{
		"name": node.Name,
	}).Error
}

func (r *hierarchyRepository) MoveNode(nodeID uint, newParentID *uint) error {
	var node models.Node
	if err := r.db.First(&node, nodeID).Error; err != nil {
		return err
	}

	oldPath := node.Path

	// Calculate new path and depth
	if newParentID != nil {
		var parent models.Node
		if err := r.db.First(&parent, *newParentID).Error; err != nil {
			return fmt.Errorf("new parent node not found: %w", err)
		}
		node.ParentID = newParentID
		node.Depth = parent.Depth + 1
		node.Path = fmt.Sprintf("%s.%d", parent.Path, node.ID)
	} else {
		node.ParentID = nil
		node.Depth = 0
		node.Path = fmt.Sprintf("%d", node.ID)
	}

	// Update the node
	if err := r.db.Save(&node).Error; err != nil {
		return err
	}

	// Update all descendants' paths
	var descendants []models.Node
	r.db.Where("path LIKE ?", oldPath+".%").Find(&descendants)
	for _, desc := range descendants {
		newDescPath := strings.Replace(desc.Path, oldPath, node.Path, 1)
		r.db.Model(&desc).Update("path", newDescPath)
	}

	return nil
}

func (r *hierarchyRepository) DeleteNode(id uint) error {
	// Get the node to find its path
	var node models.Node
	if err := r.db.First(&node, id).Error; err != nil {
		return err
	}

	// Delete all descendants first (nodes with path starting with this node's path)
	r.db.Where("path LIKE ?", node.Path+".%").Delete(&models.Node{})

	// Delete the node itself
	return r.db.Delete(&models.Node{}, id).Error
}

// ==================== Role CRUD ====================

func (r *hierarchyRepository) GetAllRoles() ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Order("is_system DESC, name ASC").Find(&roles).Error
	return roles, err
}

func (r *hierarchyRepository) GetRoleByID(id uint) (*models.Role, error) {
	var role models.Role
	err := r.db.First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *hierarchyRepository) GetRoleWithPermissions(id uint) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions").First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *hierarchyRepository) CreateRole(role *models.Role) error {
	return r.db.Create(role).Error
}

func (r *hierarchyRepository) UpdateRole(role *models.Role) error {
	// Check if it's a system role
	var existing models.Role
	if err := r.db.First(&existing, role.ID).Error; err != nil {
		return err
	}
	if existing.IsSystem {
		return fmt.Errorf("cannot modify system role")
	}

	// Update the role
	if err := r.db.Model(role).Updates(map[string]interface{}{
		"name":        role.Name,
		"description": role.Description,
	}).Error; err != nil {
		return err
	}

	// Update permissions
	return r.db.Model(role).Association("Permissions").Replace(role.Permissions)
}

func (r *hierarchyRepository) DeleteRole(id uint) error {
	// Check if it's a system role
	var role models.Role
	if err := r.db.First(&role, id).Error; err != nil {
		return err
	}
	if role.IsSystem {
		return fmt.Errorf("cannot delete system role")
	}

	// Check if role is in use
	var count int64
	r.db.Model(&models.Membership{}).Where("role_id = ?", id).Count(&count)
	if count > 0 {
		return fmt.Errorf("role is in use by %d memberships", count)
	}

	return r.db.Delete(&models.Role{}, id).Error
}

// ==================== Permission ====================

func (r *hierarchyRepository) GetAllPermissions() ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.Order("category ASC, name ASC").Find(&permissions).Error
	return permissions, err
}

func (r *hierarchyRepository) GetPermissionsByCategory() (map[string][]models.Permission, error) {
	permissions, err := r.GetAllPermissions()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]models.Permission)
	for _, p := range permissions {
		result[p.Category] = append(result[p.Category], p)
	}
	return result, nil
}

// GetUserPermissions returns all permission codes for a user based on their memberships
func (r *hierarchyRepository) GetUserPermissions(userID string) ([]string, error) {
	// Get all memberships with roles preloaded
	var memberships []models.Membership
	if err := r.db.Preload("Role.Permissions").
		Where("user_id = ?", userID).
		Find(&memberships).Error; err != nil {
		return nil, err
	}

	// Collect unique permission codes
	permSet := make(map[string]bool)
	for _, m := range memberships {
		if m.Role != nil {
			for _, p := range m.Role.Permissions {
				permSet[p.Code] = true
			}
		}
	}

	// Convert to slice
	permissions := make([]string, 0, len(permSet))
	for code := range permSet {
		permissions = append(permissions, code)
	}
	return permissions, nil
}

// ==================== Membership CRUD ====================

func (r *hierarchyRepository) GetMembersByNode(nodeID uint) ([]models.MemberWithDetails, error) {
	var memberships []models.Membership
	if err := r.db.Preload("User").Preload("Role").
		Where("node_id = ?", nodeID).
		Order("granted_at DESC").
		Find(&memberships).Error; err != nil {
		return nil, err
	}

	// Get the node for path-based inherited access
	var node models.Node
	r.db.First(&node, nodeID)

	// Also get inherited memberships from parent nodes
	var inheritedMemberships []models.Membership
	if node.Path != "" {
		pathParts := strings.Split(node.Path, ".")
		if len(pathParts) > 1 {
			// Get parent node IDs from path
			parentIDs := make([]uint, 0)
			for i := 0; i < len(pathParts)-1; i++ {
				var id uint
				fmt.Sscanf(pathParts[i], "%d", &id)
				parentIDs = append(parentIDs, id)
			}
			if len(parentIDs) > 0 {
				r.db.Preload("User").Preload("Role").Preload("Node").
					Where("node_id IN ?", parentIDs).
					Find(&inheritedMemberships)
			}
		}
	}

	// Build result with details
	result := make([]models.MemberWithDetails, 0)
	seenUsers := make(map[string]bool)

	// Direct memberships first
	for _, m := range memberships {
		if m.User == nil {
			continue
		}
		seenUsers[m.UserID] = true
		result = append(result, models.MemberWithDetails{
			ID:        m.ID,
			UserID:    m.UserID,
			UserName:  m.User.FullName,
			UserEmail: m.User.Email,
			RoleID:    m.RoleID,
			RoleName:  m.Role.Name,
			IsDirect:  true,
			GrantedAt: m.GrantedAt,
		})
	}

	// Then inherited memberships
	for _, m := range inheritedMemberships {
		if m.User == nil || seenUsers[m.UserID] {
			continue
		}
		seenUsers[m.UserID] = true
		sourceNode := ""
		if m.Node != nil {
			sourceNode = m.Node.Name
		}
		result = append(result, models.MemberWithDetails{
			ID:         m.ID,
			UserID:     m.UserID,
			UserName:   m.User.FullName,
			UserEmail:  m.User.Email,
			RoleID:     m.RoleID,
			RoleName:   m.Role.Name,
			IsDirect:   false,
			SourceNode: sourceNode,
			GrantedAt:  m.GrantedAt,
		})
	}

	return result, nil
}

func (r *hierarchyRepository) GetMembershipByID(id uint) (*models.Membership, error) {
	var membership models.Membership
	err := r.db.Preload("User").Preload("Role").Preload("Node").First(&membership, id).Error
	if err != nil {
		return nil, err
	}
	return &membership, nil
}

func (r *hierarchyRepository) GetUserMemberships(userID string) ([]models.Membership, error) {
	var memberships []models.Membership
	err := r.db.Preload("Node.Hierarchy").Preload("Role").
		Where("user_id = ?", userID).
		Find(&memberships).Error
	return memberships, err
}

func (r *hierarchyRepository) AddMember(membership *models.Membership) error {
	return r.db.Create(membership).Error
}

func (r *hierarchyRepository) UpdateMembership(membership *models.Membership) error {
	return r.db.Model(membership).Update("role_id", membership.RoleID).Error
}

func (r *hierarchyRepository) RemoveMembership(id uint) error {
	return r.db.Delete(&models.Membership{}, id).Error
}

func (r *hierarchyRepository) CheckDuplicateMembership(userID string, nodeID uint) (*models.Membership, error) {
	var membership models.Membership
	err := r.db.Where("user_id = ? AND node_id = ?", userID, nodeID).First(&membership).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &membership, nil
}

// ==================== Access Simulation ====================

func (r *hierarchyRepository) GetUserAccess(userID string) (*models.UserAccessView, error) {
	memberships, err := r.GetUserMemberships(userID)
	if err != nil {
		return nil, err
	}

	nodeAccesses := make([]models.NodeAccess, 0)
	permissionSet := make(map[string]bool)

	for _, m := range memberships {
		// Get node info
		if m.Node != nil {
			nodeAccesses = append(nodeAccesses, models.NodeAccess{
				NodeID:   m.NodeID,
				NodeName: m.Node.Name,
				NodePath: m.Node.Path,
				RoleName: m.Role.Name,
				IsDirect: m.IsDirect,
			})
		}

		// Get permissions from role
		var role models.Role
		r.db.Preload("Permissions").First(&role, m.RoleID)
		for _, p := range role.Permissions {
			permissionSet[p.Code] = true
		}
	}

	permissions := make([]string, 0, len(permissionSet))
	for p := range permissionSet {
		permissions = append(permissions, p)
	}

	return &models.UserAccessView{
		Nodes:       nodeAccesses,
		Permissions: permissions,
	}, nil
}

func (r *hierarchyRepository) SimulateAccess(userID string, changes []models.SimulationChange) (*models.SimulateAccessResponse, error) {
	// Get current state
	before, err := r.GetUserAccess(userID)
	if err != nil {
		return nil, err
	}

	// Simulate changes (we don't actually apply them)
	after := &models.UserAccessView{
		Nodes:       make([]models.NodeAccess, len(before.Nodes)),
		Permissions: make([]string, 0),
	}
	copy(after.Nodes, before.Nodes)

	// Apply simulated changes
	for _, change := range changes {
		switch change.Action {
		case "add":
			var node models.Node
			r.db.First(&node, change.NodeID)
			var role models.Role
			r.db.First(&role, change.RoleID)
			after.Nodes = append(after.Nodes, models.NodeAccess{
				NodeID:   change.NodeID,
				NodeName: node.Name,
				NodePath: node.Path,
				RoleName: role.Name,
				IsDirect: true,
			})
		case "remove":
			for i, n := range after.Nodes {
				if n.NodeID == change.NodeID {
					after.Nodes = append(after.Nodes[:i], after.Nodes[i+1:]...)
					break
				}
			}
		case "update":
			for i, n := range after.Nodes {
				if n.NodeID == change.NodeID {
					var role models.Role
					r.db.First(&role, change.RoleID)
					after.Nodes[i].RoleName = role.Name
					break
				}
			}
		}
	}

	// Recalculate permissions after changes
	permissionSet := make(map[string]bool)
	for _, n := range after.Nodes {
		var role models.Role
		r.db.Preload("Permissions").Where("name = ?", n.RoleName).First(&role)
		for _, p := range role.Permissions {
			permissionSet[p.Code] = true
		}
	}
	for p := range permissionSet {
		after.Permissions = append(after.Permissions, p)
	}

	// Calculate impact
	nodesAdded := len(after.Nodes) - len(before.Nodes)
	nodesRemoved := 0
	if nodesAdded < 0 {
		nodesRemoved = -nodesAdded
		nodesAdded = 0
	}

	level := "low"
	totalChange := nodesAdded + nodesRemoved
	if totalChange > 10 {
		level = "high"
	} else if totalChange > 3 {
		level = "medium"
	}

	return &models.SimulateAccessResponse{
		Before: *before,
		After:  *after,
		Impact: models.ImpactSummary{
			NodesAdded:   nodesAdded,
			NodesRemoved: nodesRemoved,
			Level:        level,
		},
	}, nil
}

// ==================== Audit Log ====================

func (r *hierarchyRepository) GetAuditLog(limit, offset int) ([]models.AccessAuditLog, error) {
	var logs []models.AccessAuditLog
	err := r.db.Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

func (r *hierarchyRepository) CreateAuditLog(log *models.AccessAuditLog) error {
	return r.db.Create(log).Error
}

func (r *hierarchyRepository) RevertChange(logID uint) error {
	var log models.AccessAuditLog
	if err := r.db.First(&log, logID).Error; err != nil {
		return err
	}

	// Revert based on entity type and action
	switch log.EntityType {
	case "membership":
		switch log.Action {
		case "CREATE":
			// Delete the created membership
			return r.db.Delete(&models.Membership{}, log.EntityID).Error
		case "DELETE":
			// Recreate the deleted membership
			var membership models.Membership
			if err := json.Unmarshal(log.OldValue, &membership); err != nil {
				return err
			}
			return r.db.Create(&membership).Error
		case "UPDATE":
			// Restore old value
			var oldMembership models.Membership
			if err := json.Unmarshal(log.OldValue, &oldMembership); err != nil {
				return err
			}
			return r.db.Model(&models.Membership{}).Where("id = ?", log.EntityID).
				Update("role_id", oldMembership.RoleID).Error
		}
	}

	return fmt.Errorf("cannot revert this change type")
}
