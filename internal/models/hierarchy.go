package models

import (
	"encoding/json"
	"time"
)

// Hierarchy represents a hierarchical structure (e.g., Projects, Departments)
type Hierarchy struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"not null;type:varchar(100)"`
	Icon        string    `json:"icon" gorm:"type:varchar(50)"`
	Description string    `json:"description" gorm:"type:text"`
	Nodes       []Node    `json:"nodes,omitempty" gorm:"foreignKey:HierarchyID"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Node represents a node within a hierarchy (e.g., a project, a phase)
type Node struct {
	ID          uint         `json:"id" gorm:"primaryKey;autoIncrement"`
	HierarchyID uint         `json:"hierarchyId" gorm:"not null;index"`
	Hierarchy   *Hierarchy   `json:"hierarchy,omitempty" gorm:"foreignKey:HierarchyID"`
	ParentID    *uint        `json:"parentId" gorm:"index"`
	Parent      *Node        `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Name        string       `json:"name" gorm:"not null;type:varchar(100)"`
	Path        string       `json:"path" gorm:"type:varchar(500);index"` // e.g., "1.2.3" for ltree-like queries
	Depth       int          `json:"depth" gorm:"default:0"`
	Children    []Node       `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	Members     []Membership `json:"members,omitempty" gorm:"foreignKey:NodeID"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

// Role represents an access profile (e.g., Admin, Manager, Operator)
type Role struct {
	ID          uint         `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string       `json:"name" gorm:"not null;type:varchar(100)"`
	Description string       `json:"description" gorm:"type:text"`
	IsSystem    bool         `json:"isSystem" gorm:"default:false"` // true for ADMIN (not editable)
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

// Permission represents a granular permission (e.g., tickets.view, tickets.create)
type Permission struct {
	ID          uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Code        string `json:"code" gorm:"uniqueIndex;not null;type:varchar(100)"` // e.g., "tickets.view"
	Name        string `json:"name" gorm:"not null;type:varchar(100)"`
	Category    string `json:"category" gorm:"not null;type:varchar(50);index"` // Tickets, Finance, Inventory
	Description string `json:"description" gorm:"type:text"`
}

// Membership represents a user's access to a node with a specific role
type Membership struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    string    `json:"userId" gorm:"not null;type:varchar(36);index"`
	User      *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	NodeID    uint      `json:"nodeId" gorm:"not null;index"`
	Node      *Node     `json:"node,omitempty" gorm:"foreignKey:NodeID"`
	RoleID    uint      `json:"roleId" gorm:"not null;index"`
	Role      *Role     `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	IsDirect  bool      `json:"isDirect" gorm:"default:true"` // false if inherited
	GrantedBy *string   `json:"grantedBy" gorm:"type:varchar(36)"`
	GrantedAt time.Time `json:"grantedAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AccessAuditLog represents the history of access changes
type AccessAuditLog struct {
	ID         uint            `json:"id" gorm:"primaryKey;autoIncrement"`
	Action     string          `json:"action" gorm:"not null;type:varchar(50)"` // CREATE, UPDATE, DELETE
	EntityType string          `json:"entityType" gorm:"not null;type:varchar(50);index"` // membership, role, node
	EntityID   uint            `json:"entityId" gorm:"not null;index"`
	UserID     *string         `json:"userId" gorm:"type:varchar(36)"`
	User       *User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
	OldValue   json.RawMessage `json:"oldValue" gorm:"type:jsonb"`
	NewValue   json.RawMessage `json:"newValue" gorm:"type:jsonb"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// ======= REQUEST/RESPONSE DTOs =======

// CreateHierarchyRequest represents the request to create a hierarchy
type CreateHierarchyRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
}

// CreateNodeRequest represents the request to create a node
type CreateNodeRequest struct {
	ParentID *uint  `json:"parentId"`
	Name     string `json:"name" validate:"required,min=2,max=100"`
}

// MoveNodeRequest represents the request to move a node
type MoveNodeRequest struct {
	NewParentID *uint `json:"newParentId"`
}

// CreateRoleRequest represents the request to create a role
type CreateRoleRequest struct {
	Name          string `json:"name" validate:"required,min=2,max=100"`
	Description   string `json:"description"`
	PermissionIDs []uint `json:"permissionIds"`
}

// UpdateRoleRequest represents the request to update a role
type UpdateRoleRequest struct {
	Name          string `json:"name" validate:"required,min=2,max=100"`
	Description   string `json:"description"`
	PermissionIDs []uint `json:"permissionIds"`
}

// AddMemberRequest represents the request to add a member to a node
type AddMemberRequest struct {
	UserID string `json:"userId" validate:"required"`
	RoleID uint   `json:"roleId" validate:"required"`
}

// UpdateMembershipRequest represents the request to update a membership
type UpdateMembershipRequest struct {
	RoleID uint `json:"roleId" validate:"required"`
}

// SimulateAccessRequest represents the request to simulate access changes
type SimulateAccessRequest struct {
	UserID  string            `json:"userId" validate:"required"`
	Changes []SimulationChange `json:"changes"`
}

// SimulationChange represents a single change in the simulation
type SimulationChange struct {
	Action   string `json:"action"` // add, remove, update
	NodeID   uint   `json:"nodeId"`
	RoleID   uint   `json:"roleId,omitempty"`
	OldRoleID uint  `json:"oldRoleId,omitempty"`
}

// SimulateAccessResponse represents the response of access simulation
type SimulateAccessResponse struct {
	Before      UserAccessView `json:"before"`
	After       UserAccessView `json:"after"`
	Impact      ImpactSummary  `json:"impact"`
}

// UserAccessView represents a user's current access state
type UserAccessView struct {
	Nodes       []NodeAccess      `json:"nodes"`
	Permissions []string          `json:"permissions"`
}

// NodeAccess represents access to a specific node
type NodeAccess struct {
	NodeID     uint   `json:"nodeId"`
	NodeName   string `json:"nodeName"`
	NodePath   string `json:"nodePath"`
	RoleName   string `json:"roleName"`
	IsDirect   bool   `json:"isDirect"`
	SourceNode string `json:"sourceNode,omitempty"` // For inherited access
}

// ImpactSummary represents the impact of access changes
type ImpactSummary struct {
	NodesAdded    int    `json:"nodesAdded"`
	NodesRemoved  int    `json:"nodesRemoved"`
	Level         string `json:"level"` // low, medium, high
}

// HierarchyWithTree represents a hierarchy with its full tree structure
type HierarchyWithTree struct {
	Hierarchy
	RootNodes []NodeWithChildren `json:"rootNodes"`
}

// NodeWithChildren represents a node with its children for tree rendering
type NodeWithChildren struct {
	ID          uint               `json:"id"`
	Name        string             `json:"name"`
	Path        string             `json:"path"`
	Depth       int                `json:"depth"`
	MemberCount int                `json:"memberCount"`
	Children    []NodeWithChildren `json:"children"`
}

// MemberWithDetails represents a membership with user and role details
type MemberWithDetails struct {
	ID        uint   `json:"id"`
	UserID    string `json:"userId"`
	UserName  string `json:"userName"`
	UserEmail string `json:"userEmail"`
	RoleID    uint   `json:"roleId"`
	RoleName  string `json:"roleName"`
	IsDirect  bool   `json:"isDirect"`
	SourceNode string `json:"sourceNode,omitempty"`
	GrantedAt time.Time `json:"grantedAt"`
}
