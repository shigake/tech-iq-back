package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrorLog represents an error log entry in the database
type ErrorLog struct {
	ID           string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Timestamp    time.Time `json:"timestamp" gorm:"index;not null"`
	Level        string    `json:"level" gorm:"type:varchar(20);index;not null"` // ERROR, WARN, CRITICAL
	Feature      string    `json:"feature" gorm:"type:varchar(100);index"`       // Nome da funcionalidade: "Criar Técnico", "Login", etc
	Endpoint     string    `json:"endpoint" gorm:"type:varchar(255);index"`      // /api/v1/technicians
	Method       string    `json:"method" gorm:"type:varchar(10)"`               // GET, POST, PUT, DELETE
	Action       string    `json:"action" gorm:"type:varchar(100)"`              // create, update, delete, list, etc
	ErrorCode    string    `json:"errorCode" gorm:"type:varchar(50);index"`      // Código do erro se houver
	ErrorMessage string    `json:"errorMessage" gorm:"type:text;not null"`       // Mensagem do erro
	StackTrace   string    `json:"stackTrace" gorm:"type:text"`                  // Stack trace se disponível
	RequestBody  string    `json:"requestBody" gorm:"type:text"`                 // Body da request (sanitizado)
	QueryParams  string    `json:"queryParams" gorm:"type:text"`                 // Query parameters
	UserID       string    `json:"userId" gorm:"type:varchar(36);index"`         // ID do usuário se autenticado
	UserEmail    string    `json:"userEmail" gorm:"type:varchar(255)"`           // Email do usuário
	IPAddress    string    `json:"ipAddress" gorm:"type:varchar(45)"`            // IP do cliente
	UserAgent    string    `json:"userAgent" gorm:"type:text"`                   // Browser/device info
	StatusCode   int       `json:"statusCode" gorm:"index"`                      // HTTP status code retornado
	Duration     int64     `json:"duration"`                                     // Duração da request em ms
	Resolved     bool      `json:"resolved" gorm:"default:false;index"`          // Se o erro foi resolvido
	ResolvedAt   *time.Time `json:"resolvedAt"`                                  // Quando foi resolvido
	ResolvedBy   string    `json:"resolvedBy" gorm:"type:varchar(36)"`           // Quem resolveu
	Notes        string    `json:"notes" gorm:"type:text"`                       // Notas sobre a resolução
	CreatedAt    time.Time `json:"createdAt"`
}

func (e *ErrorLog) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	return nil
}

// ErrorLogFilter represents filters for querying error logs
type ErrorLogFilter struct {
	Level      string    `json:"level"`
	Feature    string    `json:"feature"`
	Endpoint   string    `json:"endpoint"`
	ErrorCode  string    `json:"errorCode"`
	UserID     string    `json:"userId"`
	StatusCode int       `json:"statusCode"`
	Resolved   *bool     `json:"resolved"`
	StartDate  time.Time `json:"startDate"`
	EndDate    time.Time `json:"endDate"`
	Search     string    `json:"search"`
}

