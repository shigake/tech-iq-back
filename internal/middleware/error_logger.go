package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/services"

	"github.com/gofiber/fiber/v2"
)

// ErrorLoggerMiddleware creates a middleware that logs errors to the database
func ErrorLoggerMiddleware(errorLogService *services.ErrorLogService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Continue with request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start).Milliseconds()
		statusCode := c.Response().StatusCode()

		// Only log errors (status >= 400)
		if statusCode >= 400 {
			go logError(c, errorLogService, statusCode, duration, err)
		}

		return err
	}
}

func logError(c *fiber.Ctx, service *services.ErrorLogService, statusCode int, duration int64, handlerErr error) {
	// Recover from any panic in logging
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Error logging failed: %v\n", r)
		}
	}()

	method := c.Method()
	path := c.Path()
	
	// Get feature name
	feature := models.GetFeatureName(method, path)
	
	// Determine action from method
	action := getActionFromMethod(method)
	
	// Determine error level
	level := getErrorLevel(statusCode)
	
	// Get error message
	errorMessage := getErrorMessage(c, handlerErr, statusCode)
	
	// Get request body (sanitized)
	requestBody := sanitizeRequestBody(c)
	
	// Get query params
	queryParams := c.Request().URI().QueryString()
	
	// Get user info from context
	userID := ""
	userEmail := ""
	if uid := c.Locals("userId"); uid != nil {
		userID = uid.(string)
	}
	if email := c.Locals("userEmail"); email != nil {
		userEmail = email.(string)
	}
	
	// Create error log
	errorLog := &models.ErrorLog{
		Timestamp:    time.Now(),
		Level:        level,
		Feature:      feature,
		Endpoint:     path,
		Method:       method,
		Action:       action,
		ErrorCode:    fmt.Sprintf("HTTP_%d", statusCode),
		ErrorMessage: errorMessage,
		RequestBody:  requestBody,
		QueryParams:  string(queryParams),
		UserID:       userID,
		UserEmail:    userEmail,
		IPAddress:    c.IP(),
		UserAgent:    c.Get("User-Agent"),
		StatusCode:   statusCode,
		Duration:     duration,
	}
	
	// Log to database
	if err := service.LogError(errorLog); err != nil {
		fmt.Printf("Failed to log error to database: %v\n", err)
	}
}

func getActionFromMethod(method string) string {
	switch method {
	case "GET":
		return "read"
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return method
	}
}

func getErrorLevel(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "CRITICAL"
	case statusCode >= 400 && statusCode < 500:
		return "ERROR"
	default:
		return "WARN"
	}
}

func getErrorMessage(c *fiber.Ctx, handlerErr error, statusCode int) string {
	// Try to get error from response body
	body := c.Response().Body()
	if len(body) > 0 {
		var errResp map[string]interface{}
		if err := json.Unmarshal(body, &errResp); err == nil {
			if msg, ok := errResp["error"].(string); ok {
				return msg
			}
			if msg, ok := errResp["message"].(string); ok {
				return msg
			}
		}
		// If not JSON, return raw body (truncated)
		if len(body) > 500 {
			return string(body[:500]) + "..."
		}
		return string(body)
	}
	
	// Try to get error from handler error
	if handlerErr != nil {
		return handlerErr.Error()
	}
	
	// Default message based on status code
	return fmt.Sprintf("HTTP Error %d", statusCode)
}

func sanitizeRequestBody(c *fiber.Ctx) string {
	body := c.Body()
	if len(body) == 0 {
		return ""
	}
	
	// Parse JSON to sanitize sensitive fields
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// Not JSON, return truncated
		if len(body) > 1000 {
			return string(body[:1000]) + "..."
		}
		return string(body)
	}
	
	// Sanitize sensitive fields
	sensitiveFields := []string{"password", "senha", "token", "secret", "key", "apiKey", "api_key"}
	sanitizeMap(data, sensitiveFields)
	
	// Convert back to JSON
	sanitized, err := json.Marshal(data)
	if err != nil {
		return "[failed to sanitize]"
	}
	
	// Truncate if too long
	if len(sanitized) > 2000 {
		return string(sanitized[:2000]) + "..."
	}
	
	return string(sanitized)
}

func sanitizeMap(data map[string]interface{}, sensitiveFields []string) {
	for key, value := range data {
		// Check if key is sensitive
		keyLower := strings.ToLower(key)
		for _, sensitive := range sensitiveFields {
			if strings.Contains(keyLower, sensitive) {
				data[key] = "[REDACTED]"
				break
			}
		}
		
		// Recursively sanitize nested maps
		if nested, ok := value.(map[string]interface{}); ok {
			sanitizeMap(nested, sensitiveFields)
		}
	}
}

// PanicRecoveryMiddleware recovers from panics and logs them
func PanicRecoveryMiddleware(errorLogService *services.ErrorLogService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				// Get stack trace
				stack := debug.Stack()
				
				method := c.Method()
				path := c.Path()
				feature := models.GetFeatureName(method, path)
				
				// Get user info
				userID := ""
				userEmail := ""
				if uid := c.Locals("userId"); uid != nil {
					userID = uid.(string)
				}
				if email := c.Locals("userEmail"); email != nil {
					userEmail = email.(string)
				}
				
				// Create error log for panic
				errorLog := &models.ErrorLog{
					Timestamp:    time.Now(),
					Level:        "CRITICAL",
					Feature:      feature,
					Endpoint:     path,
					Method:       method,
					Action:       "panic",
					ErrorCode:    "PANIC",
					ErrorMessage: fmt.Sprintf("Panic: %v", r),
					StackTrace:   string(stack),
					RequestBody:  sanitizeRequestBody(c),
					QueryParams:  string(c.Request().URI().QueryString()),
					UserID:       userID,
					UserEmail:    userEmail,
					IPAddress:    c.IP(),
					UserAgent:    c.Get("User-Agent"),
					StatusCode:   500,
				}
				
				// Log to database
				if err := errorLogService.LogError(errorLog); err != nil {
					fmt.Printf("Failed to log panic to database: %v\n", err)
				}
				
				// Return 500 error
				c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		}()
		
		return c.Next()
	}
}

// RequestBodyBuffer middleware to allow reading body multiple times
func RequestBodyBuffer() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Store original body for later use
		body := c.Body()
		c.Locals("requestBody", bytes.Clone(body))
		return c.Next()
	}
}
