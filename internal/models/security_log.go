package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SecurityLog represents a security-related event (login, logout, failed attempts, etc.)
type SecurityLog struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID    string    `json:"userId" gorm:"type:varchar(36);index"`
	User      *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Email     string    `json:"email" gorm:"type:varchar(255)"` // Email used in the attempt
	Action    string    `json:"action" gorm:"type:varchar(50);not null;index"` // login_success, login_failed, logout, password_change, etc.
	IPAddress string    `json:"ipAddress" gorm:"type:varchar(45)"`
	UserAgent string    `json:"userAgent" gorm:"type:text"`
	Location  string    `json:"location" gorm:"type:varchar(255)"` // Approximate location based on IP
	Success   bool      `json:"success" gorm:"default:true"`
	Details   string    `json:"details" gorm:"type:text"` // Additional details/error message
	CreatedAt time.Time `json:"createdAt" gorm:"index"`
}

func (s *SecurityLog) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

// SecurityLogFilter for querying security logs
type SecurityLogFilter struct {
	UserID    string    `json:"userId"`
	Email     string    `json:"email"`
	Action    string    `json:"action"`
	Success   *bool     `json:"success"`
	IPAddress string    `json:"ipAddress"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}

// PaginatedSecurityLogs represents paginated security logs
type PaginatedSecurityLogs struct {
	Content       []SecurityLog `json:"content"`
	Page          int           `json:"page"`
	Size          int           `json:"size"`
	TotalElements int64         `json:"totalElements"`
	TotalPages    int           `json:"totalPages"`
}
