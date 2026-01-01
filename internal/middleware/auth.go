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
		c.Locals("role", claims["role"])

		return c.Next()
	}
}

func AdminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("role")
		if role != "ADMIN" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin access required",
			})
		}
		return c.Next()
	}
}

// AdminOrEmployee allows ADMIN and EMPLOYEE roles
func AdminOrEmployee() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("role")
		if role != "ADMIN" && role != "EMPLOYEE" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Employee or admin access required",
			})
		}
		return c.Next()
	}
}

// WriteAccess restricts modification operations to ADMIN and EMPLOYEE
// Technicians (USER role) can only read
func WriteAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("role")
		method := c.Method()
		
		// GET requests allowed for all authenticated users
		if method == "GET" {
			return c.Next()
		}
		
		// POST, PUT, DELETE, PATCH require ADMIN or EMPLOYEE role
		if role != "ADMIN" && role != "EMPLOYEE" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Write access denied. Read-only access for technicians.",
			})
		}
		return c.Next()
	}
}
