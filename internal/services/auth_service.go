package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/shigake/tech-iq-back/internal/config"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	SignIn(req *models.SignInRequest, ipAddress, userAgent string) (*models.AuthResponse, error)
	SignUp(req *models.SignUpRequest) (*models.AuthResponse, error)
	RefreshToken(tokenString string) (*models.AuthResponse, error)
	ChangePassword(userID string, req *models.ChangePasswordRequest, ipAddress, userAgent string) error
}

type authService struct {
	userRepo        repositories.UserRepository
	securityLogRepo repositories.SecurityLogRepository
	hierarchyRepo   repositories.HierarchyRepository
	config          *config.Config
}

func NewAuthService(userRepo repositories.UserRepository, securityLogRepo repositories.SecurityLogRepository, hierarchyRepo repositories.HierarchyRepository, config *config.Config) AuthService {
	return &authService{
		userRepo:        userRepo,
		securityLogRepo: securityLogRepo,
		hierarchyRepo:   hierarchyRepo,
		config:          config,
	}
}

func (s *authService) logSecurityEvent(userID, email, action, ipAddress, userAgent, details string, success bool) {
	secLog := &models.SecurityLog{
		UserID:    userID,
		Email:     email,
		Action:    action,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details:   details,
		Success:   success,
		CreatedAt: time.Now(),
	}
	if err := s.securityLogRepo.Create(secLog); err != nil {
		log.Printf("Failed to log security event: %v", err)
	}
}

func (s *authService) SignIn(req *models.SignInRequest, ipAddress, userAgent string) (*models.AuthResponse, error) {
	log.Printf("SignIn attempt for email: %s", req.Email)

	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		s.logSecurityEvent("", req.Email, "login_failed", ipAddress, userAgent, "User not found", false)
		return nil, errors.New("invalid credentials")
	}

	// User found, proceeding with authentication

	if !user.Active {
		s.logSecurityEvent(user.ID, req.Email, "login_failed", ipAddress, userAgent, "Account deactivated", false)
		return nil, errors.New("user account is deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.logSecurityEvent(user.ID, req.Email, "login_failed", ipAddress, userAgent, "Invalid password", false)
		return nil, errors.New("invalid credentials")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	// Log successful login
	s.logSecurityEvent(user.ID, user.Email, "login_success", ipAddress, userAgent, "", true)

	// Get user permissions from hierarchy
	var permissions []string
	if user.Role == "ADMIN" {
		// Admin has all permissions
		allPerms, _ := s.hierarchyRepo.GetAllPermissions()
		permissions = make([]string, len(allPerms))
		for i, p := range allPerms {
			permissions[i] = p.Code
		}
	} else {
		// Get permissions from user memberships
		permissions, _ = s.hierarchyRepo.GetUserPermissions(user.ID)
	}

	return &models.AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Email:        user.Email,
		Role:         user.Role,
		Permissions:  permissions,
	}, nil
}

func (s *authService) SignUp(req *models.SignUpRequest) (*models.AuthResponse, error) {
	// Check if user already exists
	existingUser, _ := s.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      "USER",
		Active:    true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	// New users have no permissions until assigned
	return &models.AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Email:        user.Email,
		Role:         user.Role,
		Permissions:  []string{},
	}, nil
}

func (s *authService) RefreshToken(tokenString string) (*models.AuthResponse, error) {
	// Parse the token - could be access token (for backwards compatibility) or refresh token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Check if this is a refresh token (has "type": "refresh")
	// If it's an access token without type, we still allow it for backwards compatibility
	tokenType, hasType := claims["type"].(string)
	if hasType && tokenType != "refresh" {
		return nil, errors.New("invalid token type, expected refresh token")
	}

	// Handle both string and float64 user IDs from JWT
	var userID string
	switch v := claims["userId"].(type) {
	case string:
		userID = v
	case float64:
		// Legacy support for numeric IDs, convert to string
		userID = fmt.Sprintf("%.0f", v)
	default:
		return nil, errors.New("invalid user ID in token")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Check if user is still active
	if !user.Active {
		return nil, errors.New("user account is deactivated")
	}

	newToken, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	// Generate new refresh token only if the old one is about to expire (less than 1 day left)
	var newRefreshToken string
	if exp, ok := claims["exp"].(float64); ok {
		expTime := time.Unix(int64(exp), 0)
		if time.Until(expTime) < 24*time.Hour {
			newRefreshToken, err = s.generateRefreshToken(user)
			if err != nil {
				return nil, err
			}
		}
	}

	// Get user permissions
	var permissions []string
	if user.Role == "ADMIN" {
		allPerms, _ := s.hierarchyRepo.GetAllPermissions()
		permissions = make([]string, len(allPerms))
		for i, p := range allPerms {
			permissions[i] = p.Code
		}
	} else {
		permissions, _ = s.hierarchyRepo.GetUserPermissions(user.ID)
	}

	return &models.AuthResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken, // Empty if not renewed
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Email:        user.Email,
		Role:         user.Role,
		Permissions:  permissions,
	}, nil
}

func (s *authService) generateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"userId":    user.ID,
		"email":     user.Email,
		"role":      user.Role,
		"roles":     []string{user.Role},
		"type":      "access",
		"createdAt": time.Now().UnixMilli(),
		"iat":       time.Now().Unix(),
		"exp":       time.Now().Add(s.config.JWTExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *authService) generateRefreshToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"userId": user.ID,
		"email":  user.Email,
		"type":   "refresh",
		"iat":    time.Now().Unix(),
		"exp":    time.Now().Add(s.config.JWTRefreshExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *authService) ChangePassword(userID string, req *models.ChangePasswordRequest, ipAddress, userAgent string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		s.logSecurityEvent(userID, user.Email, "password_change_failed", ipAddress, userAgent, "Invalid current password", false)
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password
	user.Password = string(hashedPassword)
	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	// Log successful password change
	s.logSecurityEvent(userID, user.Email, "password_change", ipAddress, userAgent, "", true)
	return nil
}