// PaginatedErrorLogs represents a paginated list of error logs
type PaginatedErrorLogs struct {
	Data       []ErrorLog `json:"data"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	TotalPages int        `json:"totalPages"`
}

// ErrorLogStats represents statistics about error logs
type ErrorLogStats struct {
	TotalErrors      int64            `json:"totalErrors"`
	UnresolvedErrors int64            `json:"unresolvedErrors"`
	ErrorsByLevel    map[string]int64 `json:"errorsByLevel"`
	ErrorsByFeature  map[string]int64 `json:"errorsByFeature"`
	ErrorsByEndpoint map[string]int64 `json:"errorsByEndpoint"`
	ErrorsToday      int64            `json:"errorsToday"`
	ErrorsThisWeek   int64            `json:"errorsThisWeek"`
}

// ResolveErrorRequest represents the request to resolve an error
type ResolveErrorRequest struct {
	Notes string `json:"notes"`
}

// FeatureMapping maps endpoints to human-readable feature names
var FeatureMapping = map[string]string{
	// Auth
	"POST /api/v1/auth/signin":     "Login",
	"POST /api/v1/auth/signup":     "Cadastro de Usuário",
	"POST /api/v1/auth/refresh":    "Refresh Token",
	"POST /api/v1/auth/logout":     "Logout",
	
	// Technicians
	"GET /api/v1/technicians":      "Listar Técnicos",
	"POST /api/v1/technicians":     "Criar Técnico",
	"GET /api/v1/technicians/:id":  "Visualizar Técnico",
	"PUT /api/v1/technicians/:id":  "Atualizar Técnico",
	"DELETE /api/v1/technicians/:id": "Excluir Técnico",
	"GET /api/v1/technicians/search": "Buscar Técnicos",
	
	// Tickets
	"GET /api/v1/tickets":          "Listar Tickets",
	"POST /api/v1/tickets":         "Criar Ticket",
	"GET /api/v1/tickets/:id":      "Visualizar Ticket",
	"PUT /api/v1/tickets/:id":      "Atualizar Ticket",
	"DELETE /api/v1/tickets/:id":   "Excluir Ticket",
	
	// Clients
	"GET /api/v1/clients":          "Listar Clientes",
	"POST /api/v1/clients":         "Criar Cliente",
	"GET /api/v1/clients/:id":      "Visualizar Cliente",
	"PUT /api/v1/clients/:id":      "Atualizar Cliente",
	"DELETE /api/v1/clients/:id":   "Excluir Cliente",
	
	// Categories
	"GET /api/v1/categories":       "Listar Categorias",
	"POST /api/v1/categories":      "Criar Categoria",
	"PUT /api/v1/categories/:id":   "Atualizar Categoria",
	"DELETE /api/v1/categories/:id": "Excluir Categoria",
	
	// Users
	"GET /api/v1/users":            "Listar Usuários",
	"POST /api/v1/users":           "Criar Usuário",
	"GET /api/v1/users/:id":        "Visualizar Usuário",
	"PUT /api/v1/users/:id":        "Atualizar Usuário",
	"DELETE /api/v1/users/:id":     "Excluir Usuário",
	
	// Financial
	"GET /api/v1/financial/entries":     "Listar Lançamentos",
	"POST /api/v1/financial/entries":    "Criar Lançamento",
	"GET /api/v1/financial/dashboard":   "Dashboard Financeiro",
	"GET /api/v1/financial/batches":     "Listar Lotes",
	"POST /api/v1/financial/batches":    "Criar Lote",
	
	// Stock
	"GET /api/v1/stock/items":           "Listar Itens de Estoque",
	"POST /api/v1/stock/items":          "Criar Item de Estoque",
	"GET /api/v1/stock/locations":       "Listar Locais de Estoque",
	"POST /api/v1/stock/movements":      "Registrar Movimentação",
	
	// Hierarchy
	"GET /api/v1/hierarchies":           "Listar Hierarquias",
	"POST /api/v1/hierarchies":          "Criar Hierarquia",
	"GET /api/v1/nodes/:id/members":     "Listar Membros do Nó",
	"POST /api/v1/nodes/:id/members":    "Adicionar Membro ao Nó",
	
	// Geo
	"GET /api/v1/geo/technicians":       "Mapa de Técnicos",
	"POST /api/v1/geo/location":         "Atualizar Localização",
	"GET /api/v1/geo/technician/:id/history": "Histórico de Localização",
	
	// Dashboard
	"GET /api/v1/dashboard/stats":       "Dashboard - Estatísticas",
	"GET /api/v1/dashboard/recent-activity": "Dashboard - Atividades Recentes",
}

// GetFeatureName returns the human-readable feature name for an endpoint
func GetFeatureName(method, path string) string {
	key := method + " " + path
	if feature, ok := FeatureMapping[key]; ok {
		return feature
	}
	
	// Try to find a pattern match
	for pattern, feature := range FeatureMapping {
		if matchesPattern(key, pattern) {
			return feature
		}
	}
	
	return path // Return the path if no mapping found
}

// matchesPattern checks if a path matches a pattern with :id placeholders
func matchesPattern(actual, pattern string) bool {
	actualParts := splitPath(actual)
	patternParts := splitPath(pattern)
	
	if len(actualParts) != len(patternParts) {
		return false
	}
	
	for i, part := range patternParts {
		if part == ":id" || part == ":nodeId" || part == ":technicianId" {
			continue // Matches any value
		}
		if part != actualParts[i] {
			return false
		}
	}
	
	return true
}

func splitPath(s string) []string {
	var parts []string
	for _, p := range []byte(s) {
		if p == '/' || p == ' ' {
			continue
		}
	}
	// Simple split
	result := []string{}
	current := ""
	for _, c := range s {
		if c == '/' || c == ' ' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
