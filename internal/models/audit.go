package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditAction represents the type of action performed
type AuditAction string

const (
	AuditActionCreate AuditAction = "CREATE"
	AuditActionUpdate AuditAction = "UPDATE"
	AuditActionDelete AuditAction = "DELETE"
	AuditActionView   AuditAction = "VIEW"
	AuditActionExport AuditAction = "EXPORT"
	AuditActionImport AuditAction = "IMPORT"
	AuditActionLogin  AuditAction = "LOGIN"
	AuditActionLogout AuditAction = "LOGOUT"
)

// AuditLog represents a generic audit log entry for any entity
type AuditLog struct {
	ID          string          `json:"id" gorm:"type:varchar(36);primaryKey"`
	Action      AuditAction     `json:"action" gorm:"type:varchar(20);not null;index"`
	EntityType  string          `json:"entityType" gorm:"type:varchar(50);not null;index"` // technician, client, ticket, user, etc.
	EntityID    string          `json:"entityId" gorm:"type:varchar(36);not null;index"`
	EntityName  string          `json:"entityName" gorm:"type:varchar(255)"` // Human-readable name for display
	UserID      *string         `json:"userId" gorm:"type:varchar(36);index"`
	User        *User           `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
	UserName    string          `json:"userName" gorm:"type:varchar(255)"`    // Cached user name for performance
	UserEmail   string          `json:"userEmail" gorm:"type:varchar(255)"`   // Cached email for display
	OldValue    json.RawMessage `json:"oldValue,omitempty" gorm:"type:jsonb"` // Previous state
	NewValue    json.RawMessage `json:"newValue,omitempty" gorm:"type:jsonb"` // New state
	Changes     json.RawMessage `json:"changes,omitempty" gorm:"type:jsonb"`  // Diff of changed fields only
	IPAddress   string          `json:"ipAddress" gorm:"type:varchar(45)"`    // IPv4 or IPv6
	UserAgent   string          `json:"userAgent" gorm:"type:text"`
	Description string          `json:"description" gorm:"type:text"`         // Optional human-readable description
	Metadata    json.RawMessage `json:"metadata,omitempty" gorm:"type:jsonb"` // Extra context (e.g., reason for change)
	CreatedAt   time.Time       `json:"createdAt" gorm:"index"`
}

// BeforeCreate generates UUID for new audit log entries
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// TableName returns the table name for AuditLog
func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditLogFilter represents filter options for querying audit logs
type AuditLogFilter struct {
	EntityType  string      `json:"entityType"`
	EntityID    string      `json:"entityId"`
	Action      AuditAction `json:"action"`
	UserID      string      `json:"userId"`
	StartDate   *time.Time  `json:"startDate"`
	EndDate     *time.Time  `json:"endDate"`
	SearchQuery string      `json:"searchQuery"`
	Page        int         `json:"page"`
	Size        int         `json:"size"`
}

// AuditLogResponse represents the response for audit log queries
type AuditLogResponse struct {
	Items      []AuditLog `json:"items"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	Size       int        `json:"size"`
	TotalPages int        `json:"totalPages"`
}

// EntityTypeLabels maps entity types to human-readable labels
var EntityTypeLabels = map[string]string{
	"technician": "Técnico",
	"client":     "Cliente",
	"ticket":     "Chamado",
	"user":       "Usuário",
	"category":   "Categoria",
	"stock_item": "Item de Estoque",
	"stock":      "Estoque",
	"financial":  "Financeiro",
	"hierarchy":  "Hierarquia",
	"node":       "Nó de Hierarquia",
	"role":       "Perfil",
	"membership": "Permissão de Usuário",
	"permission": "Permissão",
	"geo":        "Geolocalização",
	"settings":   "Configurações",
}

// ActionLabels maps actions to human-readable labels
var ActionLabels = map[AuditAction]string{
	AuditActionCreate: "Criação",
	AuditActionUpdate: "Alteração",
	AuditActionDelete: "Exclusão",
	AuditActionView:   "Visualização",
	AuditActionExport: "Exportação",
	AuditActionImport: "Importação",
	AuditActionLogin:  "Login",
	AuditActionLogout: "Logout",
}

// CreateAuditLogRequest represents the request to manually create an audit log
type CreateAuditLogRequest struct {
	Action      AuditAction     `json:"action" validate:"required"`
	EntityType  string          `json:"entityType" validate:"required"`
	EntityID    string          `json:"entityId" validate:"required"`
	EntityName  string          `json:"entityName"`
	OldValue    json.RawMessage `json:"oldValue,omitempty"`
	NewValue    json.RawMessage `json:"newValue,omitempty"`
	Description string          `json:"description"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// AuditStats represents statistics for audit logs
type AuditStats struct {
	TotalLogs      int64            `json:"totalLogs"`
	LogsByAction   map[string]int64 `json:"logsByAction"`
	LogsByEntity   map[string]int64 `json:"logsByEntity"`
	LogsByUser     []UserAuditStat  `json:"logsByUser"`
	RecentActivity []AuditLog       `json:"recentActivity"`
}

// UserAuditStat represents audit statistics per user
type UserAuditStat struct {
	UserID    string `json:"userId"`
	UserName  string `json:"userName"`
	UserEmail string `json:"userEmail"`
	Count     int64  `json:"count"`
}
