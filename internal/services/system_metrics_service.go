package services

import (
	"database/sql"
	"runtime"
	"time"

	"github.com/shigake/tech-iq-back/internal/cache"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"gorm.io/gorm"
)

var (
	startTime    = time.Now()
	serverVersion = "1.0.0"
)

type SystemMetricsService interface {
	GetMetrics() (*models.SystemMetrics, error)
}

type systemMetricsService struct {
	db              *gorm.DB
	redisClient     *cache.RedisClient
	userRepo        repositories.UserRepository
	ticketRepo      repositories.TicketRepository
	securityLogRepo repositories.SecurityLogRepository
}

func NewSystemMetricsService(
	db *gorm.DB,
	redisClient *cache.RedisClient,
	userRepo repositories.UserRepository,
	ticketRepo repositories.TicketRepository,
	securityLogRepo repositories.SecurityLogRepository,
) SystemMetricsService {
	return &systemMetricsService{
		db:              db,
		redisClient:     redisClient,
		userRepo:        userRepo,
		ticketRepo:      ticketRepo,
		securityLogRepo: securityLogRepo,
	}
}

func (s *systemMetricsService) GetMetrics() (*models.SystemMetrics, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get DB connection stats
	sqlDB, err := s.db.DB()
	var dbConns, dbMaxConns int
	if err == nil {
		stats := sqlDB.Stats()
		dbConns = stats.InUse
		dbMaxConns = stats.MaxOpenConnections
	}

	// Calculate uptime
	uptime := int64(time.Since(startTime).Seconds())

	// Get business metrics
	activeUsers := s.getActiveUsers(sqlDB)
	todayLogins := s.getTodayLogins()
	todayTickets := s.getTodayTickets()
	openTickets := s.getOpenTickets()

	// Get request metrics (last 24h)
	totalRequests, avgResponseTime, errorRate := s.getRequestMetrics(sqlDB)

	metrics := &models.SystemMetrics{
		// Server info
		ServerUptime:  uptime,
		ServerVersion: serverVersion,
		GoVersion:     runtime.Version(),

		// Memory metrics
		MemoryUsed:    memStats.Alloc,
		MemoryTotal:   memStats.Sys,
		MemoryPercent: float64(memStats.Alloc) / float64(memStats.Sys) * 100,

		// CPU metrics (goroutines as proxy for load)
		CPUUsage:      float64(runtime.NumGoroutine()) / 100.0, // Rough estimate
		NumGoroutines: runtime.NumGoroutine(),

		// Database metrics
		DBConnections: dbConns,
		DBMaxConn:     dbMaxConns,

		// Request metrics
		TotalRequests:   totalRequests,
		AvgResponseTime: avgResponseTime,
		ErrorRate:       errorRate,

		// Cache metrics
		CacheHitRate: s.getCacheHitRate(),
		CacheSize:    s.getCacheSize(),

		// Business metrics
		ActiveUsers:  activeUsers,
		TodayLogins:  todayLogins,
		TodayTickets: todayTickets,
		OpenTickets:  openTickets,

		Timestamp: time.Now(),
	}

	return metrics, nil
}

func (s *systemMetricsService) getActiveUsers(sqlDB *sql.DB) int64 {
	var count int64
	since := time.Now().Add(-24 * time.Hour)
	s.db.Model(&models.User{}).Where("last_login_at >= ?", since).Count(&count)
	return count
}

func (s *systemMetricsService) getTodayLogins() int64 {
	var count int64
	today := time.Now().Truncate(24 * time.Hour)
	s.db.Model(&models.SecurityLog{}).
		Where("action = ?", "login_success").
		Where("created_at >= ?", today).
		Count(&count)
	return count
}

func (s *systemMetricsService) getTodayTickets() int64 {
	var count int64
	today := time.Now().Truncate(24 * time.Hour)
	s.db.Model(&models.Ticket{}).Where("created_at >= ?", today).Count(&count)
	return count
}

func (s *systemMetricsService) getOpenTickets() int64 {
	var count int64
	s.db.Model(&models.Ticket{}).
		Where("status IN ?", []string{"OPEN", "IN_PROGRESS", "PENDING"}).
		Count(&count)
	return count
}

func (s *systemMetricsService) getRequestMetrics(sqlDB *sql.DB) (int64, float64, float64) {
	var totalRequests int64
	var avgResponseTime float64
	var errorCount int64

	since := time.Now().Add(-24 * time.Hour)

	// Total requests
	s.db.Model(&models.RequestMetric{}).Where("created_at >= ?", since).Count(&totalRequests)

	// Average response time
	s.db.Model(&models.RequestMetric{}).
		Where("created_at >= ?", since).
		Select("COALESCE(AVG(response_time), 0)").
		Row().Scan(&avgResponseTime)

	// Error count (5xx status codes)
	s.db.Model(&models.RequestMetric{}).
		Where("created_at >= ?", since).
		Where("status_code >= 500").
		Count(&errorCount)

	var errorRate float64
	if totalRequests > 0 {
		errorRate = float64(errorCount) / float64(totalRequests) * 100
	}

	return totalRequests, avgResponseTime, errorRate
}

func (s *systemMetricsService) getCacheHitRate() float64 {
	if s.redisClient == nil {
		return 0
	}
	_, hitRate, err := s.redisClient.GetStats()
	if err != nil {
		return 0
	}
	return hitRate
}

func (s *systemMetricsService) getCacheSize() int64 {
	if s.redisClient == nil {
		return 0
	}
	dbSize, _, err := s.redisClient.GetStats()
	if err != nil {
		return 0
	}
	return dbSize
}
