package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID             string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Email          string    `json:"email" gorm:"uniqueIndex;not null;type:varchar(255)"`
	Password       string    `json:"-" gorm:"not null"`
	FirstName      string    `json:"firstName" gorm:"type:varchar(100)"`
	LastName       string    `json:"lastName" gorm:"type:varchar(100)"`
	FullName       string    `json:"fullName" gorm:"type:varchar(255)"`
	Role           string    `json:"role" gorm:"type:varchar(50);default:USER"` // ADMIN, EMPLOYEE, USER
	ProfilePicture string    `json:"profilePicture" gorm:"type:text"`
	Active         bool      `json:"active" gorm:"default:true"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	// Generate FullName
	if u.FullName == "" {
		u.FullName = u.FirstName + " " + u.LastName
	}
	return nil
}

func (u *User) BeforeSave(tx *gorm.DB) error {
	// Update FullName on save
	u.FullName = u.FirstName + " " + u.LastName
	return nil
}

// SignInRequest represents the login request
type SignInRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// SignUpRequest represents the registration request
type SignUpRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
	FirstName string `json:"firstName" validate:"required,min=2"`
	LastName  string `json:"lastName" validate:"required,min=2"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	Token     string `json:"token"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

// ChangePasswordRequest represents the change password request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required,min=6"`
	NewPassword     string `json:"newPassword" validate:"required,min=6"`
}

// CreateUserRequest represents admin request to create a new user
type CreateUserRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
	FirstName string `json:"firstName" validate:"required,min=2"`
	LastName  string `json:"lastName" validate:"required,min=2"`
	Role      string `json:"role" validate:"required,oneof=ADMIN EMPLOYEE USER"`
}

// UpdateUserRequest represents admin request to update a user
type UpdateUserRequest struct {
	Email     string `json:"email" validate:"omitempty,email"`
	FirstName string `json:"firstName" validate:"omitempty,min=2"`
	LastName  string `json:"lastName" validate:"omitempty,min=2"`
	Role      string `json:"role" validate:"omitempty,oneof=ADMIN EMPLOYEE USER"`
	Active    *bool  `json:"active"`
}

// ResetPasswordRequest represents admin request to reset user password
type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword" validate:"required,min=6"`
}

// UserResponse represents a user response without sensitive data
type UserResponse struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	FullName       string `json:"fullName"`
	Role           string `json:"role"`
	ProfilePicture string `json:"profilePicture,omitempty"`
	Active         bool   `json:"active"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

// ToResponse converts a User to UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:             u.ID,
		Email:          u.Email,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		FullName:       u.FullName,
		Role:           u.Role,
		ProfilePicture: u.ProfilePicture,
		Active:         u.Active,
		CreatedAt:      u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
