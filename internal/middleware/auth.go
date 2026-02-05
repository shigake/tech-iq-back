package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

var hierarchyRepo repositories.HierarchyRepository

func SetHierarchyRepository(repo repositories.HierarchyRepository) {
	hierarchyRepo = repo
}

func JWTProtected(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Support both "Bearer <token>" and just "<token>" format
		tokenString := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid signing method")
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}

		// Store user info in context for later use
		c.Locals("userId", claims["userId"])
		c.Locals("email", claims["email"])
		c.Locals("userRole", claims["role"])

		return c.Next()
	}
}

// AdminOnly - deprecated, permissions are now handled by the profile system
// Kept for compatibility, does nothing
func AdminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

// AdminOrEmployee - deprecated, permissions are now handled by the profile system
// Kept for compatibility, does nothing
func AdminOrEmployee() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

// WriteAccess - deprecated, permissions are now handled by the profile system
// Kept for compatibility, does nothing
func WriteAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

func getUserPermissions(userID string, userRole string) []string {
	if hierarchyRepo == nil {
		return nil
	}
	
	permissions, err := hierarchyRepo.GetUserPermissions(userID)
	if err != nil || len(permissions) == 0 {
		permissions, _ = hierarchyRepo.GetPermissionsByRoleName(userRole)
	}
	return permissions
}

func RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userId").(string)
		if !ok || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		userRole, _ := c.Locals("userRole").(string)
		permissions := getUserPermissions(userID, userRole)
		if permissions == nil || len(permissions) == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied: no permissions found",
			})
		}

		for _, p := range permissions {
			if p == permission || p == "*" {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied: missing permission " + permission,
		})
	}
}

func RequireAnyPermission(permissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userId").(string)
		if !ok || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		userRole, _ := c.Locals("userRole").(string)
		userPerms := getUserPermissions(userID, userRole)
		if userPerms == nil || len(userPerms) == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied: no permissions found",
			})
		}

		for _, userPerm := range userPerms {
			if userPerm == "*" {
				return c.Next()
			}
			for _, required := range permissions {
				if userPerm == required {
					return c.Next()
				}
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied: missing required permission",
		})
	}
}
