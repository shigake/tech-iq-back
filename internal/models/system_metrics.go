package models

import "time"

// SystemMetrics represents current system metrics
type SystemMetrics struct {
	// Server info
	ServerUptime    int64   `json:"serverUptime"`    // Uptime in seconds
	ServerVersion   string  `json:"serverVersion"`
	GoVersion       string  `json:"goVersion"`
	
	// Memory metrics
	MemoryUsed      uint64  `json:"memoryUsed"`      // Bytes
	MemoryTotal     uint64  `json:"memoryTotal"`     // Bytes
	MemoryPercent   float64 `json:"memoryPercent"`
	
	// CPU metrics
	CPUUsage        float64 `json:"cpuUsage"`        // Percentage
	NumGoroutines   int     `json:"numGoroutines"`
	
	// Database metrics
	DBConnections   int     `json:"dbConnections"`
	DBMaxConn       int     `json:"dbMaxConnections"`
	
	// Request metrics (last 24h)
	TotalRequests   int64   `json:"totalRequests"`
	AvgResponseTime float64 `json:"avgResponseTime"` // Milliseconds
	ErrorRate       float64 `json:"errorRate"`       // Percentage
	
	// Cache metrics
	CacheHitRate    float64 `json:"cacheHitRate"`    // Percentage
	CacheSize       int64   `json:"cacheSize"`       // Number of items
	
	// Business metrics
	ActiveUsers     int64   `json:"activeUsers"`     // Users active in last 24h
	TodayLogins     int64   `json:"todayLogins"`
	TodayTickets    int64   `json:"todayTickets"`
	OpenTickets     int64   `json:"openTickets"`
	
	Timestamp       time.Time `json:"timestamp"`
}

// RequestMetric stores individual request metrics for aggregation
type RequestMetric struct {
	ID           string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Path         string    `json:"path" gorm:"type:varchar(255);index"`
	Method       string    `json:"method" gorm:"type:varchar(10)"`
	StatusCode   int       `json:"statusCode" gorm:"index"`
	ResponseTime float64   `json:"responseTime"` // Milliseconds
	UserID       string    `json:"userId" gorm:"type:varchar(36);index"`
	IPAddress    string    `json:"ipAddress" gorm:"type:varchar(45)"`
	CreatedAt    time.Time `json:"createdAt" gorm:"index"`
}
