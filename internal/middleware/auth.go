package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

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
