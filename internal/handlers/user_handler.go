package handlers

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	repo     repositories.UserRepository
	validate *validator.Validate
}

func NewUserHandler(repo repositories.UserRepository) *UserHandler {
	return &UserHandler{
		repo:     repo,
		validate: validator.New(),
	}
}

// GetUsers returns all users with pagination
// @Summary List users
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param search query string false "Search term"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/users [get]
func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
	// Check if user is admin
	userRole := c.Locals("userRole").(string)
	if userRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can list users",
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	search := c.Query("search", "")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	users, total, err := h.repo.GetAllPaginated(page, limit, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch users",
		})
	}

	// Convert to response format
	userResponses := make([]models.UserResponse, len(users))
	for i, u := range users {
		userResponses[i] = u.ToResponse()
	}

	return c.JSON(fiber.Map{
		"data":  userResponses,
		"total": total,
		"page":  page,
		"limit": limit,
		"pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// GetUser returns a single user by ID
// @Summary Get user by ID
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} models.UserResponse
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	userRole := c.Locals("userRole").(string)
	currentUserID := c.Locals("userID").(string)
	requestedID := c.Params("id")

	// Users can view their own profile, admins can view anyone
	if userRole != "ADMIN" && currentUserID != requestedID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	user, err := h.repo.FindByID(requestedID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(user.ToResponse())
}

// CreateUser creates a new user (admin only)
// @Summary Create user
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.CreateUserRequest true "User data"
// @Success 201 {object} models.UserResponse
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	// Check if user is admin
	userRole := c.Locals("userRole").(string)
	if userRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can create users",
		})
	}

	var req models.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": formatValidationErrors(err),
		})
	}

	// Check if email already exists
	existingUser, _ := h.repo.FindByEmail(req.Email)
	if existingUser != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Email already registered",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	user := &models.User{
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      req.Role,
		Active:    true,
	}

	if err := h.repo.Create(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(user.ToResponse())
}

// UpdateUser updates a user (admin only, or self for profile)
// @Summary Update user
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body models.UpdateUserRequest true "User data"
// @Success 200 {object} models.UserResponse
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	userRole := c.Locals("userRole").(string)
	currentUserID := c.Locals("userID").(string)
	targetID := c.Params("id")

	// Only admins can update other users
	if userRole != "ADMIN" && currentUserID != targetID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	var req models.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": formatValidationErrors(err),
		})
	}

	user, err := h.repo.FindByID(targetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Non-admins cannot change role or active status
	if userRole != "ADMIN" {
		req.Role = ""
		req.Active = nil
	}

	// Update fields if provided
	if req.Email != "" && req.Email != user.Email {
		// Check if email is taken
		existing, _ := h.repo.FindByEmail(req.Email)
		if existing != nil && existing.ID != user.ID {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Email already in use",
			})
		}
		user.Email = req.Email
	}

	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Active != nil {
		user.Active = *req.Active
	}

	if err := h.repo.Update(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}

	return c.JSON(user.ToResponse())
}

// DeleteUser deletes a user (admin only)
// @Summary Delete user
// @Tags Users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 204
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	userRole := c.Locals("userRole").(string)
	if userRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can delete users",
		})
	}

	currentUserID := c.Locals("userID").(string)
	targetID := c.Params("id")

	// Cannot delete yourself
	if currentUserID == targetID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete your own account",
		})
	}

	user, err := h.repo.FindByID(targetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Don't allow deleting the last admin
	if user.Role == "ADMIN" {
		count, _ := h.repo.CountByRole("ADMIN")
		if count <= 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot delete the last admin user",
			})
		}
	}

	if err := h.repo.Delete(targetID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ResetPassword resets a user's password (admin only)
// @Summary Reset user password
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body models.ResetPasswordRequest true "New password"
// @Success 200 {object} map[string]string
// @Router /api/v1/users/{id}/reset-password [post]
func (h *UserHandler) ResetPassword(c *fiber.Ctx) error {
	userRole := c.Locals("userRole").(string)
	if userRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can reset passwords",
		})
	}

	targetID := c.Params("id")

	var req models.ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": formatValidationErrors(err),
		})
	}

	user, err := h.repo.FindByID(targetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	user.Password = string(hashedPassword)

	if err := h.repo.Update(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reset password",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}

// ToggleUserStatus toggles user active status (admin only)
// @Summary Toggle user status
// @Tags Users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} models.UserResponse
// @Router /api/v1/users/{id}/toggle-status [post]
func (h *UserHandler) ToggleUserStatus(c *fiber.Ctx) error {
	userRole := c.Locals("userRole").(string)
	if userRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can toggle user status",
		})
	}

	currentUserID := c.Locals("userID").(string)
	targetID := c.Params("id")

	// Cannot deactivate yourself
	if currentUserID == targetID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot deactivate your own account",
		})
	}

	user, err := h.repo.FindByID(targetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	user.Active = !user.Active

	if err := h.repo.Update(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user status",
		})
	}

	return c.JSON(user.ToResponse())
}

// SearchUsers searches users by name or email
// @Summary Search users
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Max results" default(10)
// @Success 200 {array} models.UserResponse
// @Router /api/v1/users/search [get]
func (h *UserHandler) SearchUsers(c *fiber.Ctx) error {
	query := c.Query("q", "")
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if query == "" {
		return c.JSON([]models.UserResponse{})
	}

	if limit < 1 || limit > 50 {
		limit = 10
	}

	users, _, err := h.repo.GetAllPaginated(1, limit, query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search users",
		})
	}

	responses := make([]models.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse()
	}

	return c.JSON(responses)
}
