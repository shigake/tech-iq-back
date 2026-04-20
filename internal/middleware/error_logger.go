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

type loggedRequest struct {
	method       string
	path         string
	ip           string
	userAgent    string
	userID       string
	userEmail    string
	requestBody  string
	queryParams  string
	responseBody []byte
	handlerErr   error
	statusCode   int
	duration     int64
}

func ErrorLoggerMiddleware(errorLogService *services.ErrorLogService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start).Milliseconds()
		statusCode := c.Response().StatusCode()

		if statusCode < 400 {
			return err
		}

		snapshot := snapshotRequest(c, statusCode, duration, err)
		go logError(errorLogService, snapshot)

		return err
	}
}

func snapshotRequest(c *fiber.Ctx, statusCode int, duration int64, handlerErr error) loggedRequest {
	respBody := c.Response().Body()
	respCopy := make([]byte, len(respBody))
	copy(respCopy, respBody)

	return loggedRequest{
		method:       c.Method(),
		path:         c.Path(),
		ip:           c.IP(),
		userAgent:    c.Get("User-Agent"),
		userID:       localsString(c, "userId"),
		userEmail:    localsString(c, "email"),
		requestBody:  sanitizeRequestBody(c.Body()),
		queryParams:  string(c.Request().URI().QueryString()),
		responseBody: respCopy,
		handlerErr:   handlerErr,
		statusCode:   statusCode,
		duration:     duration,
	}
}

func localsString(c *fiber.Ctx, key string) string {
	v := c.Locals(key)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func logError(service *services.ErrorLogService, r loggedRequest) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Printf("Error logging failed: %v\n", rec)
		}
	}()

	errorLog := &models.ErrorLog{
		Timestamp:    time.Now(),
		Level:        getErrorLevel(r.statusCode),
		Feature:      models.GetFeatureName(r.method, r.path),
		Endpoint:     r.path,
		Method:       r.method,
		Action:       getActionFromMethod(r.method),
		ErrorCode:    fmt.Sprintf("HTTP_%d", r.statusCode),
		ErrorMessage: getErrorMessage(r.responseBody, r.handlerErr, r.statusCode),
		RequestBody:  r.requestBody,
		QueryParams:  r.queryParams,
		UserID:       r.userID,
		UserEmail:    r.userEmail,
		IPAddress:    r.ip,
		UserAgent:    r.userAgent,
		StatusCode:   r.statusCode,
		Duration:     r.duration,
	}

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

func getErrorMessage(body []byte, handlerErr error, statusCode int) string {
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
		if len(body) > 500 {
			return string(body[:500]) + "..."
		}
		return string(body)
	}

	if handlerErr != nil {
		return handlerErr.Error()
	}

	return fmt.Sprintf("HTTP Error %d", statusCode)
}

func sanitizeRequestBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		if len(body) > 1000 {
			return string(body[:1000]) + "..."
		}
		return string(body)
	}

	sensitiveFields := []string{"password", "senha", "token", "secret", "key", "apiKey", "api_key"}
	sanitizeMap(data, sensitiveFields)

	sanitized, err := json.Marshal(data)
	if err != nil {
		return "[failed to sanitize]"
	}

	if len(sanitized) > 2000 {
		return string(sanitized[:2000]) + "..."
	}

	return string(sanitized)
}

func sanitizeMap(data map[string]interface{}, sensitiveFields []string) {
	for key, value := range data {
		keyLower := strings.ToLower(key)
		for _, sensitive := range sensitiveFields {
			if strings.Contains(keyLower, sensitive) {
				data[key] = "[REDACTED]"
				break
			}
		}

		if nested, ok := value.(map[string]interface{}); ok {
			sanitizeMap(nested, sensitiveFields)
		}
	}
}

func PanicRecoveryMiddleware(errorLogService *services.ErrorLogService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()

				method := c.Method()
				path := c.Path()
				feature := models.GetFeatureName(method, path)

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
					RequestBody:  sanitizeRequestBody(c.Body()),
					QueryParams:  string(c.Request().URI().QueryString()),
					UserID:       localsString(c, "userId"),
					UserEmail:    localsString(c, "userEmail"),
					IPAddress:    c.IP(),
					UserAgent:    c.Get("User-Agent"),
					StatusCode:   500,
				}

				if err := errorLogService.LogError(errorLog); err != nil {
					fmt.Printf("Failed to log panic to database: %v\n", err)
				}

				c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		}()

		return c.Next()
	}
}

func RequestBodyBuffer() fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := c.Body()
		c.Locals("requestBody", bytes.Clone(body))
		return c.Next()
	}
}
