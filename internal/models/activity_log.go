package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ActivityLog represents a user activity log entry
type ActivityLog struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID      string    `json:"userId" gorm:"type:varchar(36);index;not null"`
	User        *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Action      string    `json:"action" gorm:"type:varchar(100);not null"` // login, logout, create, update, delete, etc.
	Resource    string    `json:"resource" gorm:"type:varchar(100)"`        // ticket, technician, client, etc.
	ResourceID  string    `json:"resourceId" gorm:"type:varchar(36)"`       // ID of the affected resource
	Description string    `json:"description" gorm:"type:text"`             // Human-readable description
	IPAddress   string    `json:"ipAddress" gorm:"type:varchar(45)"`        // IPv4 or IPv6
	UserAgent   string    `json:"userAgent" gorm:"type:text"`               // Browser/device info
	Metadata    string    `json:"metadata" gorm:"type:text"`                // JSON string with additional data
	CreatedAt   time.Time `json:"createdAt" gorm:"index"`
}

func (a *ActivityLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// CreateActivityLogRequest represents the request to create an activity log
type CreateActivityLogRequest struct {
	Action      string `json:"action" validate:"required"`
	Resource    string `json:"resource"`
	ResourceID  string `json:"resourceId"`
	Description string `json:"description"`
	IPAddress   string `json:"ipAddress"`
	UserAgent   string `json:"userAgent"`
	Metadata    string `json:"metadata"`
}

// ActivityLogFilter represents filters for querying activity logs
type ActivityLogFilter struct {
	UserID     string    `json:"userId"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resourceId"`
	StartDate  time.Time `json:"startDate"`
	EndDate    time.Time `json:"endDate"`
}

// PaginatedActivityLogs represents a paginated list of activity logs
type PaginatedActivityLogs struct {
	Data       []ActivityLog `json:"data"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalPages int           `json:"totalPages"`
}
