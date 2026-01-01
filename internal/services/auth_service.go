package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tech-erp/backend/internal/config"
	"github.com/tech-erp/backend/internal/models"
	"github.com/tech-erp/backend/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	SignIn(req *models.SignInRequest) (*models.AuthResponse, error)
	SignUp(req *models.SignUpRequest) (*models.AuthResponse, error)
	RefreshToken(tokenString string) (*models.AuthResponse, error)
	ChangePassword(userID string, req *models.ChangePasswordRequest) error
}

type authService struct {
	userRepo repositories.UserRepository
	config   *config.Config
}

func NewAuthService(userRepo repositories.UserRepository, config *config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		config:   config,
	}
}

func (s *authService) SignIn(req *models.SignInRequest) (*models.AuthResponse, error) {
	log.Printf("SignIn attempt for email: %s", req.Email)
	
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		log.Printf("User not found: %v", err)
		return nil, errors.New("invalid credentials")
	}

	log.Printf("User found: ID=%s, Email=%s, Active=%v, PasswordLen=%d", user.ID, user.Email, user.Active, len(user.Password))

	if !user.Active {
		log.Printf("User account is deactivated")
		return nil, errors.New("user account is deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		log.Printf("Password mismatch: %v", err)
		return nil, errors.New("invalid credentials")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:     token,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Role:      user.Role,
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

	return &models.AuthResponse{
		Token:     token,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Role:      user.Role,
	}, nil
}

func (s *authService) RefreshToken(tokenString string) (*models.AuthResponse, error) {
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

	newToken, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:     newToken,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Role:      user.Role,
	}, nil
}

func (s *authService) generateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"userId":    user.ID,
		"email":     user.Email,
		"role":      user.Role,
		"roles":     []string{user.Role},
		"createdAt": time.Now().UnixMilli(),
		"iat":       time.Now().Unix(),
		"exp":       time.Now().Add(s.config.JWTExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *authService) ChangePassword(userID string, req *models.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password
	user.Password = string(hashedPassword)
	return s.userRepo.Update(user)
}
