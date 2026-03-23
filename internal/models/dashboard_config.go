package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DashboardWidget struct {
	ID      string `json:"id"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	W       int    `json:"w"`
	H       int    `json:"h"`
	Visible bool   `json:"visible"`
}

type DashboardWidgets []DashboardWidget

func (w DashboardWidgets) Value() (driver.Value, error) {
	if w == nil {
		return "[]", nil
	}
	return json.Marshal(w)
}

func (w *DashboardWidgets) Scan(value interface{}) error {
	if value == nil {
		*w = []DashboardWidget{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan DashboardWidgets")
	}
	return json.Unmarshal(bytes, w)
}

type UserDashboardConfig struct {
	ID        string           `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string           `json:"userId" gorm:"type:varchar(36);not null;uniqueIndex"`
	User      *User            `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Widgets   DashboardWidgets `json:"widgets" gorm:"type:jsonb;default:'[]'"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

func (c *UserDashboardConfig) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

type SaveDashboardConfigRequest struct {
	Widgets []DashboardWidget `json:"widgets" validate:"required"`
}
